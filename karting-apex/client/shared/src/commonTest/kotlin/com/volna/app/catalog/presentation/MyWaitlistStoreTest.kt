package com.volna.app.catalog.presentation

import com.volna.app.catalog.MyWaitlistEntry
import com.volna.app.catalog.WaitlistEntry
import com.volna.app.catalog.WaitlistEntryStatus
import com.volna.app.catalog.WaitlistRepository
import com.volna.app.catalog.WaitlistStatus
import com.volna.app.catalog.data.MyWaitlistEntryDto
import com.volna.app.catalog.data.toDomain
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.domain.model.SlotId
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

/** «Мои очереди» в профиле: загрузка, выход из очереди, ошибки и тексты статусов. */
class MyWaitlistStoreTest {
    private val soon = myEntry("slot-soon", "Городское кольцо", WaitlistEntryStatus.Waiting, position = 3)
    private val later = myEntry("slot-later", "Спортивная трасса", WaitlistEntryStatus.Offered, position = 0)

    @Test
    fun loadsEntriesAndLeavesOne() = runTest {
        val repository = FakeRepository(Result.success(listOf(soon, later)))
        val store = MyWaitlistStore(repository, backgroundScope)

        store.accept(MyWaitlistIntent.Load)
        runCurrent()
        assertEquals(listOf(soon, later), store.state.value.entries)

        store.accept(MyWaitlistIntent.Leave(soon.slotId))
        // Пока запрос идёт, повторное нажатие ничего не отправляет.
        store.accept(MyWaitlistIntent.Leave(soon.slotId))
        assertTrue(soon.slotId in store.state.value.leaving)
        repository.leaveResult.complete(Result.success(Unit))
        runCurrent()

        assertEquals(listOf(later), store.state.value.entries)
        assertTrue(store.state.value.leaving.isEmpty())
        assertEquals(listOf(soon.slotId), repository.leaves)
    }

    @Test
    fun failedLeaveKeepsEntryAndShowsMessage() = runTest {
        val repository = FakeRepository(Result.success(listOf(soon)))
        val store = MyWaitlistStore(repository, backgroundScope)
        store.accept(MyWaitlistIntent.Load)
        runCurrent()

        store.accept(MyWaitlistIntent.Leave(soon.slotId))
        repository.leaveResult.complete(Result.failure(AppFailureException(AppFailure.NetworkUnavailable)))
        runCurrent()

        assertEquals(listOf(soon), store.state.value.entries)
        assertTrue(store.state.value.leaving.isEmpty())
        assertEquals("Нет соединения. Проверьте интернет и попробуйте снова.", store.state.value.message)
    }

    @Test
    fun failedLoadKeepsSectionHidden() = runTest {
        val failure = AppFailure.Api(ApiErrorCode.NotFound, "Лист ожидания недоступен")
        val store = MyWaitlistStore(FakeRepository(Result.failure(AppFailureException(failure))), backgroundScope)
        store.accept(MyWaitlistIntent.Load)
        runCurrent()
        assertTrue(store.state.value.entries.isEmpty())
        assertNull(store.state.value.message)
    }

    @Test
    fun mapsServerItem() {
        val dto = MyWaitlistEntryDto(
            slotId = "slot-1", routeName = "Городское кольцо", startAt = Instant.parse("2026-09-23T14:58:00Z"),
            status = "notified", seatsCount = 2, position = 0, offerExpiresAt = Instant.parse("2026-09-23T15:24:00Z"),
        )
        val entry = dto.toDomain()
        assertEquals(SlotId("slot-1"), entry.slotId)
        assertEquals(WaitlistEntryStatus.Offered, entry.entry.status)
        assertEquals(2, entry.entry.seatsCount)
    }

    @Test
    fun statusTexts() {
        val moscow = TimeZone.of("Europe/Moscow")
        assertEquals("3-й в очереди · 1 место", myEntryStatusText(soon, moscow))
        val offered = later.copy(entry = later.entry.copy(offerExpiresAt = Instant.parse("2026-09-23T15:24:00Z")))
        assertEquals("Место освободилось — запишитесь до 18:24", myEntryStatusText(offered, moscow))
    }

    private fun myEntry(slot: String, route: String, status: WaitlistEntryStatus, position: Int) = MyWaitlistEntry(
        slotId = SlotId(slot),
        routeName = route,
        startAt = Instant.parse("2026-09-23T09:00:00Z"),
        entry = WaitlistEntry(status = status, seatsCount = 1, position = position, offerExpiresAt = null),
    )

    private class FakeRepository(private val mineResult: Result<List<MyWaitlistEntry>>) : WaitlistRepository {
        val leaveResult = CompletableDeferred<Result<Unit>>()
        val leaves = mutableListOf<SlotId>()

        override suspend fun mine(): Result<List<MyWaitlistEntry>> = mineResult
        override suspend fun leave(slotId: SlotId): Result<Unit> {
            leaves += slotId
            return leaveResult.await()
        }

        override suspend fun status(slotId: SlotId): Result<WaitlistStatus> = error("not used")
        override suspend fun join(slotId: SlotId, seatsCount: Int): Result<WaitlistEntry> = error("not used")
    }
}
