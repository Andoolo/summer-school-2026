package com.volna.app.catalog.presentation

import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.TrackDirection
import com.volna.app.domain.model.TrackPassport
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

/** Поздние ответы по прошлой трассе не должны попадать на экран открытой сейчас. */
class TrackStoreRaceTest {
    private val routeA = RouteId("route-a")
    private val routeB = RouteId("route-b")
    private val recordsA = listOf(LeaderboardEntry(position = 1, name = "Сергей", bestLapMs = 41890, laps = 3))
    private val recordsB = listOf(LeaderboardEntry(position = 1, name = "Иван", bestLapMs = 56771, laps = 2))

    @Test
    fun stalePassportDoesNotReplaceNewerTrack() = runTest {
        val repository = ControlledTrackRepository()
        val store = TrackStore(repository, backgroundScope)

        store.accept(TrackIntent.Load(routeA))
        runCurrent()
        store.accept(TrackIntent.Load(routeB))
        runCurrent()
        repository.passport(routeB).complete(Result.success(passport(routeB, "Спортивная трасса")))
        runCurrent()
        repository.passport(routeA).complete(Result.success(passport(routeA, "Городское кольцо")))
        runCurrent()

        assertEquals(routeB, assertIs<Loadable.Content<TrackPassport>>(store.state.value.passport).value.id)
    }

    @Test
    fun staleLeaderboardDoesNotReplaceNewerTrackRecords() = runTest {
        val repository = ControlledTrackRepository()
        val store = TrackStore(repository, backgroundScope)

        store.accept(TrackIntent.Load(routeA))
        runCurrent()
        repository.passport(routeA).complete(Result.success(passport(routeA, "Городское кольцо")))
        runCurrent()
        store.accept(TrackIntent.Load(routeB))
        runCurrent()
        repository.passport(routeB).complete(Result.success(passport(routeB, "Спортивная трасса")))
        runCurrent()
        repository.board(routeB).complete(Result.success(recordsB))
        runCurrent()
        repository.board(routeA).complete(Result.success(recordsA))
        runCurrent()

        assertEquals(recordsB, store.state.value.leaderboard)
    }

    private fun passport(id: RouteId, name: String) = TrackPassport(
        id = id,
        name = name,
        type = RouteType.Novice,
        capacityCap = 8,
        durationMin = 20,
        geometry = null,
        lengthM = 980,
        corners = 10,
        direction = TrackDirection.Clockwise,
        mainStraightM = 200,
        surface = "Асфальт",
        record = null,
    )

    private class ControlledTrackRepository : SlotRepository {
        private val passports = mutableMapOf<RouteId, CompletableDeferred<Result<TrackPassport>>>()
        private val boards = mutableMapOf<RouteId, CompletableDeferred<Result<List<LeaderboardEntry>>>>()

        fun passport(id: RouteId) = passports.getOrPut(id) { CompletableDeferred() }
        fun board(id: RouteId) = boards.getOrPut(id) { CompletableDeferred() }

        override suspend fun trackPassport(routeId: RouteId): Result<TrackPassport> = passport(routeId).await()
        override suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>> = board(routeId).await()
        override suspend fun listSlots(filters: SlotFilters, page: PageRequest): Result<Page<Slot>> = error("not used")
        override suspend fun getSlot(slotId: SlotId): Result<Slot> = error("not used")
    }
}
