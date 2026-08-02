package com.volna.app.booking.presentation

import com.volna.app.booking.BookingRepository
import com.volna.app.booking.IdempotencyKey
import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.core.time.AppClock
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.Booking
import com.volna.app.domain.model.BookingDraft
import com.volna.app.domain.model.BookingId
import com.volna.app.domain.model.BookingStatus
import com.volna.app.domain.model.GeoPoint
import com.volna.app.domain.model.Instructor
import com.volna.app.domain.model.InstructorId
import com.volna.app.domain.model.MeetingPoint
import com.volna.app.domain.model.MoneyRub
import com.volna.app.domain.model.Route
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.model.SlotStatus
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.yield
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertIs

class BookingDetailsStoreTest {
    @Test
    fun slotStartedCancelFailureClosesSheetAndRefreshesBooking() = runTest {
        val booking = booking(
            status = BookingStatus.Active,
            slotStartAt = Instant.parse("2026-07-01T12:00:00Z"),
        )
        val refreshed = booking.copy(slot = booking.slot?.copy(startAt = Instant.parse("2026-06-01T12:00:00Z")))
        val repository = FakeBookingRepository(
            initialBooking = booking,
            refreshedBooking = refreshed,
            cancelResult = Result.failure(
                AppFailureException(
                    AppFailure.Api(
                        code = ApiErrorCode.SlotStarted,
                        message = "backend message",
                    ),
                ),
            ),
        )
        val store = BookingDetailsStore(
            bookingRepository = repository,
            clock = AppClock { Instant.parse("2026-06-01T10:00:00Z") },
            scope = CoroutineScope(coroutineContext),
        )

        store.accept(BookingDetailsIntent.Load(booking.id))
        yield()
        store.accept(BookingDetailsIntent.AskCancel)
        store.accept(BookingDetailsIntent.ConfirmCancel)
        yield()
        yield()

        val state = store.state.value
        val content = assertIs<Loadable.Content<Booking>>(state.booking)
        assertEquals(refreshed.slot?.startAt, content.value.slot?.startAt)
        assertFalse(state.showCancelConfirm)
        assertEquals("Слот уже стартовал — отмена недоступна.", state.message)
        assertEquals(1, repository.cancelCalls)
        assertEquals(2, repository.getCalls)
    }

    private class FakeBookingRepository(
        private val initialBooking: Booking,
        private val refreshedBooking: Booking,
        private val cancelResult: Result<Booking>,
    ) : BookingRepository {
        var getCalls: Int = 0
            private set
        var cancelCalls: Int = 0
            private set

        override suspend fun createBooking(draft: BookingDraft, idempotencyKey: IdempotencyKey): Result<Booking> =
            Result.failure(UnsupportedOperationException())

        override suspend fun listBookings(status: BookingStatus?, page: PageRequest): Result<Page<Booking>> =
            Result.failure(UnsupportedOperationException())

        override suspend fun getBooking(bookingId: BookingId): Result<Booking> {
            getCalls += 1
            return Result.success(if (getCalls == 1) initialBooking else refreshedBooking)
        }

        override suspend fun cancelBooking(bookingId: BookingId): Result<Booking> {
            cancelCalls += 1
            return cancelResult
        }
    }

    private fun booking(
        status: BookingStatus,
        slotStartAt: Instant,
    ): Booking = Booking(
        id = BookingId("booking-1"),
        slotId = SlotId("slot-1"),
        clientId = null,
        seatsCount = 1,
        rentalCount = 0,
        status = status,
        priceTotal = null,
        createdAt = Instant.parse("2026-06-01T09:00:00Z"),
        cancelledAt = null,
        slot = Slot(
            id = SlotId("slot-1"),
            startAt = slotStartAt,
            route = Route(
                id = RouteId("route-1"),
                name = "Острова и каналы",
                type = RouteType.Novice,
                capacityCap = 8,
                durationMin = 90,
            ),
            instructor = Instructor(InstructorId("instructor-1"), "Мария"),
            totalSeats = 8,
            freeSeats = 4,
            freeRentalBoards = 4,
            price = MoneyRub(2_500),
            rentalPrice = MoneyRub(800),
            meetingPoint = MeetingPoint("Лодочная станция", GeoPoint(59.978, 30.262)),
            status = SlotStatus.Scheduled,
        ),
        isFirstBooking = null,
    )
}
