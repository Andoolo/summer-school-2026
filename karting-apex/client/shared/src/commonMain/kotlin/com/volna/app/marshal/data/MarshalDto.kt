package com.volna.app.marshal.data

import kotlinx.datetime.Instant
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class RaceRosterDto(
    @SerialName("slot_id")
    val slotId: String,
    @SerialName("route_name")
    val routeName: String,
    @SerialName("start_at")
    val startAt: Instant,
    @SerialName("already_raced")
    val alreadyRaced: Boolean,
    val participants: List<RaceParticipantDto>,
)

@Serializable
data class RaceParticipantDto(
    @SerialName("booking_id")
    val bookingId: String,
    @SerialName("racer_name")
    val racerName: String,
    val seats: Int,
    val laps: Int,
    @SerialName("best_lap_ms")
    val bestLapMs: Int? = null,
)

@Serializable
data class LapResultsRequestDto(
    @SerialName("lap_times_ms")
    val lapTimesMs: List<Int>,
)

@Serializable
data class LapResultsResponseDto(
    @SerialName("racer_name")
    val racerName: String,
    @SerialName("route_name")
    val routeName: String,
    val laps: Int,
    @SerialName("best_lap_ms")
    val bestLapMs: Int,
    @SerialName("is_track_record")
    val isTrackRecord: Boolean,
)
