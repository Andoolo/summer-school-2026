package com.volna.app.catalog.presentation

import com.volna.app.catalog.InstructorRepository
import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.core.ui.EmptyReason
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.Instructor
import com.volna.app.domain.model.InstructorId
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.TrackPassport
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

/**
 * Каталог: фильтры в шторке и пресеты дат. Часы и часовой пояс фиксированы: пресеты
 * считаются от «сегодня» в поясе пользователя, и ошибка на границе дня видна только так.
 */
class SlotListStoreFiltersTest {
    private val moscow = TimeZone.of("Europe/Moscow") // UTC+3, без перехода на летнее время

    // 14 сентября 2026 — понедельник; 19-е — суббота, 20-е — воскресенье.
    private val mondayNoon = Instant.parse("2026-09-14T09:00:00Z")
    private val saturdayNoon = Instant.parse("2026-09-19T09:00:00Z")
    private val sundayNoon = Instant.parse("2026-09-20T09:00:00Z")

    private fun TestScope.store(
        now: Instant,
        slots: RecordingSlots = RecordingSlots(),
        instructors: CountingInstructors = CountingInstructors(),
    ) = SlotListStore(slots, instructors, backgroundScope, now = { now }, zone = { moscow })

    /** Полночь по Москве, выраженная в UTC. */
    private fun moscowMidnight(date: String) = Instant.parse("${date}T00:00:00+03:00")

    private fun TestScope.draftDatesFor(preset: SlotDatePreset, now: Instant): SlotFilters {
        val store = store(now)
        store.accept(SlotListIntent.SelectDatePreset(preset))
        return store.state.value.draftFilters
    }

    @Test
    fun todayCoversCalendarDayInUserTimeZone() = runTest {
        val filters = draftDatesFor(SlotDatePreset.Today, mondayNoon)
        assertEquals(moscowMidnight("2026-09-14"), filters.dateFrom)
        assertEquals(moscowMidnight("2026-09-15"), filters.dateTo)
    }

    @Test
    fun todayUsesLocalDateNotUtcDate() = runTest {
        // 22:30 UTC 14-го — это уже 01:30 15-го по Москве: «сегодня» — 15 сентября.
        val filters = draftDatesFor(SlotDatePreset.Today, Instant.parse("2026-09-14T22:30:00Z"))
        assertEquals(moscowMidnight("2026-09-15"), filters.dateFrom)
        assertEquals(moscowMidnight("2026-09-16"), filters.dateTo)
    }

    @Test
    fun thisWeekRunsUntilEndOfSunday() = runTest {
        val fromMonday = draftDatesFor(SlotDatePreset.NextSevenDays, mondayNoon)
        assertEquals(moscowMidnight("2026-09-14"), fromMonday.dateFrom)
        assertEquals(moscowMidnight("2026-09-21"), fromMonday.dateTo)

        val fromSunday = draftDatesFor(SlotDatePreset.NextSevenDays, sundayNoon)
        assertEquals(moscowMidnight("2026-09-20"), fromSunday.dateFrom)
        assertEquals(moscowMidnight("2026-09-21"), fromSunday.dateTo)
    }

    @Test
    fun weekendIsUpcomingSaturdayAndSunday() = runTest {
        val fromMonday = draftDatesFor(SlotDatePreset.Weekend, mondayNoon)
        assertEquals(moscowMidnight("2026-09-19"), fromMonday.dateFrom)
        assertEquals(moscowMidnight("2026-09-21"), fromMonday.dateTo)

        val fromSaturday = draftDatesFor(SlotDatePreset.Weekend, saturdayNoon)
        assertEquals(moscowMidnight("2026-09-19"), fromSaturday.dateFrom)
        assertEquals(moscowMidnight("2026-09-21"), fromSaturday.dateTo)

        // В воскресенье выходные — это только оставшееся воскресенье, а не следующая суббота.
        val fromSunday = draftDatesFor(SlotDatePreset.Weekend, sundayNoon)
        assertEquals(moscowMidnight("2026-09-20"), fromSunday.dateFrom)
        assertEquals(moscowMidnight("2026-09-21"), fromSunday.dateTo)
    }

    @Test
    fun anyPresetClearsDates() = runTest {
        val store = store(mondayNoon)
        store.accept(SlotListIntent.SelectDatePreset(SlotDatePreset.Weekend))
        store.accept(SlotListIntent.SelectDatePreset(SlotDatePreset.Any))
        assertEquals(null, store.state.value.draftFilters.dateFrom)
        assertEquals(null, store.state.value.draftFilters.dateTo)
    }

