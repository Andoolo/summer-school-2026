package com.volna.app.auth.presentation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.volna.app.auth.AuthRepository
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.asAppFailure
import com.volna.app.core.logging.AppLogger
import com.volna.app.core.mvi.MviStore
import com.volna.app.core.phone.isRussianPhoneInputComplete
import com.volna.app.core.phone.normalizePhoneE164
import com.volna.app.core.phone.sanitizePhoneInput
import com.volna.app.core.ui.ActionStatus
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.Phone
import com.volna.app.profile.ProfileRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

enum class AuthStep {
    Phone,
    Otp,
    Name,
}

data class AuthState(
    val step: AuthStep = AuthStep.Phone,
    val phoneInput: String = "",
    val codeInput: String = "",
    val nameInput: String = "",
    val ttlSeconds: Int? = null,
    val resendAfterSeconds: Int? = null,
    val resendSecondsRemaining: Int = 0,
    val client: Client? = null,
    val actionStatus: ActionStatus = ActionStatus.Idle,
    val fieldError: String? = null,
    val message: String? = null,
) {
    val isSubmitting: Boolean = actionStatus == ActionStatus.Submitting
    val isPhoneValid: Boolean = phoneInput.isRussianPhoneInputComplete()
    val isCodeValid: Boolean = codeInput.matches(Regex("^\\d{4,6}$"))
    val isNameValid: Boolean = nameInput.trim().length in 1..100
    val canRequestCode: Boolean = !isSubmitting && isPhoneValid && resendSecondsRemaining == 0
    val canVerifyCode: Boolean = !isSubmitting && isCodeValid
    val canResendCode: Boolean = !isSubmitting && resendSecondsRemaining == 0
    val canContinueName: Boolean = !isSubmitting && isNameValid
}

sealed interface AuthIntent {
    data class PhoneChanged(val value: String) : AuthIntent
    data object RequestCode : AuthIntent
    data class CodeChanged(val value: String) : AuthIntent
    data object VerifyCode : AuthIntent
    data object ResendCode : AuthIntent
    data object BackToPhone : AuthIntent
    data class NameChanged(val value: String) : AuthIntent
    data object ContinueWithName : AuthIntent
    data object MessageShown : AuthIntent
    data object Reset : AuthIntent
}

sealed interface AuthEffect {
    data object Authenticated : AuthEffect
}

