package com.volna.app.catalog.presentation

import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.GeoPoint
import com.volna.app.domain.model.Instructor
import com.volna.app.domain.model.InstructorId
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.MeetingPoint
import com.volna.app.domain.model.MoneyRub
import com.volna.app.domain.model.Route
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.SlotStatus
import com.volna.app.domain.model.TrackPassport
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertTrue

/**
 * Экран заезда: сам заезд и рекорды трассы (F4). Ответы сети управляются вручную, чтобы
 * проверять порядок их прихода — на медленной сети он бывает любым. Стор работает в
 * backgroundScope: незавершённые «запросы» отменяются в конце теста, а не валят его.
 */
class SlotDetailsStoreTest {
    private val slotA = slot("slot-a", "route-a", "Городское кольцо")
    private val slotB = slot("slot-b", "route-b", "Спортивная трасса")
    private val recordsA = listOf(LeaderboardEntry(position = 1, name = "Сергей", bestLapMs = 41890, laps = 3))
    private val recordsB = listOf(LeaderboardEntry(position = 1, name = "Иван", bestLapMs = 56771, laps = 2))

    @Test
    fun loadsSlotThenLeaderboard() = runTest {
        val repository = ControlledSlotRepository()
        val store = SlotDetailsStore(repository, backgroundScope)

        store.accept(SlotDetailsIntent.Load(slotA.id))
        runCurrent()
        assertEquals(Loadable.Loading, store.state.value.slot)

        repository.slot(slotA.id).complete(Result.success(slotA))
        runCurrent()
        assertEquals(slotA, assertIs<Loadable.Content<Slot>>(store.state.value.slot).value)
        // Рекорды грузятся после заезда и секцию не блокируют.
        assertTrue(store.state.value.leaderboard.isEmpty())

        repository.board(slotA.route.id).complete(Result.success(recordsA))
        runCurrent()
        assertEquals(recordsA, store.state.value.leaderboard)
    }

    @Test
    fun leaderboardFailureKeepsSlotAndHidesSection() = runTest {
        val repository = ControlledSlotRepository()
        val store = SlotDetailsStore(repository, backgroundScope)

        store.accept(SlotDetailsIntent.Load(slotA.id))
        runCurrent()
        repository.slot(slotA.id).complete(Result.success(slotA))
        runCurrent()
        repository.board(slotA.route.id).complete(Result.failure(AppFailureException(AppFailure.NetworkUnavailable)))
        runCurrent()

        assertIs<Loadable.Content<Slot>>(store.state.value.slot)
        assertTrue(store.state.value.leaderboard.isEmpty())
    }

    @Test
    fun slotFailureShowsErrorAndRetryRepeatsLastSlot() = runTest {
        val repository = ControlledSlotRepository()
        val store = SlotDetailsStore(repository, backgroundScope)

        store.accept(SlotDetailsIntent.Load(slotA.id))
        runCurrent()
        repository.slot(slotA.id).complete(Result.failure(AppFailureException(AppFailure.NetworkUnavailable)))
        runCurrent()
        assertEquals(Loadable.Error(AppFailure.NetworkUnavailable), store.state.value.slot)

        repository.resetSlot(slotA.id)
        store.accept(SlotDetailsIntent.Retry)
        runCurrent()
        repository.slot(slotA.id).complete(Result.success(slotA))
        runCurrent()
        assertIs<Loadable.Content<Slot>>(store.state.value.slot)
        assertEquals(listOf(slotA.id, slotA.id), repository.requestedSlots)
    }

    @Test
    fun unauthorizedSignsOut() = runTest {
        val repository = ControlledSlotRepository()
        val store = SlotDetailsStore(repository, backgroundScope)

        store.accept(SlotDetailsIntent.Load(slotA.id))
        runCurrent()
        repository.slot(slotA.id).complete(Result.failure(AppFailureException(AppFailure.Unauthorized)))
        runCurrent()

        assertEquals(SlotDetailsEffect.SignedOut, store.effects())
    }