    @Test
    fun draftChangesApplyOnlyOnApply() = runTest {
        val slots = RecordingSlots()
        val store = store(mondayNoon, slots)
        store.accept(SlotListIntent.Load)
        runCurrent()

        store.accept(SlotListIntent.OpenFilters)
        store.accept(SlotListIntent.ToggleRouteType(RouteType.Experienced))
        store.accept(SlotListIntent.ToggleOnlyAvailable)
        store.accept(SlotListIntent.CloseFilters)
        runCurrent()
        // Закрыли шторку без «Применить» — каталог не перезапрашивается и фильтр не меняется.
        assertEquals(SlotFilters(), store.state.value.filters)
        assertEquals(1, slots.requests.size)

        store.accept(SlotListIntent.OpenFilters)
        // Черновик при открытии — это текущие фильтры, а не брошенные правки.
        assertEquals(SlotFilters(), store.state.value.draftFilters)
        store.accept(SlotListIntent.ToggleRouteType(RouteType.Experienced))
        store.accept(SlotListIntent.SelectDatePreset(SlotDatePreset.Weekend))
        store.accept(SlotListIntent.ApplyFilters)
        runCurrent()

        val applied = store.state.value.filters
        assertEquals(setOf(RouteType.Experienced), applied.routeTypes)
        assertEquals(SlotDatePreset.Weekend, store.state.value.datePreset)
        assertEquals(false, store.state.value.filtersVisible)
        assertEquals(2, slots.requests.size)
        assertEquals(setOf(RouteType.Experienced), slots.requests.last().routeTypes)
        assertEquals(moscowMidnight("2026-09-19"), slots.requests.last().dateFrom)
    }

    @Test
    fun resetClearsDraftButNotAppliedFilters() = runTest {
        val store = store(mondayNoon)
        store.accept(SlotListIntent.OpenFilters)
        store.accept(SlotListIntent.ToggleRouteType(RouteType.Novice))
        store.accept(SlotListIntent.ApplyFilters)
        runCurrent()

        store.accept(SlotListIntent.OpenFilters)
        store.accept(SlotListIntent.ResetFilters)

        assertEquals(SlotFilters(), store.state.value.draftFilters)
        assertEquals(SlotDatePreset.Any, store.state.value.draftDatePreset)
        assertEquals(setOf(RouteType.Novice), store.state.value.filters.routeTypes)
    }

    @Test
    fun togglesAddAndRemove() = runTest {
        val store = store(mondayNoon)
        val marat = InstructorId("marat")
        store.accept(SlotListIntent.ToggleInstructor(marat))
        store.accept(SlotListIntent.ToggleRouteType(RouteType.Novice))
        assertEquals(setOf(marat), store.state.value.draftFilters.instructorIds)
        store.accept(SlotListIntent.ToggleInstructor(marat))
        store.accept(SlotListIntent.ToggleRouteType(RouteType.Novice))
        assertEquals(SlotFilters(), store.state.value.draftFilters)
    }

    @Test
    fun emptyResultExplainsWhetherFiltersAreToBlame() = runTest {
        val store = store(mondayNoon)
        store.accept(SlotListIntent.Load)
        runCurrent()
        assertEquals(Loadable.Empty(EmptyReason.NoSlots), store.state.value.slots)

        store.accept(SlotListIntent.OpenFilters)
        store.accept(SlotListIntent.ToggleOnlyAvailable)
        store.accept(SlotListIntent.ApplyFilters)
        runCurrent()
        assertEquals(Loadable.Empty(EmptyReason.NoSlotsByFilters), store.state.value.slots)
    }

    @Test
    fun failureShowsErrorAndUnauthorizedSignsOut() = runTest {
        val slots = RecordingSlots(result = Result.failure(AppFailureException(AppFailure.NetworkUnavailable)))
        val store = store(mondayNoon, slots)
        store.accept(SlotListIntent.Load)
        runCurrent()
        assertEquals(Loadable.Error(AppFailure.NetworkUnavailable), store.state.value.slots)

        slots.result = Result.failure(AppFailureException(AppFailure.Unauthorized))
        store.accept(SlotListIntent.Retry)
        runCurrent()
        assertEquals(SlotListEffect.SignedOut, store.effects())
    }

    @Test
    fun instructorsLoadOnceAndRetryOnDemand() = runTest {
        val instructors = CountingInstructors(result = Result.failure(AppFailureException(AppFailure.NetworkUnavailable)))
        val store = store(mondayNoon, instructors = instructors)

        store.accept(SlotListIntent.OpenFilters)
        runCurrent()
        assertIs<Loadable.Error>(store.state.value.instructors)

        instructors.result = Result.success(Page(listOf(Instructor(InstructorId("marat"), "Марат")), 100, 0, 1))
        store.accept(SlotListIntent.RetryInstructors)
        runCurrent()
        assertIs<Loadable.Content<List<Instructor>>>(store.state.value.instructors)

        // Повторное открытие шторки не перезапрашивает справочник.
        store.accept(SlotListIntent.CloseFilters)
        store.accept(SlotListIntent.OpenFilters)
        runCurrent()
        assertEquals(2, instructors.calls)
    }

    class RecordingSlots(
        var result: Result<Page<Slot>> = Result.success(Page(emptyList(), 20, 0, 0)),
    ) : SlotRepository {
        val requests = mutableListOf<SlotFilters>()

        override suspend fun listSlots(filters: SlotFilters, page: PageRequest): Result<Page<Slot>> {
            requests += filters
            return result
        }

        override suspend fun getSlot(slotId: SlotId): Result<Slot> = error("not used")
        override suspend fun leaderboard(routeId: RouteId): Result<List<LeaderboardEntry>> = error("not used")
        override suspend fun trackPassport(routeId: RouteId): Result<TrackPassport> = error("not used")
    }

    class CountingInstructors(
        var result: Result<Page<Instructor>> = Result.success(Page(emptyList(), 100, 0, 0)),
    ) : InstructorRepository {
        var calls = 0

        override suspend fun listInstructors(page: PageRequest): Result<Page<Instructor>> {
            calls++
            return result
        }
    }
}