class AuthStore(
    private val authRepository: AuthRepository,
    private val profileRepository: ProfileRepository,
    scope: CoroutineScope? = null,
) : ViewModel(), MviStore<AuthState, AuthIntent, AuthEffect> {
    private val mutableState = MutableStateFlow(AuthState())
    private val effects = Channel<AuthEffect>(Channel.BUFFERED)
    private val storeScope = scope ?: viewModelScope
    private var resendTimer: Job? = null

    override val state: StateFlow<AuthState> = mutableState

    override fun accept(intent: AuthIntent) {
        when (intent) {
            is AuthIntent.PhoneChanged -> onPhoneChanged(intent.value)
            AuthIntent.RequestCode -> requestCode()
            is AuthIntent.CodeChanged -> onCodeChanged(intent.value)
            AuthIntent.VerifyCode -> verifyCode()
            AuthIntent.ResendCode -> resendCode()
            AuthIntent.BackToPhone -> backToPhone()
            is AuthIntent.NameChanged -> onNameChanged(intent.value)
            AuthIntent.ContinueWithName -> continueWithName()
            AuthIntent.MessageShown -> mutableState.update { it.copy(message = null) }
            AuthIntent.Reset -> reset()
        }
    }

    override suspend fun effects(): AuthEffect = effects.receive()

    private fun onPhoneChanged(value: String) {
        mutableState.update {
            it.copy(
                phoneInput = sanitizePhoneInput(value),
                fieldError = null,
                message = null,
            )
        }
    }

    private fun onCodeChanged(value: String) {
        mutableState.update {
            it.copy(
                codeInput = value.filter(Char::isDigit).take(6),
                fieldError = null,
                message = null,
            )
        }
    }

    private fun onNameChanged(value: String) {
        mutableState.update {
            it.copy(
                nameInput = value.take(100),
                fieldError = null,
                message = null,
            )
        }
    }

    private fun requestCode() {
        val phoneValue = normalizePhoneE164(mutableState.value.phoneInput)
        if (!mutableState.value.isPhoneValid) {
            mutableState.update {
                it.copy(fieldError = "Похоже, номер введён не полностью")
            }
            return
        }
        if (mutableState.value.isSubmitting) return

        storeScope.launch {
            mutableState.update { it.copy(actionStatus = ActionStatus.Submitting, message = null) }
            val result = authRepository.requestCode(Phone(phoneValue))
            result.fold(
                onSuccess = { response ->
                    startResendTimer(response.resendAfterSeconds)
                    mutableState.update {
                        it.copy(
                            step = AuthStep.Otp,
                            ttlSeconds = response.ttlSeconds,
                            resendAfterSeconds = response.resendAfterSeconds,
                            codeInput = "",
                            actionStatus = ActionStatus.Idle,
                            fieldError = null,
                        )
                    }
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to request auth code")
                    val appFailure = failure.asAppFailure()
                    if (appFailure.isTooManyRequests()) {
                        startResendTimer(mutableState.value.resendAfterSeconds ?: DEFAULT_RESEND_SECONDS)
                    }
                    mutableState.update {
                        it.copy(
                            actionStatus = ActionStatus.Idle,
                            message = requestCodeMessage(appFailure),
                        )
                    }
                },
            )
        }
    }

    private fun verifyCode() {
        val current = mutableState.value
        if (!current.isCodeValid) {
            mutableState.update { it.copy(fieldError = "Введите код из SMS") }
            return
        }
        if (current.isSubmitting) return

        storeScope.launch {
            mutableState.update { it.copy(actionStatus = ActionStatus.Submitting, message = null) }
            val result = authRepository.verifyCode(Phone(normalizePhoneE164(current.phoneInput)), current.codeInput)
            result.fold(
                onSuccess = { response ->
                    resendTimer?.cancel()
                    if (response.isNew) {
                        mutableState.update {
                            it.copy(
                                step = AuthStep.Name,
                                client = response.client,
                                actionStatus = ActionStatus.Idle,
                                fieldError = null,
                                message = null,
                            )
                        }
                    } else {
                        mutableState.update { it.copy(actionStatus = ActionStatus.Idle) }
                        effects.send(AuthEffect.Authenticated)
                    }
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to verify auth code")
                    val appFailure = failure.asAppFailure()
                    if (appFailure.isTooManyRequests()) {
                        startResendTimer(current.resendAfterSeconds ?: DEFAULT_RESEND_SECONDS)
                    }
                    mutableState.update {
                        it.copy(
                            actionStatus = ActionStatus.Idle,
                            message = verifyCodeMessage(appFailure),
                        )
                    }
                },
            )
        }
    }

    private fun resendCode() {
        if (!mutableState.value.canResendCode) return
        requestCode()
    }

    private fun backToPhone() {
        resendTimer?.cancel()
        mutableState.update {
            it.copy(
                step = AuthStep.Phone,
                codeInput = "",
                ttlSeconds = null,
                resendAfterSeconds = null,
                resendSecondsRemaining = 0,
                actionStatus = ActionStatus.Idle,
                fieldError = null,
                message = null,
            )
        }
    }

    private fun continueWithName() {
        val name = mutableState.value.nameInput.trim()
        if (name.isEmpty()) {
            mutableState.update { it.copy(fieldError = "Укажите, как к вам обращаться") }
            return
        }
        if (mutableState.value.isSubmitting) return

        storeScope.launch {
            mutableState.update { it.copy(actionStatus = ActionStatus.Submitting, fieldError = null, message = null) }
            val result = profileRepository.updateName(name)
            result.fold(
                onSuccess = { client ->
                    mutableState.update {
                        it.copy(
                            client = client,
                            actionStatus = ActionStatus.Idle,
                            fieldError = null,
                            message = null,
                        )
                    }
                    effects.send(AuthEffect.Authenticated)
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to update profile name during auth")
                    val appFailure = failure.asAppFailure()
                    if (appFailure == AppFailure.Unauthorized) {
                        resetToPhoneAfterUnauthorized()
                    } else {
                        mutableState.update {
                            it.copy(
                                actionStatus = ActionStatus.Idle,
                                fieldError = nameStepFieldError(appFailure),
                                message = nameStepMessage(appFailure),
                            )
                        }
                    }
                },
            )
        }
    }

    private fun resetToPhoneAfterUnauthorized() {
        resendTimer?.cancel()
        mutableState.update {
            AuthState(
                phoneInput = it.phoneInput,
                message = "Сессия истекла, войдите снова",
            )
        }
    }

    private fun reset() {
        resendTimer?.cancel()
        mutableState.value = AuthState()
    }

    private fun startResendTimer(seconds: Int) {
        resendTimer?.cancel()
        mutableState.update { it.copy(resendSecondsRemaining = seconds.coerceAtLeast(0)) }
        resendTimer = storeScope.launch {
            while (mutableState.value.resendSecondsRemaining > 0) {
                delay(1_000)
                mutableState.update { state ->
                    state.copy(resendSecondsRemaining = (state.resendSecondsRemaining - 1).coerceAtLeast(0))
                }
            }
        }
    }

    private fun requestCodeMessage(failure: AppFailure): String = when {
        failure.isTooManyRequests() -> "Повторная отправка будет доступна после таймера"
        failure == AppFailure.NetworkUnavailable -> "Не удалось загрузить. Проверьте соединение и попробуйте снова"
        else -> "Не удалось войти. Попробуйте ещё раз"
    }

    private fun verifyCodeMessage(failure: AppFailure): String = when {
        failure.isTooManyRequests() -> "Слишком много попыток. Запросите новый код"
        failure.isInvalidCode() -> "Код неверен или просрочен. Запросите новый код"
        failure == AppFailure.NetworkUnavailable -> "Не удалось загрузить. Проверьте соединение и попробуйте снова"
        else -> "Произошла ошибка. Попробуйте позже"
    }

    private fun nameStepFieldError(failure: AppFailure): String? = when {
        failure.isBadRequest() -> "Проверьте имя — кажется, тут лишние символы"
        else -> null
    }

    private fun nameStepMessage(failure: AppFailure): String? = when {
        failure.isBadRequest() -> null
        failure == AppFailure.NetworkUnavailable -> "Не удалось загрузить. Проверьте соединение и попробуйте снова"
        else -> "Произошла ошибка. Попробуйте позже"
    }

    private fun AppFailure.isInvalidCode(): Boolean =
        this is AppFailure.Api && code == ApiErrorCode.InvalidCode

    private fun AppFailure.isTooManyRequests(): Boolean =
        this is AppFailure.Api && code == ApiErrorCode.TooManyRequests

    private fun AppFailure.isBadRequest(): Boolean =
        this is AppFailure.Api && code == ApiErrorCode.BadRequest

    private companion object {
        const val DEFAULT_RESEND_SECONDS = 60
    }
}
