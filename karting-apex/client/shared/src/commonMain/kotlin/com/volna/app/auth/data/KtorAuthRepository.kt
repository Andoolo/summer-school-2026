package com.volna.app.auth.data

import com.volna.app.auth.AuthMethods
import com.volna.app.auth.AuthRepository
import com.volna.app.auth.RequestCodeResult
import com.volna.app.auth.SessionRepository
import com.volna.app.auth.VerifyCodeResult
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.asAppFailure
import com.volna.app.core.network.VolnaApiClient
import com.volna.app.domain.model.Phone
import io.ktor.client.request.HttpRequestBuilder
import io.ktor.client.request.setBody
import io.ktor.http.HttpMethod

class KtorAuthRepository(
    private val apiClient: VolnaApiClient,
    private val sessionRepository: SessionRepository,
) : AuthRepository {
    override suspend fun authMethods(): Result<AuthMethods> =
        apiClient.send<AuthMethodsDto>("/auth/methods")
            .map { it.toDomain() }
            .recoverCatching { failure ->
                // Бэкенд без /auth/methods (ещё не обновлённый) — прежнее поведение:
                // только вход по номеру.
                val appFailure = failure.asAppFailure()
                if (appFailure is AppFailure.Api && appFailure.code == ApiErrorCode.NotFound) {
                    AuthMethods(sms = true, demo = false, telegramBotUsername = null)
                } else {
                    throw failure
                }
            }

    override suspend fun demoLogin(): Result<VerifyCodeResult> {
        val result = apiClient.send<DemoLoginResponseDto>("/auth/demo") {
            postJson()
        }
        result.getOrNull()?.let { response ->
            sessionRepository.saveToken(response.token)
        }
        return result.map { it.toDomain() }
    }

    override suspend fun requestCode(phone: Phone): Result<RequestCodeResult> =
        apiClient.send<RequestCodeResponseDto>("/auth/request-code") {
            postJson()
            setBody(RequestCodeRequestDto(phone.value))
        }.map { it.toDomain() }

    override suspend fun verifyCode(phone: Phone, code: String): Result<VerifyCodeResult> {
        val result = apiClient.send<VerifyCodeResponseDto>("/auth/verify-code") {
            postJson()
            setBody(VerifyCodeRequestDto(phone.value, code))
        }
        result.getOrNull()?.let { response ->
            sessionRepository.saveToken(response.token)
        }
        return result.map { it.toDomain() }
    }

    override suspend fun logout(): Result<Unit> {
        val result = apiClient.sendUnit("/auth/logout", authorized = true) {
            postJson()
        }
        sessionRepository.clearToken()
        return result
    }
}

private fun HttpRequestBuilder.postJson() {
    method = HttpMethod.Post
}
