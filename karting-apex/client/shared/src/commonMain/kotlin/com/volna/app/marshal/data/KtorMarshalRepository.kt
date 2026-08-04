package com.volna.app.marshal.data

import com.volna.app.core.network.VolnaApiClient
import com.volna.app.domain.model.BookingId
import com.volna.app.domain.model.SlotId
import com.volna.app.marshal.LapEntryResult
import com.volna.app.marshal.MarshalRepository
import com.volna.app.marshal.RaceParticipant
import com.volna.app.marshal.RaceRoster
import io.ktor.client.request.header
import io.ktor.client.request.setBody
import io.ktor.http.HttpMethod

private const val MARSHAL_TOKEN_HEADER = "X-Marshal-Token"

class KtorMarshalRepository(
    private val apiClient: VolnaApiClient,
) : MarshalRepository {
    // authorized = false намеренно: маршал не пользуется клиентской сессией,
    // его доступ подтверждается отдельным заголовком.
    override suspend fun raceRoster(token: String, slotId: SlotId): Result<RaceRoster> =
        apiClient.send<RaceRosterDto>("/slots/${slotId.value}/participants", authorized = false) {
            method = HttpMethod.Get
            header(MARSHAL_TOKEN_HEADER, token)
        }.map { it.toDomain() }

    override suspend fun submitLapResults(
        token: String,
        bookingId: BookingId,
        lapTimesMs: List<Int>,
    ): Result<LapEntryResult> =
        apiClient.send<LapResultsResponseDto>("/bookings/${bookingId.value}/lap-results", authorized = false) {
            method = HttpMethod.Post
            header(MARSHAL_TOKEN_HEADER, token)
            setBody(LapResultsRequestDto(lapTimesMs))
        }.map { it.toDomain() }
}

private fun RaceRosterDto.toDomain(): RaceRoster = RaceRoster(
    slotId = SlotId(slotId),
    routeName = routeName,
    startAt = startAt,
    alreadyRaced = alreadyRaced,
    participants = participants.map {
        RaceParticipant(
            bookingId = BookingId(it.bookingId),
            racerName = it.racerName,
            seats = it.seats,
            laps = it.laps,
            bestLapMs = it.bestLapMs,
        )
    },
)

private fun LapResultsResponseDto.toDomain(): LapEntryResult = LapEntryResult(
    racerName = racerName,
    routeName = routeName,
    laps = laps,
    bestLapMs = bestLapMs,
    isTrackRecord = isTrackRecord,
)
