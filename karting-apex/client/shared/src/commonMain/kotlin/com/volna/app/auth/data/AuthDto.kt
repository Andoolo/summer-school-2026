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
