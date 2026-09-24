package com.volna.app.catalog.data

import com.volna.app.catalog.MyWaitlistEntry
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

    override suspend fun mine(): Result<List<MyWaitlistEntry>> =
        apiClient.send<MyWaitlistResponseDto>("/waitlist", authorized = true) {
            method = HttpMethod.Get
        }.map { response -> response.items.map { it.toDomain() } }

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
    @SerialName("offer_minutes") val offerMinutes: Int? = null,
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
    offerMinutes = offerMinutes,
)

@Serializable
internal data class MyWaitlistEntryDto(
    @SerialName("slot_id") val slotId: String,
    @SerialName("route_name") val routeName: String,
    @SerialName("start_at") val startAt: Instant,
    val status: String,
    @SerialName("seats_count") val seatsCount: Int,
    val position: Int,
    @SerialName("offer_expires_at") val offerExpiresAt: Instant? = null,
)

@Serializable
internal data class MyWaitlistResponseDto(
    val items: List<MyWaitlistEntryDto> = emptyList(),
)

internal fun MyWaitlistEntryDto.toDomain() = MyWaitlistEntry(
    slotId = SlotId(slotId),
    routeName = routeName,
    startAt = startAt,
    entry = WaitlistEntryDto(status = status, seatsCount = seatsCount, position = position, offerExpiresAt = offerExpiresAt).toDomain(),
)
