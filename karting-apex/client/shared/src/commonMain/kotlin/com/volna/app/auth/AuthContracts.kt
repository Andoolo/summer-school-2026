package com.volna.app.auth

import com.volna.app.core.error.AppFailure
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.Phone

data class RequestCodeResult(
    val ttlSeconds: Int,
    val resendAfterSeconds: Int,
)

data class VerifyCodeResult(
    val token: String,
    val client: Client,
    val isNew: Boolean,
)

interface AuthRepository {
    suspend fun requestCode(phone: Phone): Result<RequestCodeResult>
    suspend fun verifyCode(phone: Phone, code: String): Result<VerifyCodeResult>
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
