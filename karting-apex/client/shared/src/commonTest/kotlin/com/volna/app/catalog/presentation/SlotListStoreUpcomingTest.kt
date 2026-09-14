package com.volna.app.catalog.presentation

import com.volna.app.catalog.InstructorRepository
import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.domain.model.Instructor
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.TrackPassport
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals

class SlotListStoreUpcomingTest {
    private val now = Instant.parse("2026-09-14T12:00:00Z")

    private class RecordingSlotRepository : SlotRepository {
        val requested = mutableListOf<SlotFilters>()

        override suspend fun listSlots(filters: SlotFilters, page: PageRequest): Result<Page<Slot>> {
            requested += filters
            return Result.success(Page(items = emptyList(), limit = page.limit, offset = page.offset, total = 0))
        }

        override suspend fun getSlot(slotId: SlotId): Result<Slot> = error("not used")
        override suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>> = error("not used")
        override suspend fun trackPassport(routeId: RouteId): Result<TrackPassport> = error("not used")
    }

    private object NoInstructors : InstructorRepository {
        override suspend fun listInstructors(page: PageRequest): Result<Page<Instructor>> =
            Result.success(Page(items = emptyList(), limit = page.limit, offset = page.offset, total = 0))
    }

    @Test
    fun catalogNeverRequestsPastSlots() = runTest {
        val repository = RecordingSlotRepository()
        val store = SlotListStore(repository, NoInstructors, CoroutineScope(coroutineContext), now = { now })

        store.accept(SlotListIntent.Load)
        advanceUntilIdle()

        // Без фильтров — всё равно только заезды с текущего момента.
        assertEquals(now, repository.requested.single().dateFrom)
        // Сам выбранный фильтр при этом не меняется: в интерфейсе по-прежнему «не выбрано».
        assertEquals(SlotFilters(), store.state.value.filters)
    }
}
