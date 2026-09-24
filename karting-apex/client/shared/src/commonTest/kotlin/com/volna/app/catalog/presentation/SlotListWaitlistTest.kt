package com.volna.app.catalog.presentation

import com.volna.app.catalog.InstructorRepository
import com.volna.app.catalog.MyWaitlistEntry
import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.catalog.WaitlistEntry
import com.volna.app.catalog.WaitlistEntryStatus
import com.volna.app.catalog.WaitlistRepository
import com.volna.app.catalog.WaitlistStatus
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.domain.model.Instructor
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.TrackPassport
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class SlotListWaitlistTest {
    private val now = Instant.parse("2026-09-14T12:00:00Z")
    private val slotId = SlotId("7a170000-0000-4000-8000-000000000001")
    private val waiting = WaitlistEntry(WaitlistEntryStatus.Waiting, seatsCount = 1, position = 3, offerExpiresAt = null)

    private object NoSlots : SlotRepository {
        override suspend fun listSlots(filters: SlotFilters, page: PageRequest): Result<Page<Slot>> =
            Result.success(Page(items = emptyList(), limit = page.limit, offset = page.offset, total = 0))

        override suspend fun getSlot(slotId: SlotId): Result<Slot> = error("not used")
        override suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>> = error("not used")
        override suspend fun trackPassport(routeId: RouteId): Result<TrackPassport> = error("not used")
    }

    private object NoInstructors : InstructorRepository {
        override suspend fun listInstructors(page: PageRequest): Result<Page<Instructor>> =
            Result.success(Page(items = emptyList(), limit = page.limit, offset = page.offset, total = 0))
    }

    private class FakeWaitlist(private val mine: suspend () -> Result<List<MyWaitlistEntry>>) : WaitlistRepository {
        override suspend fun status(slotId: SlotId): Result<WaitlistStatus> = error("not used")
        override suspend fun join(slotId: SlotId, seatsCount: Int): Result<WaitlistEntry> = error("not used")
        override suspend fun leave(slotId: SlotId): Result<Unit> = error("not used")
        override suspend fun mine(): Result<List<MyWaitlistEntry>> = mine.invoke()
    }

    // Стор живёт в backgroundScope, поэтому шаги — runCurrent(): advanceUntilIdle() фоновые
    // задачи сам не запускает.
    private fun TestScope.store(waitlist: WaitlistRepository) =
        SlotListStore(NoSlots, NoInstructors, backgroundScope, now = { now }, waitlistRepository = waitlist)

    @Test
    fun catalogKnowsWhereThePersonIsQueued() = runTest {
        val store = store(FakeWaitlist { Result.success(listOf(MyWaitlistEntry(slotId, "Кольцо", now, waiting))) })

        store.accept(SlotListIntent.Load)
        runCurrent()

        assertTrue(store.state.value.waitlist.available)
        assertEquals(waiting, store.state.value.waitlist.mine[slotId])
    }

    @Test
    fun waitlistSwitchedOffOnServerHidesItsLabel() = runTest {
        val notFound = AppFailureException(AppFailure.Api(ApiErrorCode.NotFound, "Запрашиваемый ресурс не найден."))
        val store = store(FakeWaitlist { Result.failure(notFound) })

        store.accept(SlotListIntent.Load)
        runCurrent()

        assertFalse(store.state.value.waitlist.available)
    }

    @Test
    fun networkErrorKeepsTheGeneralLabel() = runTest {
        val store = store(FakeWaitlist { Result.failure(AppFailureException(AppFailure.NetworkUnavailable)) })

        store.accept(SlotListIntent.Load)
        runCurrent()

        // Не знаем — не прячем: на проде лист ожидания включён.
        assertTrue(store.state.value.waitlist.available)
        assertTrue(store.state.value.waitlist.mine.isEmpty())
    }

    @Test
    fun lateAnswerAfterSignOutDoesNotShowPreviousQueues() = runTest {
        val answer = CompletableDeferred<Result<List<MyWaitlistEntry>>>()
        val store = store(FakeWaitlist { answer.await() })

        store.accept(SlotListIntent.Load)
        runCurrent()
        store.accept(SlotListIntent.Reset)
        answer.complete(Result.success(listOf(MyWaitlistEntry(slotId, "Кольцо", now, waiting))))
        runCurrent()

        assertTrue(store.state.value.waitlist.mine.isEmpty())
    }

    @Test
    fun seatsLabelPrefersOwnQueue() {
        val offered = WaitlistEntry(WaitlistEntryStatus.Offered, seatsCount = 1, position = 0, offerExpiresAt = now)
        assertEquals("Свободно мест", seatsLabel(hasSeats = true, queueEntry = null, waitlistAvailable = true))
        assertEquals("Вам предложено место", seatsLabel(hasSeats = true, queueEntry = offered, waitlistAvailable = true))
        assertEquals("Мест нет · вы в очереди", seatsLabel(hasSeats = false, queueEntry = offered, waitlistAvailable = true))
        assertEquals("Мест нет · 3-й в очереди", seatsLabel(hasSeats = false, queueEntry = waiting, waitlistAvailable = true))
        assertEquals("Мест нет · есть лист ожидания", seatsLabel(hasSeats = false, queueEntry = null, waitlistAvailable = true))
        assertEquals("Мест нет", seatsLabel(hasSeats = false, queueEntry = null, waitlistAvailable = false))
    }
}
