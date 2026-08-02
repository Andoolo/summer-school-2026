package com.volna.app.booking

import com.volna.app.catalog.Page
import com.volna.app.catalog.PageRequest
import com.volna.app.domain.model.Booking
import com.volna.app.domain.model.BookingDraft
import com.volna.app.domain.model.BookingId
import com.volna.app.domain.model.BookingStatus
import kotlin.jvm.JvmInline

@JvmInline
value class IdempotencyKey(val value: String)

interface IdempotencyKeyFactory {
    fun next(): IdempotencyKey
}

interface BookingRepository {
    suspend fun createBooking(draft: BookingDraft, idempotencyKey: IdempotencyKey): Result<Booking>
    suspend fun listBookings(status: BookingStatus? = null, page: PageRequest = PageRequest()): Result<Page<Booking>>
    suspend fun getBooking(bookingId: BookingId): Result<Booking>
    suspend fun cancelBooking(bookingId: BookingId): Result<Booking>
}
