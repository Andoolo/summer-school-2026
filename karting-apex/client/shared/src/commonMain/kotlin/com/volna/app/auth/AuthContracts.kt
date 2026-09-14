package com.volna.app.auth

import com.volna.app.core.error.AppFailure
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.Phone
import kotlinx.datetime.Instant

data class RequestCodeResult(
    val ttlSeconds: Int,
    val resendAfterSeconds: Int,
)

data class VerifyCodeResult(
    val token: String,
    val client: Client,
    val isNew: Boolean,
)

/**
 * Способы входа, которые сейчас доступны. Решает бэкенд: форма входа по SMS не должна
 * появляться там, где SMS не отправляются, а кнопка Telegram — пока бот не настроен.
 */
data class AuthMethods(
    val sms: Boolean,
    val demo: Boolean,
    val telegramBotUsername: String?,
)

/** Начатый вход через Telegram. */
data class TelegramLoginStart(
    /** Секрет опроса: только по нему сервер выдаёт сессию. */
    val pollToken: String,
    val deepLink: String,
    /** Код сверки: бот показывает тот же код. */
    val confirmCode: String,
    val expiresAt: Instant,
)

sealed interface TelegramPollResult {
    data object Pending : TelegramPollResult
    data object Expired : TelegramPollResult
    data class Confirmed(val result: VerifyCodeResult) : TelegramPollResult
}

interface AuthRepository {
    suspend fun authMethods(): Result<AuthMethods>
    suspend fun requestCode(phone: Phone): Result<RequestCodeResult>
    suspend fun verifyCode(phone: Phone, code: String): Result<VerifyCodeResult>
    /** Гостевой вход: новый временный аккаунт без регистрации. */
    suspend fun demoLogin(): Result<VerifyCodeResult>
    suspend fun telegramStart(): Result<TelegramLoginStart>
    suspend fun telegramPoll(pollToken: String): Result<TelegramPollResult>
    suspend fun logout(): Result<Unit>
}

interface SessionRepository {
    suspend fun token(): String?
    suspend fun saveToken(token: String)
    suspend fun clearToken()
}

sealed interface AuthFailure {
    data object InvalidPhone : AuthFailure
    data object InvalidCode : AuthFailure
    data object TooManyRequests : AuthFailure
    data class External(val failure: AppFailure) : AuthFailure
}
