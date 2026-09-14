package com.volna.app.catalog.data

import com.volna.app.catalog.WaitlistEntry
import com.volna.app.catalog.WaitlistEntryStatus
import com.volna.app.catalog.WaitlistRepository
import com.volna.app.catalog.WaitlistStatus
import com.volna.app.core.network.VolnaApiClient
import com.volna.app.domain.model.SlotId
import io.ktor.client.request.setBody
import io.ktor.http.HttpMethod
import kotlinx.datetime.Instant
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

class KtorWaitlistRepository(
    private val apiClient: VolnaApiClient,
) : WaitlistRepository {
    override suspend fun status(slotId: SlotId): Result<WaitlistStatus> =
        apiClient.send<WaitlistStatusDto>(path(slotId), authorized = true) {
            method = HttpMethod.Get
        }.map { it.toDomain() }

    override suspend fun join(slotId: SlotId, seatsCount: Int): Result<WaitlistEntry> =
        apiClient.send<WaitlistEntryDto>(path(slotId), authorized = true) {
            method = HttpMethod.Post
            setBody(JoinWaitlistRequestDto(seatsCount))
        }.map { it.toDomain() }

    override suspend fun leave(slotId: SlotId): Result<Unit> =
        apiClient.sendUnit(path(slotId), authorized = true) {
            method = HttpMethod.Delete
        }

    private fun path(slotId: SlotId) = "/slots/${slotId.value}/waitlist"
}

@Serializable
internal data class JoinWaitlistRequestDto(
    @SerialName("seats_count") val seatsCount: Int,
)

@Serializable
internal data class WaitlistEntryDto(
    val status: String,
    @SerialName("seats_count") val seatsCount: Int,
    val position: Int,
    @SerialName("offer_expires_at") val offerExpiresAt: Instant? = null,
)

@Serializable
internal data class WaitlistStatusDto(
    val entry: WaitlistEntryDto? = null,
    @SerialName("telegram_linked") val telegramLinked: Boolean = false,
    @SerialName("notifications_enabled") val notificationsEnabled: Boolean = false,
)

internal fun WaitlistEntryDto.toDomain() = WaitlistEntry(
    status = if (status == "notified") WaitlistEntryStatus.Offered else WaitlistEntryStatus.Waiting,
    seatsCount = seatsCount,
    position = position,
    offerExpiresAt = offerExpiresAt,
)

internal fun WaitlistStatusDto.toDomain() = WaitlistStatus(
    entry = entry?.toDomain(),
    telegramLinked = telegramLinked,
    notificationsEnabled = notificationsEnabled,
)
