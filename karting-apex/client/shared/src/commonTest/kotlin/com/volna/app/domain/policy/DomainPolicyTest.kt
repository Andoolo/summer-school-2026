package com.volna.app.domain.policy

import com.volna.app.domain.model.Booking
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
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.time.Duration.Companion.hours
import kotlin.time.Duration.Companion.seconds

class DomainPolicyTest {
    @Test
    fun availabilityIsLimitedByFreeSeatsRouteCapAndClientMaximum() {
        val slot = slot(freeSeats = 10, capacityCap = 8, freeRentalBoards = 12)

        val availability = AvailabilityPolicy.availability(slot)

        assertEquals(3, availability.maxSeatsForBooking)
        assertEquals(12, availability.freeRentalBoards)
    }

    @Test
    fun rentalBoardsCannotExceedFreeRentalBoards() {
        val draft = com.volna.app.domain.model.BookingDraft(
            slot = slot(freeSeats = 3, capacityCap = 8, freeRentalBoards = 1),
            seatsCount = 2,
            rentalCount = 2,
        )

        assertEquals(
            AvailabilityViolation.TooManyRentalBoards(1),
            AvailabilityPolicy.validate(draft),
        )
    }

    @Test
    fun validDraftHasNoAvailabilityViolation() {
        val draft = com.volna.app.domain.model.BookingDraft(
            slot = slot(freeSeats = 3, capacityCap = 8, freeRentalBoards = 2),
            seatsCount = 2,
            rentalCount = 1,
        )

        assertNull(AvailabilityPolicy.validate(draft))
    }

    @Test
    fun bookingPriceUsesSeatAndRentalPrices() {
        val total = BookingPriceCalculator.calculate(
            slot = slot(price = 2_500, rentalPrice = 800),
            seatsCount = 2,
            rentalCount = 1,
        )

        assertEquals(MoneyRub(5_800), total)
    }

    @Test
    fun bookingPriceForExistingBookingIsDerivedFromSlotPrices() {
        val booking = Booking(
            id = BookingId("booking-1"),
            slotId = SlotId("slot-1"),
            clientId = null,
            seatsCount = 3,
            rentalCount = 2,
            status = BookingStatus.Active,
            priceTotal = MoneyRub(1),
            createdAt = Instant.parse("2026-06-01T12:00:00Z"),
            cancelledAt = null,
            slot = slot(price = 2_000, rentalPrice = 500),
            isFirstBooking = null,
        )

        assertEquals(MoneyRub(7_000), BookingPriceCalculator.calculate(booking))
    }

    @Test
    fun exactlyTwoHoursBeforeStartIsEarlyCancellation() {
        val startAt = Instant.parse("2026-07-01T12:00:00Z")

        assertEquals(CancellationKind.Early, CancellationPolicy.classify(startAt - 2.hours, startAt))
        assertEquals(CancellationKind.Early, CancellationPolicy.classify(startAt - 2.hours - 1.seconds, startAt))
        assertEquals(CancellationKind.Late, CancellationPolicy.classify(startAt - 2.hours + 1.seconds, startAt))
        assertEquals(CancellationKind.UnavailableAfterStart, CancellationPolicy.classify(startAt, startAt))
    }

    private fun slot(
        freeSeats: Int = 5,
        capacityCap: Int = 8,
        freeRentalBoards: Int = 5,
        price: Int = 2_500,
        rentalPrice: Int = 800,
    ): Slot = Slot(
        id = SlotId("slot-1"),
        startAt = Instant.parse("2026-07-01T12:00:00Z"),
        route = Route(
            id = RouteId("route-1"),
            name = "Острова и каналы",
            type = RouteType.Novice,
            capacityCap = capacityCap,
            durationMin = 90,
        ),
        instructor = Instructor(InstructorId("instructor-1"), "Мария"),
        totalSeats = capacityCap,
        freeSeats = freeSeats,
        freeRentalBoards = freeRentalBoards,
        price = MoneyRub(price),
        rentalPrice = MoneyRub(rentalPrice),
        meetingPoint = MeetingPoint("Лодочная станция", GeoPoint(59.978, 30.262)),
        status = SlotStatus.Scheduled,
    )
}
