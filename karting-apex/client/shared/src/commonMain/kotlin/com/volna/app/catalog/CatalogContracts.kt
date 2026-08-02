package com.volna.app.catalog

import com.volna.app.domain.model.Instructor
import com.volna.app.domain.model.InstructorId
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.TrackPassport
import kotlinx.datetime.Instant

data class SlotFilters(
    val dateFrom: Instant? = null,
    val dateTo: Instant? = null,
    val routeTypes: Set<RouteType> = emptySet(),
    val instructorIds: Set<InstructorId> = emptySet(),
    val onlyAvailable: Boolean = false,
)

data class PageRequest(
    val limit: Int = 20,
    val offset: Int = 0,
)

data class Page<T>(
    val items: List<T>,
    val limit: Int,
    val offset: Int,
    val total: Int,
)

interface SlotRepository {
    suspend fun listSlots(filters: SlotFilters, page: PageRequest = PageRequest()): Result<Page<Slot>>
    suspend fun getSlot(slotId: SlotId): Result<Slot>

    /** Рекорды трассы: топ лучших кругов (картинг-фича, GET /routes/{id}/leaderboard). */
    suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>>

    /** Карточка трассы: схема, характеристики и рекорд круга (картинг-фича F5, GET /routes/{id}). */
    suspend fun trackPassport(routeId: RouteId): Result<TrackPassport>
}

interface InstructorRepository {
    suspend fun listInstructors(page: PageRequest = PageRequest(limit = 100)): Result<Page<Instructor>>
}
