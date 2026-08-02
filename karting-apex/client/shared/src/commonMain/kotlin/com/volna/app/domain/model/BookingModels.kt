package com.volna.app.domain.model

import kotlinx.datetime.Instant

enum class BookingStatus {
    Active,
    Cancelled,
    LateCancel,
}

data class Booking(
    val id: BookingId,
    val slotId: SlotId,
    val clientId: ClientId?,
    val seatsCount: Int,
    val rentalCount: Int,
    val status: BookingStatus,
    val priceTotal: MoneyRub?,
    val createdAt: Instant,
    val cancelledAt: Instant?,
    val slot: Slot?,
    val isFirstBooking: Boolean?,
)

data class BookingDraft(
    val slot: Slot,
    val seatsCount: Int,
    val rentalCount: Int,
)
