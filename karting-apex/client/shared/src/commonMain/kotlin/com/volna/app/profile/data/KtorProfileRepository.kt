package com.volna.app.profile.data

import com.volna.app.auth.RequestCodeResult
import com.volna.app.auth.SessionRepository
import com.volna.app.core.network.VolnaApiClient
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.Phone
import com.volna.app.profile.ProfileRepository
import io.ktor.client.request.HttpRequestBuilder
import io.ktor.client.request.setBody
import io.ktor.http.HttpMethod

class KtorProfileRepository(
    private val apiClient: VolnaApiClient,
    private val sessionRepository: SessionRepository,
) : ProfileRepository {
    override suspend fun getProfile(): Result<Client> =
        apiClient.send<ProfileClientDto>("/profile", authorized = true) {
            method = HttpMethod.Get
        }.map { it.toDomain() }

    override suspend fun updateName(name: String): Result<Client> =
        apiClient.send<ProfileClientDto>("/profile", authorized = true) {
            method = HttpMethod.Patch
            setBody(UpdateProfileRequestDto(name))
        }.map { it.toDomain() }

    override suspend fun deleteAccount(): Result<Unit> {
        val result = apiClient.sendUnit("/profile", authorized = true) {
            method = HttpMethod.Delete
        }
        if (result.isSuccess) {
            sessionRepository.clearToken()
        }
        return result
    }

    override suspend fun requestPhoneChangeCode(newPhone: Phone): Result<RequestCodeResult> =
        apiClient.send<ProfileRequestCodeResponseDto>("/profile/phone/request-code", authorized = true) {
            method = HttpMethod.Post
            setBody(ChangePhoneRequestCodeRequestDto(newPhone.value))
        }.map { it.toDomain() }

    override suspend fun confirmPhoneChange(newPhone: Phone, code: String): Result<Client> =
        apiClient.send<ProfileClientDto>("/profile/phone/confirm", authorized = true) {
            method = HttpMethod.Post
            setBody(ChangePhoneConfirmRequestDto(newPhone.value, code))
        }.map { it.toDomain() }
}
