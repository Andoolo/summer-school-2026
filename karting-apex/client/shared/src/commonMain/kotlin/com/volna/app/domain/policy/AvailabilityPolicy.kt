package com.volna.app.domain.policy

import com.volna.app.domain.model.BookingDraft
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotStatus
import kotlin.math.min

data class Availability(
    val maxSeatsForBooking: Int,
    val freeRentalBoards: Int,
    val canBook: Boolean,
)

sealed interface AvailabilityViolation {
    data object SlotCancelled : AvailabilityViolation
    data object NoSeats : AvailabilityViolation
    data class TooManySeats(val maxSeats: Int) : AvailabilityViolation
    data class TooManyRentalBoards(val freeRentalBoards: Int) : AvailabilityViolation
}

object AvailabilityPolicy {
    private const val MaxClientSeats = 3

    fun availability(slot: Slot): Availability {
        val maxSeats = if (slot.status == SlotStatus.Scheduled) {
            min(slot.freeSeats, min(slot.route.capacityCap, MaxClientSeats))
        } else {
            0
        }
        return Availability(
            maxSeatsForBooking = maxSeats,
            freeRentalBoards = slot.freeRentalBoards,
            canBook = maxSeats > 0,
        )
    }

    fun validate(draft: BookingDraft): AvailabilityViolation? {
        val availability = availability(draft.slot)
        return when {
            draft.slot.status != SlotStatus.Scheduled -> AvailabilityViolation.SlotCancelled
            availability.maxSeatsForBooking == 0 -> AvailabilityViolation.NoSeats
            draft.seatsCount !in 1..availability.maxSeatsForBooking ->
                AvailabilityViolation.TooManySeats(availability.maxSeatsForBooking)
            draft.rentalCount !in 0..draft.seatsCount ->
                AvailabilityViolation.TooManyRentalBoards(draft.seatsCount)
            draft.rentalCount > availability.freeRentalBoards ->
                AvailabilityViolation.TooManyRentalBoards(availability.freeRentalBoards)
            else -> null
        }
    }
}
