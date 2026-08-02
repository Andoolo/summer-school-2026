package com.volna.app.profile.data

import kotlinx.datetime.Instant
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ProfileClientDto(
    val id: String,
    val name: String? = null,
    val phone: String,
    @SerialName("created_at")
    val createdAt: Instant,
)

@Serializable
data class UpdateProfileRequestDto(
    val name: String,
)

@Serializable
data class ChangePhoneRequestCodeRequestDto(
    @SerialName("new_phone")
    val newPhone: String,
)

@Serializable
data class ChangePhoneConfirmRequestDto(
    @SerialName("new_phone")
    val newPhone: String,
    val code: String,
)

@Serializable
data class ProfileRequestCodeResponseDto(
    @SerialName("ttl_seconds")
    val ttlSeconds: Int,
    @SerialName("resend_after_seconds")
    val resendAfterSeconds: Int,
)