    @Test
    fun staleLeaderboardOfPreviousSlotIsIgnored() = runTest {
        val repository = ControlledSlotRepository()
        val store = SlotDetailsStore(repository, backgroundScope)

        store.accept(SlotDetailsIntent.Load(slotA.id))
        runCurrent()
        repository.slot(slotA.id).complete(Result.success(slotA))
        runCurrent()
        // Рекорды трассы A ещё в пути, а человек уже открыл заезд B.
        store.accept(SlotDetailsIntent.Load(slotB.id))
        runCurrent()
        repository.slot(slotB.id).complete(Result.success(slotB))
        runCurrent()
        repository.board(slotB.route.id).complete(Result.success(recordsB))
        runCurrent()
        // Поздний ответ по трассе A не должен подменить рекорды трассы B.
        repository.board(slotA.route.id).complete(Result.success(recordsA))
        runCurrent()

        assertEquals(slotB, assertIs<Loadable.Content<Slot>>(store.state.value.slot).value)
        assertEquals(recordsB, store.state.value.leaderboard)
    }

    @Test
    fun staleSlotResponseDoesNotReplaceNewerSlot() = runTest {
        val repository = ControlledSlotRepository()
        val store = SlotDetailsStore(repository, backgroundScope)

        store.accept(SlotDetailsIntent.Load(slotA.id))
        runCurrent()
        store.accept(SlotDetailsIntent.Load(slotB.id))
        runCurrent()
        repository.slot(slotB.id).complete(Result.success(slotB))
        runCurrent()
        // Ответ по A пришёл последним — на экране должен остаться B.
        repository.slot(slotA.id).complete(Result.success(slotA))
        runCurrent()

        assertEquals(slotB, assertIs<Loadable.Content<Slot>>(store.state.value.slot).value)
        assertEquals(listOf(slotB.route.id), repository.requestedBoards)
    }

    @Test
    fun routeMapOpensAndClosesAndResetsOnLoad() = runTest {
        val repository = ControlledSlotRepository()
        val store = SlotDetailsStore(repository, backgroundScope)

        store.accept(SlotDetailsIntent.OpenRouteMap)
        assertTrue(store.state.value.showRouteMap)
        store.accept(SlotDetailsIntent.DismissRouteMap)
        assertEquals(false, store.state.value.showRouteMap)

        store.accept(SlotDetailsIntent.OpenRouteMap)
        store.accept(SlotDetailsIntent.Load(slotA.id))
        runCurrent()
        // Шторка карты от прошлого заезда не должна оставаться открытой.
        assertEquals(false, store.state.value.showRouteMap)
    }

    private class ControlledSlotRepository : SlotRepository {
        private val slots = mutableMapOf<SlotId, CompletableDeferred<Result<Slot>>>()
        private val boards = mutableMapOf<RouteId, CompletableDeferred<Result<List<LeaderboardEntry>>>>()
        val requestedSlots = mutableListOf<SlotId>()
        val requestedBoards = mutableListOf<RouteId>()

        fun slot(id: SlotId) = slots.getOrPut(id) { CompletableDeferred() }
        fun board(id: RouteId) = boards.getOrPut(id) { CompletableDeferred() }
        fun resetSlot(id: SlotId) {
            slots.remove(id)
        }

        override suspend fun getSlot(slotId: SlotId): Result<Slot> {
            requestedSlots += slotId
            return slot(slotId).await()
        }

        override suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>> {
            requestedBoards += routeId
            return board(routeId).await()
        }

        override suspend fun listSlots(filters: SlotFilters, page: PageRequest): Result<Page<Slot>> = error("not used")
        override suspend fun trackPassport(routeId: RouteId): Result<TrackPassport> = error("not used")
    }
}

internal fun slot(id: String, routeId: String, routeName: String) = Slot(
    id = SlotId(id),
    startAt = Instant.parse("2026-09-23T09:00:00Z"),
    route = Route(
        id = RouteId(routeId),
        name = routeName,
        type = RouteType.Novice,
        capacityCap = 8,
        durationMin = 20,
        geometry = null,
    ),
    instructor = Instructor(InstructorId("instructor-1"), "Марат"),
    totalSeats = 8,
    freeSeats = 5,
    freeRentalBoards = 12,
    price = MoneyRub(2500),
    rentalPrice = MoneyRub(800),
    meetingPoint = MeetingPoint(title = "Главный бокс", coordinates = GeoPoint(55.75, 37.61)),
    status = SlotStatus.Scheduled,
)
