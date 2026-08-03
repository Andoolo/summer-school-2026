package com.volna.app.catalog.presentation

import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.TrackDirection
import com.volna.app.domain.model.TrackPassport
import com.volna.app.domain.model.TrackRecord
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.yield
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertTrue

class TrackStoreTest {
    @Test
    fun loadsPassportAndLeaderboard() = runTest {
        val repository = FakeTrackRepository(
            passport = Result.success(passport()),
            leaderboard = Result.success(
                listOf(
                    LeaderboardEntry(position = 1, name = "Сергей", bestLapMs = 41890, laps = 3),
                    LeaderboardEntry(position = 2, name = "Иван", bestLapMs = 42318, laps = 3),
                ),
            ),
        )
        val store = TrackStore(repository, CoroutineScope(coroutineContext))

        store.accept(TrackIntent.Load(RouteId("route-1")))
        yield()
        yield()

        val loaded = assertIs<Loadable.Content<TrackPassport>>(store.state.value.passport)
        assertEquals("Городское кольцо", loaded.value.name)
        assertEquals(980, loaded.value.lengthM)
        assertEquals(2, store.state.value.leaderboard.size)
    }

    @Test
    fun leaderboardFailureDoesNotBreakTheScreen() = runTest {
        // Таблица рекордов — вспомогательный блок. Если она не пришла, карточка
        // трассы всё равно обязана показаться: скрывается только сама таблица.
        val repository = FakeTrackRepository(
            passport = Result.success(passport()),
            leaderboard = Result.failure(AppFailureException(AppFailure.NetworkUnavailable)),
        )
        val store = TrackStore(repository, CoroutineScope(coroutineContext))

        store.accept(TrackIntent.Load(RouteId("route-1")))
        yield()
        yield()

        assertIs<Loadable.Content<TrackPassport>>(store.state.value.passport)
        assertTrue(store.state.value.leaderboard.isEmpty())
    }

    @Test
    fun passportFailureShowsError() = runTest {
        val repository = FakeTrackRepository(
            passport = Result.failure(AppFailureException(AppFailure.NetworkUnavailable)),
            leaderboard = Result.success(emptyList()),
        )
        val store = TrackStore(repository, CoroutineScope(coroutineContext))

        store.accept(TrackIntent.Load(RouteId("route-1")))
        yield()

        assertIs<Loadable.Error>(store.state.value.passport)
        // Рекорды не запрашиваем, если самой трассы нет.
        assertEquals(0, repository.leaderboardCalls)
    }

    @Test
    fun retryRepeatsLastRequest() = runTest {
        val repository = FakeTrackRepository(
            passport = Result.failure(AppFailureException(AppFailure.NetworkUnavailable)),
            leaderboard = Result.success(emptyList()),
        )
        val store = TrackStore(repository, CoroutineScope(coroutineContext))

        store.accept(TrackIntent.Load(RouteId("route-1")))
        yield()
        repository.passport = Result.success(passport())
        store.accept(TrackIntent.Retry)
        yield()
        yield()

        assertIs<Loadable.Content<TrackPassport>>(store.state.value.passport)
        assertEquals(RouteId("route-1"), repository.lastRequestedRoute)
    }

    @Test
    fun resetClearsStateSoPreviousTrackDoesNotFlash() = runTest {
        // Без сброса при возврате на экран на мгновение показалась бы прошлая трасса.
        val repository = FakeTrackRepository(
            passport = Result.success(passport()),
            leaderboard = Result.success(
                listOf(LeaderboardEntry(position = 1, name = "Сергей", bestLapMs = 41890, laps = 3)),
            ),
        )
        val store = TrackStore(repository, CoroutineScope(coroutineContext))

        store.accept(TrackIntent.Load(RouteId("route-1")))
        yield()
        yield()
        store.accept(TrackIntent.Reset)

        assertEquals(Loadable.Initial, store.state.value.passport)
        assertTrue(store.state.value.leaderboard.isEmpty())
    }

    private fun passport() = TrackPassport(
        id = RouteId("route-1"),
        name = "Городское кольцо",
        type = RouteType.Novice,
        capacityCap = 8,
        durationMin = 20,
        geometry = null,
        lengthM = 980,
        corners = 10,
        direction = TrackDirection.Clockwise,
        mainStraightM = 200,
        surface = "Асфальт",
        record = TrackRecord(name = "Сергей", lapMs = 41890),
    )
}

private class FakeTrackRepository(
    var passport: Result<TrackPassport>,
    private val leaderboard: Result<List<LeaderboardEntry>>,
) : SlotRepository {
    var leaderboardCalls = 0
        private set
    var lastRequestedRoute: RouteId? = null
        private set

    override suspend fun trackPassport(routeId: RouteId): Result<TrackPassport> {
        lastRequestedRoute = routeId
        return passport
    }

    override suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>> {
        leaderboardCalls++
        return leaderboard
    }

    override suspend fun listSlots(filters: SlotFilters, page: PageRequest): Result<Page<Slot>> =
        Result.success(Page(items = emptyList(), limit = page.limit, offset = page.offset, total = 0))

    override suspend fun getSlot(slotId: SlotId): Result<Slot> =
        Result.failure(AppFailureException(AppFailure.Unknown))
}
