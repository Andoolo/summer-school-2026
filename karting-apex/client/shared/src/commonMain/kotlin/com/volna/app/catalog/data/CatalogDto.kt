package com.volna.app.catalog.data

import kotlinx.datetime.Instant
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class SlotListResponseDto(
    val items: List<SlotDto>,
    val meta: PaginationMetaDto,
)

@Serializable
data class InstructorListResponseDto(
    val items: List<InstructorDto>,
    val meta: PaginationMetaDto,
)

@Serializable
data class PaginationMetaDto(
    val limit: Int,
    val offset: Int,
    val total: Int,
)

@Serializable
data class SlotDto(
    val id: String,
    @SerialName("start_at")
    val startAt: Instant,
    val route: RouteDto,
    val instructor: InstructorDto,
    @SerialName("total_seats")
    val totalSeats: Int,
    @SerialName("free_seats")
    val freeSeats: Int,
    @SerialName("free_rental_boards")
    val freeRentalBoards: Int,
    val price: Int,
    @SerialName("rental_price")
    val rentalPrice: Int,
    @SerialName("meeting_point")
    val meetingPoint: String? = null,
    @SerialName("meeting_point_lat")
    val meetingPointLat: Double? = null,
    @SerialName("meeting_point_lng")
    val meetingPointLng: Double? = null,
    val status: String,
)

@Serializable
data class RouteDto(
    val id: String,
    val name: String,
    val type: String,
    @SerialName("capacity_cap")
    val capacityCap: Int,
    @SerialName("duration_min")
    val durationMin: Int,
    val geometry: JsonElement? = null,
)

@Serializable
data class InstructorDto(
    val id: String,
    val name: String,
)

@Serializable
data class LeaderboardResponseDto(
    @SerialName("route_id")
    val routeId: String,
    @SerialName("route_name")
    val routeName: String,
    val entries: List<LeaderboardEntryDto>,
)

@Serializable
data class LeaderboardEntryDto(
    val position: Int,
    val name: String,
    @SerialName("best_lap_ms")
    val bestLapMs: Int,
    val laps: Int,
)
