package com.volna.app.profile.presentation

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
import com.volna.app.core.ui.Loadable
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

enum class ProfileMode {
    View,
    Edit,
    ConfirmPhone,
}

data class ProfileState(
    val profile: Loadable<Client> = Loadable.Initial,
    val mode: ProfileMode = ProfileMode.View,
    val nameInput: String = "",
    val phoneInput: String = "",
    val codeInput: String = "",
    val pendingPhone: String? = null,
    val ttlSeconds: Int? = null,
    val resendAfterSeconds: Int? = null,
    val resendSecondsRemaining: Int = 0,
    val actionStatus: ActionStatus = ActionStatus.Idle,
    val logoutConfirmVisible: Boolean = false,
    val deleteConfirmVisible: Boolean = false,
    val fieldError: String? = null,
    val message: String? = null,
) {
    val isSubmitting: Boolean = actionStatus == ActionStatus.Submitting
    val isNameValid: Boolean = nameInput.trim().length in 1..100
    val isPhoneValid: Boolean = phoneInput.isRussianPhoneInputComplete()
    val isCodeValid: Boolean = codeInput.matches(Regex("^\\d{4,6}$"))
    val canSave: Boolean = !isSubmitting && isNameValid && isPhoneValid
    val canConfirmPhone: Boolean = !isSubmitting && isCodeValid
    val canResendCode: Boolean = !isSubmitting && resendSecondsRemaining == 0
}

sealed interface ProfileIntent {
    data object Load : ProfileIntent
    data object EditClicked : ProfileIntent
    data object EditCancelled : ProfileIntent
    data class NameChanged(val value: String) : ProfileIntent
    data class PhoneChanged(val value: String) : ProfileIntent
    data object SaveClicked : ProfileIntent
    data class CodeChanged(val value: String) : ProfileIntent
    data object ConfirmPhoneClicked : ProfileIntent
    data object ResendPhoneCode : ProfileIntent
    data object BackToEdit : ProfileIntent
    data object LogoutClicked : ProfileIntent
    data object LogoutDismissed : ProfileIntent
    data object LogoutConfirmed : ProfileIntent
    data object DeleteClicked : ProfileIntent
    data object DeleteDismissed : ProfileIntent
    data object DeleteConfirmed : ProfileIntent
    data object MessageShown : ProfileIntent
    data object Reset : ProfileIntent
}

sealed interface ProfileEffect {
    data object SignedOut : ProfileEffect
}

