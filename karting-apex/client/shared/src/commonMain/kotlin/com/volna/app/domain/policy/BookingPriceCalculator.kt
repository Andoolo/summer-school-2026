package com.volna.app.domain.policy

import com.volna.app.domain.model.BookingDraft
import com.volna.app.domain.model.Booking
import com.volna.app.domain.model.MoneyRub
import com.volna.app.domain.model.Slot

object BookingPriceCalculator {
    fun calculate(slot: Slot, seatsCount: Int, rentalCount: Int): MoneyRub {
        require(seatsCount in 1..3) { "seatsCount must be in 1..3." }
        require(rentalCount in 0..seatsCount) { "rentalCount must be in 0..seatsCount." }
        return (slot.price * seatsCount) + (slot.rentalPrice * rentalCount)
    }

    fun calculate(draft: BookingDraft): MoneyRub =
        calculate(draft.slot, draft.seatsCount, draft.rentalCount)

    fun calculate(booking: Booking): MoneyRub? =
        booking.slot?.let { calculate(it, booking.seatsCount, booking.rentalCount) } ?: booking.priceTotal
}
