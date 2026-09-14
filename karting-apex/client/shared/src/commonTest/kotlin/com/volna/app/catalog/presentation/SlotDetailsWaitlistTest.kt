package com.volna.app.catalog.presentation

import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.catalog.WaitlistEntry
import com.volna.app.catalog.WaitlistEntryStatus
import com.volna.app.catalog.WaitlistRepository
import com.volna.app.catalog.WaitlistStatus
import com.volna.app.catalog.data.WaitlistEntryDto
import com.volna.app.catalog.data.WaitlistStatusDto
import com.volna.app.catalog.data.toDomain
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.TrackPassport
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/** Лист ожидания на экране заезда: загрузка статуса, вход в очередь, выход и ошибки. */
class SlotDetailsWaitlistTest {
    private val fullSlot = slot("slot-full", "route-a", "Городское кольцо").copy(freeSeats = 0)
    private val linked = WaitlistStatus(entry = null, telegramLinked = true, notificationsEnabled = true)
    private val waiting = WaitlistEntry(WaitlistEntryStatus.Waiting, seatsCount = 2, position = 3, offerExpiresAt = null)

    @Test
    fun joinsAndLeavesQueue() = runTest {
        val waitlist = FakeWaitlistRepository(status = Result.success(linked), join = Result.success(waiting))
        val store = SlotDetailsStore(ImmediateSlotRepository(fullSlot), backgroundScope, waitlist)

        store.accept(SlotDetailsIntent.Load(fullSlot.id))
        runCurrent()
        assertEquals(linked, store.state.value.waitlist.status)

        store.accept(SlotDetailsIntent.SelectWaitlistSeats(2))
        store.accept(SlotDetailsIntent.JoinWaitlist)
        runCurrent()
        assertEquals(listOf(fullSlot.id to 2), waitlist.joins)
        assertEquals(waiting, store.state.value.waitlist.status?.entry)
        assertEquals(false, store.state.value.waitlist.inProgress)

        store.accept(SlotDetailsIntent.LeaveWaitlist)
        runCurrent()
        assertEquals(1, waitlist.leaves)
        assertNull(store.state.value.waitlist.status?.entry)
    }

    @Test
    fun seatsAreClampedToAllowedRange() = runTest {
        val store = SlotDetailsStore(ImmediateSlotRepository(fullSlot), backgroundScope, FakeWaitlistRepository(Result.success(linked)))
        store.accept(SlotDetailsIntent.SelectWaitlistSeats(9))
        assertEquals(3, store.state.value.waitlist.seats)
        store.accept(SlotDetailsIntent.SelectWaitlistSeats(0))
        assertEquals(1, store.state.value.waitlist.seats)
    }

    @Test
    fun showsServerMessageOnFailure() = runTest {
        val failure = AppFailure.Api(ApiErrorCode.WaitlistLimit, "Можно стоять в очереди не больше чем на 5 заездов.")
        val waitlist = FakeWaitlistRepository(Result.success(linked), join = Result.failure(AppFailureException(failure)))
        val store = SlotDetailsStore(ImmediateSlotRepository(fullSlot), backgroundScope, waitlist)
        store.accept(SlotDetailsIntent.Load(fullSlot.id))
        runCurrent()

        store.accept(SlotDetailsIntent.JoinWaitlist)
        runCurrent()
        assertEquals(failure.message, store.state.value.waitlist.message)
        assertEquals(false, store.state.value.waitlist.inProgress)
        // Выбор мест сбрасывает прошлую ошибку.
        store.accept(SlotDetailsIntent.SelectWaitlistSeats(1))
        assertNull(store.state.value.waitlist.message)
    }

    @Test
    fun reloadsSlotWhenSeatsAppeared() = runTest {
        val seatsAppeared = AppFailure.Api(ApiErrorCode.SeatsAvailable, "Места уже есть — можно записаться сразу.")
        val slots = ImmediateSlotRepository(fullSlot)
        val waitlist = FakeWaitlistRepository(Result.success(linked), join = Result.failure(AppFailureException(seatsAppeared)))
        val store = SlotDetailsStore(slots, backgroundScope, waitlist)
        store.accept(SlotDetailsIntent.Load(fullSlot.id))
        runCurrent()

        store.accept(SlotDetailsIntent.JoinWaitlist)
        runCurrent()
        assertEquals(2, slots.requests, "slot must be reloaded to show the booking button")
    }

    @Test
    fun disabledWaitlistHidesSection() = runTest {
        val store = SlotDetailsStore(ImmediateSlotRepository(fullSlot), backgroundScope)
        store.accept(SlotDetailsIntent.Load(fullSlot.id))
        runCurrent()
        assertNull(store.state.value.waitlist.status)
        // Без статуса вход в очередь ничего не делает.
        store.accept(SlotDetailsIntent.JoinWaitlist)
        runCurrent()
        assertEquals(false, store.state.value.waitlist.inProgress)
    }

    @Test
    fun mapsServerStatus() {
        val dto = WaitlistStatusDto(
            entry = WaitlistEntryDto(status = "notified", seatsCount = 1, position = 0, offerExpiresAt = Instant.parse("2026-09-23T09:15:00Z")),
            telegramLinked = true,
            notificationsEnabled = true,
        )
        val status = dto.toDomain()
        assertEquals(WaitlistEntryStatus.Offered, status.entry?.status)
        assertNotNull(status.entry?.offerExpiresAt)
        assertEquals(WaitlistEntryStatus.Waiting, WaitlistEntryDto("waiting", 2, 1).toDomain().status)
    }

    @Test
    fun texts() {
        val moscow = TimeZone.of("Europe/Moscow")
        assertEquals("3-й в очереди · 2 места", waitingText(3, 2))
        assertEquals("1-й в очереди · 1 место", waitingText(1, 1))
        assertTrue(offerText(Instant.parse("2026-09-23T09:05:00Z"), moscow).startsWith("Запишитесь до 12:05"))
        assertTrue(joinBlockedHint(telegramLinked = false, notificationsEnabled = true)!!.contains("войдите через Telegram"))
        assertTrue(joinBlockedHint(telegramLinked = true, notificationsEnabled = false)!!.contains("/notify"))
        assertNull(joinBlockedHint(telegramLinked = true, notificationsEnabled = true))
    }

    private class ImmediateSlotRepository(private val slot: Slot) : SlotRepository {
        var requests = 0
        override suspend fun getSlot(slotId: SlotId): Result<Slot> {
            requests++
            return Result.success(slot)
        }

        override suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>> = Result.success(emptyList())
        override suspend fun listSlots(filters: SlotFilters, page: PageRequest): Result<Page<Slot>> = error("not used")
        override suspend fun trackPassport(routeId: RouteId): Result<TrackPassport> = error("not used")
    }

    private class FakeWaitlistRepository(
        private val status: Result<WaitlistStatus>,
        private val join: Result<WaitlistEntry> = Result.failure(IllegalStateException("not set")),
    ) : WaitlistRepository {
        val joins = mutableListOf<Pair<SlotId, Int>>()
        var leaves = 0

        override suspend fun status(slotId: SlotId): Result<WaitlistStatus> = status
        override suspend fun join(slotId: SlotId, seatsCount: Int): Result<WaitlistEntry> {
            joins += slotId to seatsCount
            return join
        }

        override suspend fun leave(slotId: SlotId): Result<Unit> {
            leaves++
            return Result.success(Unit)
        }

        override suspend fun mine(): Result<List<com.volna.app.catalog.MyWaitlistEntry>> = Result.success(emptyList())
    }
}
