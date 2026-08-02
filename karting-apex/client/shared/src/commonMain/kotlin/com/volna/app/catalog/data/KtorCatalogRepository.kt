package com.volna.app.catalog.data

import com.volna.app.catalog.InstructorRepository
import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.domain.model.Instructor
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.TrackPassport
import com.volna.app.core.network.VolnaApiClient
import io.ktor.client.request.parameter
import io.ktor.http.HttpMethod

class KtorSlotRepository(
    private val apiClient: VolnaApiClient,
) : SlotRepository {
    override suspend fun listSlots(filters: SlotFilters, page: PageRequest): Result<Page<Slot>> =
        apiClient.send<SlotListResponseDto>("/slots", authorized = true) {
            method = HttpMethod.Get
            filters.dateFrom?.let { parameter("date_from", it.toString()) }
            filters.dateTo?.let { parameter("date_to", it.toString()) }
            filters.routeTypes.forEach { parameter("route_type", it.toApiValue()) }
            filters.instructorIds.forEach { parameter("instructor_id", it.value) }
            if (filters.onlyAvailable) parameter("only_available", true)
            parameter("limit", page.limit)
            parameter("offset", page.offset)
        }.map { it.toDomain() }

    override suspend fun getSlot(slotId: SlotId): Result<Slot> =
        apiClient.send<SlotDto>("/slots/${slotId.value}", authorized = true) {
            method = HttpMethod.Get
        }.map { it.toDomain() }

    override suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>> =
        apiClient.send<LeaderboardResponseDto>("/routes/${routeId.value}/leaderboard", authorized = false) {
            method = HttpMethod.Get
        }.map { response -> response.entries.map { it.toDomain() } }

    override suspend fun trackPassport(routeId: RouteId): Result<TrackPassport> =
        apiClient.send<TrackPassportDto>("/routes/${routeId.value}", authorized = false) {
            method = HttpMethod.Get
        }.map { it.toDomain() }
}

class KtorInstructorRepository(
    private val apiClient: VolnaApiClient,
) : InstructorRepository {
    override suspend fun listInstructors(page: PageRequest): Result<Page<Instructor>> =
        apiClient.send<InstructorListResponseDto>("/instructors", authorized = true) {
            method = HttpMethod.Get
            parameter("limit", page.limit)
            parameter("offset", page.offset)
        }.map { it.toDomain() }
}

private fun RouteType.toApiValue(): String = when (this) {
    RouteType.Novice -> "novice"
    RouteType.Experienced -> "experienced"
}
