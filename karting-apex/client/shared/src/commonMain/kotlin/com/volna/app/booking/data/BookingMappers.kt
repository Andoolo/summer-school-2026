package com.volna.app.booking.data

import com.volna.app.catalog.Page
import com.volna.app.catalog.data.toDomain
import com.volna.app.domain.model.Booking
import com.volna.app.domain.model.BookingId
import com.volna.app.domain.model.BookingStatus
import com.volna.app.domain.model.ClientId
import com.volna.app.domain.model.MoneyRub
import com.volna.app.domain.model.SlotId

fun BookingListResponseDto.toDomain(): Page<Booking> = Page(
    items = items.map { it.toDomain() },
    limit = meta.limit,
    offset = meta.offset,
    total = meta.total,
)

fun BookingDto.toDomain(): Booking = Booking(
    id = BookingId(id),
    slotId = SlotId(slotId),
    clientId = clientId?.let(::ClientId),
    seatsCount = seatsCount,
    rentalCount = rentalCount,
    status = status.toBookingStatus(),
    priceTotal = priceTotal?.let(::MoneyRub),
    createdAt = createdAt,
    cancelledAt = cancelledAt,
    slot = slot?.toDomain(),
    isFirstBooking = isFirstBooking,
)

fun BookingStatus.toApiValue(): String = when (this) {
    BookingStatus.Active -> "active"
    BookingStatus.Cancelled -> "cancelled"
    BookingStatus.LateCancel -> "late_cancel"
}

private fun String.toBookingStatus(): BookingStatus = when (this) {
    "cancelled" -> BookingStatus.Cancelled
    "late_cancel" -> BookingStatus.LateCancel
    else -> BookingStatus.Active
}