class ProfileStore(
    private val profileRepository: ProfileRepository,
    private val authRepository: AuthRepository,
    scope: CoroutineScope? = null,
) : ViewModel(), MviStore<ProfileState, ProfileIntent, ProfileEffect> {
    private val mutableState = MutableStateFlow(ProfileState())
    private val effects = Channel<ProfileEffect>(Channel.BUFFERED)
    private val storeScope = scope ?: viewModelScope
    private var resendTimer: Job? = null

    override val state: StateFlow<ProfileState> = mutableState

    override fun accept(intent: ProfileIntent) {
        when (intent) {
            ProfileIntent.Load -> loadProfile()
            ProfileIntent.EditClicked -> startEdit()
            ProfileIntent.EditCancelled -> cancelEdit()
            is ProfileIntent.NameChanged -> onNameChanged(intent.value)
            is ProfileIntent.PhoneChanged -> onPhoneChanged(intent.value)
            ProfileIntent.SaveClicked -> saveProfile()
            is ProfileIntent.CodeChanged -> onCodeChanged(intent.value)
            ProfileIntent.ConfirmPhoneClicked -> confirmPhone()
            ProfileIntent.ResendPhoneCode -> resendPhoneCode()
            ProfileIntent.BackToEdit -> backToEdit()
            ProfileIntent.LogoutClicked -> mutableState.update { it.copy(logoutConfirmVisible = true) }
            ProfileIntent.LogoutDismissed -> mutableState.update { it.copy(logoutConfirmVisible = false) }
            ProfileIntent.LogoutConfirmed -> logout()
            ProfileIntent.DeleteClicked -> mutableState.update { it.copy(deleteConfirmVisible = true) }
            ProfileIntent.DeleteDismissed -> mutableState.update { it.copy(deleteConfirmVisible = false) }
            ProfileIntent.DeleteConfirmed -> deleteAccount()
            ProfileIntent.MessageShown -> mutableState.update { it.copy(message = null) }
            ProfileIntent.Reset -> reset()
        }
    }

    override suspend fun effects(): ProfileEffect = effects.receive()

    private fun loadProfile() {
        if (mutableState.value.profile == Loadable.Loading) return

        storeScope.launch {
            mutableState.update { it.copy(profile = Loadable.Loading, message = null) }
            profileRepository.getProfile().fold(
                onSuccess = { client ->
                    mutableState.update {
                        it.copy(
                            profile = Loadable.Content(client),
                            nameInput = client.name.orEmpty(),
                            phoneInput = sanitizePhoneInput(client.phone.value),
                        )
                    }
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to load profile")
                    val appFailure = failure.asAppFailure()
                    if (appFailure == AppFailure.Unauthorized) {
                        effects.send(ProfileEffect.SignedOut)
                    } else {
                        mutableState.update { it.copy(profile = Loadable.Error(appFailure)) }
                    }
                },
            )
        }
    }

    private fun startEdit() {
        val client = (mutableState.value.profile as? Loadable.Content)?.value ?: return
        mutableState.update {
            it.copy(
                mode = ProfileMode.Edit,
                nameInput = client.name.orEmpty(),
                phoneInput = sanitizePhoneInput(client.phone.value),
                codeInput = "",
                pendingPhone = null,
                fieldError = null,
                message = null,
            )
        }
    }

    private fun cancelEdit() {
        resendTimer?.cancel()
        val client = (mutableState.value.profile as? Loadable.Content)?.value
        mutableState.update {
            it.copy(
                mode = ProfileMode.View,
                nameInput = client?.name.orEmpty(),
                phoneInput = sanitizePhoneInput(client?.phone?.value.orEmpty()),
                codeInput = "",
                pendingPhone = null,
                ttlSeconds = null,
                resendAfterSeconds = null,
                resendSecondsRemaining = 0,
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

    private fun saveProfile() {
        val current = mutableState.value
        val currentClient = (current.profile as? Loadable.Content)?.value ?: return
        val name = current.nameInput.trim()
        val phone = normalizePhoneE164(current.phoneInput)
        if (name.length !in 1..100) {
            mutableState.update { it.copy(fieldError = "Проверьте имя — кажется, тут лишние символы") }
            return
        }
        if (!current.isPhoneValid) {
            mutableState.update { it.copy(fieldError = "Похоже, номер введён не полностью") }
            return
        }
        if (current.isSubmitting) return

        val currentName = currentClient.name.orEmpty()
        if (phone == currentClient.phone.value) {
            updateName(name)
        } else if (name != currentName) {
            updateNameBeforePhoneChange(name, phone)
        } else {
            requestPhoneChangeCode(phone)
        }
    }

    private fun updateName(name: String) {
        storeScope.launch {
            mutableState.update {
                it.copy(actionStatus = ActionStatus.Submitting, fieldError = null, message = null)
            }
            profileRepository.updateName(name).fold(
                onSuccess = { client ->
                    mutableState.update {
                        it.copy(
                            profile = Loadable.Content(client),
                            mode = ProfileMode.View,
                            nameInput = client.name.orEmpty(),
                            phoneInput = sanitizePhoneInput(client.phone.value),
                            actionStatus = ActionStatus.Idle,
                            message = "Профиль обновлён",
                        )
                    }
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to update profile name")
                    handleActionFailure(failure.asAppFailure(), ::profileUpdateFieldError, ::profileUpdateMessage)
                },
            )
        }
    }

    private fun updateNameBeforePhoneChange(name: String, phone: String) {
        storeScope.launch {
            mutableState.update {
                it.copy(actionStatus = ActionStatus.Submitting, fieldError = null, message = null)
            }
            profileRepository.updateName(name).fold(
                onSuccess = { client ->
                    mutableState.update {
                        it.copy(
                            profile = Loadable.Content(client),
                            nameInput = client.name.orEmpty(),
                            actionStatus = ActionStatus.Idle,
                        )
                    }
                    requestPhoneChangeCode(phone)
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to update profile name before phone change")
                    handleActionFailure(failure.asAppFailure(), ::profileUpdateFieldError, ::profileUpdateMessage)
                },
            )
        }
    }

    private fun requestPhoneChangeCode(phone: String) {
        storeScope.launch {
            mutableState.update {
                it.copy(actionStatus = ActionStatus.Submitting, fieldError = null, message = null)
            }
            profileRepository.requestPhoneChangeCode(Phone(phone)).fold(
                onSuccess = { response ->
                    startResendTimer(response.resendAfterSeconds)
                    mutableState.update {
                        it.copy(
                            mode = ProfileMode.ConfirmPhone,
                            pendingPhone = phone,
                            ttlSeconds = response.ttlSeconds,
                            resendAfterSeconds = response.resendAfterSeconds,
                            codeInput = "",
                            actionStatus = ActionStatus.Idle,
                        )
                    }
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to request phone change code")
                    val appFailure = failure.asAppFailure()
                    if (appFailure.isTooManyRequests()) {
                        startResendTimer(mutableState.value.resendAfterSeconds ?: DEFAULT_RESEND_SECONDS)
                    }
                    handleActionFailure(appFailure, ::phoneChangeFieldError, ::phoneChangeMessage)
                },
            )
        }
    }

    private fun confirmPhone() {
        val current = mutableState.value
        val pendingPhone = current.pendingPhone ?: normalizePhoneE164(current.phoneInput)
        if (!current.isCodeValid) {
            mutableState.update { it.copy(fieldError = "Введите код из SMS") }
            return
        }
        if (!pendingPhone.isE164Phone() || current.isSubmitting) return

        storeScope.launch {
            mutableState.update {
                it.copy(actionStatus = ActionStatus.Submitting, fieldError = null, message = null)
            }
            profileRepository.confirmPhoneChange(Phone(pendingPhone), current.codeInput).fold(
                onSuccess = { client ->
                    resendTimer?.cancel()
                    mutableState.update {
                        it.copy(
                            profile = Loadable.Content(client),
                            mode = ProfileMode.View,
                            nameInput = client.name.orEmpty(),
                            phoneInput = sanitizePhoneInput(client.phone.value),
                            codeInput = "",
                            pendingPhone = null,
                            ttlSeconds = null,
                            resendAfterSeconds = null,
                            resendSecondsRemaining = 0,
                            actionStatus = ActionStatus.Idle,
                            message = "Изменения сохранены",
                        )
                    }
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to confirm phone change")
                    handleActionFailure(failure.asAppFailure(), ::confirmPhoneFieldError, ::confirmPhoneMessage)
                },
            )
        }
    }

    private fun resendPhoneCode() {
        val phone = mutableState.value.pendingPhone ?: normalizePhoneE164(mutableState.value.phoneInput)
        if (!mutableState.value.canResendCode || !phone.isE164Phone()) return
        requestPhoneChangeCode(phone)
    }

    private fun backToEdit() {
        resendTimer?.cancel()
        mutableState.update {
            it.copy(
                mode = ProfileMode.Edit,
                codeInput = "",
                ttlSeconds = null,
                resendAfterSeconds = null,
                resendSecondsRemaining = 0,
                fieldError = null,
                message = null,
            )
        }
    }

    private fun logout() {
        if (mutableState.value.isSubmitting) return

        storeScope.launch {
            mutableState.update {
                it.copy(
                    actionStatus = ActionStatus.Submitting,
                    logoutConfirmVisible = false,
                    message = null,
                )
            }
            authRepository.logout().fold(
                onSuccess = {
                    mutableState.update { it.copy(actionStatus = ActionStatus.Idle) }
                    effects.send(ProfileEffect.SignedOut)
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to logout")
                    mutableState.update { it.copy(actionStatus = ActionStatus.Idle) }
                    effects.send(ProfileEffect.SignedOut)
                },
            )
        }
    }

    private fun deleteAccount() {
        if (mutableState.value.isSubmitting) return

        storeScope.launch {
            mutableState.update {
                it.copy(
                    actionStatus = ActionStatus.Submitting,
                    deleteConfirmVisible = false,
                    message = null,
                )
            }
            profileRepository.deleteAccount().fold(
                onSuccess = {
                    resendTimer?.cancel()
                    mutableState.update { it.copy(actionStatus = ActionStatus.Idle) }
                    effects.send(ProfileEffect.SignedOut)
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to delete account")
                    val appFailure = failure.asAppFailure()
                    if (appFailure == AppFailure.Unauthorized) {
                        effects.send(ProfileEffect.SignedOut)
                    } else {
                        mutableState.update {
                            it.copy(
                                actionStatus = ActionStatus.Idle,
                                message = actionMessage(appFailure),
                            )
                        }
                    }
                },
            )
        }
    }

    private fun handleActionFailure(
        appFailure: AppFailure,
        fieldError: (AppFailure) -> String?,
        message: (AppFailure) -> String?,
    ) {
        if (appFailure == AppFailure.Unauthorized) {
            storeScope.launch { effects.send(ProfileEffect.SignedOut) }
        } else {
            mutableState.update {
                it.copy(
                    actionStatus = ActionStatus.Idle,
                    fieldError = fieldError(appFailure),
                    message = message(appFailure),
                )
            }
        }
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

    private fun reset() {
        resendTimer?.cancel()
        mutableState.value = ProfileState()
    }

    private fun profileUpdateFieldError(failure: AppFailure): String? = when {
        failure.isBadRequest() -> "Проверьте имя — кажется, тут лишние символы"
        else -> null
    }

    private fun profileUpdateMessage(failure: AppFailure): String? = when {
        failure.isBadRequest() -> null
        else -> actionMessage(failure)
    }

    private fun phoneChangeFieldError(failure: AppFailure): String? = when {
        failure.isBadRequest() -> "Похоже, номер введён не полностью"
        else -> null
    }

    private fun phoneChangeMessage(failure: AppFailure): String? = when {
        failure.isBadRequest() -> null
        failure.isPhoneConflict() -> "Этот номер уже используется. Укажите другой"
        failure.isTooManyRequests() -> "Слишком много попыток. Подождите немного"
        else -> actionMessage(failure)
    }

    private fun confirmPhoneFieldError(failure: AppFailure): String? = when {
        failure.isInvalidCode() || failure.isBadRequest() -> "Неверный код. Проверьте и введите ещё раз"
        else -> null
    }

    private fun confirmPhoneMessage(failure: AppFailure): String? = when {
        failure.isInvalidCode() || failure.isBadRequest() -> null
        failure.isPhoneConflict() -> "Этот номер уже используется. Укажите другой"
        else -> actionMessage(failure)
    }

    private fun actionMessage(failure: AppFailure): String = when (failure) {
        AppFailure.NetworkUnavailable -> "Нет соединения. Проверьте подключение"
        AppFailure.Timeout -> "Произошла ошибка. Попробуйте позже"
        else -> "Произошла ошибка. Попробуйте позже"
    }

    private fun AppFailure.isBadRequest(): Boolean =
        this is AppFailure.Api && code == ApiErrorCode.BadRequest

    private fun AppFailure.isInvalidCode(): Boolean =
        this is AppFailure.Api && code == ApiErrorCode.InvalidCode

    private fun AppFailure.isPhoneConflict(): Boolean =
        this is AppFailure.Api && code == ApiErrorCode.PhoneConflict

    private fun AppFailure.isTooManyRequests(): Boolean =
        this is AppFailure.Api && code == ApiErrorCode.TooManyRequests

    private companion object {
        const val DEFAULT_RESEND_SECONDS = 60
    }
}

private fun String.isE164Phone(): Boolean = Regex("^\\+[1-9]\\d{1,14}$").matches(this)
