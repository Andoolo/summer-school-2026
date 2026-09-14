package com.volna.app.auth.data

import kotlinx.datetime.Instant
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class RequestCodeRequestDto(
    val phone: String,
)

@Serializable
data class RequestCodeResponseDto(
    @SerialName("ttl_seconds")
    val ttlSeconds: Int,
    @SerialName("resend_after_seconds")
    val resendAfterSeconds: Int,
)

@Serializable
data class VerifyCodeRequestDto(
    val phone: String,
    val code: String,
)

@Serializable
data class VerifyCodeResponseDto(
    val token: String,
    val client: ClientDto,
    @SerialName("is_new")
    val isNew: Boolean,
)

@Serializable
data class ClientDto(
    val id: String,
    val name: String? = null,
    val phone: String,
    @SerialName("created_at")
    val createdAt: Instant,
)

@Serializable
data class AuthMethodsDto(
    val sms: Boolean = false,
    val demo: Boolean = false,
    val telegram: TelegramMethodDto? = null,
)

@Serializable
data class TelegramMethodDto(
    @SerialName("bot_username")
    val botUsername: String,
)

@Serializable
data class DemoLoginResponseDto(
    val token: String,
    val client: ClientDto,
    @SerialName("demo_expires_at")
    val demoExpiresAt: Instant,
)

@Serializable
data class TelegramStartResponseDto(
    @SerialName("poll_token")
    val pollToken: String,
    @SerialName("deep_link")
    val deepLink: String,
    @SerialName("confirm_code")
    val confirmCode: String,
    @SerialName("expires_at")
    val expiresAt: Instant,
)

@Serializable
data class TelegramPollRequestDto(
    @SerialName("poll_token")
    val pollToken: String,
)

@Serializable
data class TelegramPollResponseDto(
    val status: String,
    val token: String? = null,
    val client: ClientDto? = null,
    @SerialName("is_new")
    val isNew: Boolean = false,
)
