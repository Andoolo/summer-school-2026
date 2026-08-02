package com.volna.app.booking.data

import com.volna.app.catalog.data.PaginationMetaDto
import com.volna.app.catalog.data.SlotDto
import kotlinx.datetime.Instant
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class CreateBookingRequestDto(
    @SerialName("slot_id")
    val slotId: String,
    @SerialName("seats_count")
    val seatsCount: Int,
    @SerialName("rental_count")
    val rentalCount: Int,
)

@Serializable
data class BookingListResponseDto(
    val items: List<BookingDto>,
    val meta: PaginationMetaDto,
)

@Serializable
data class BookingDto(
    val id: String,
    @SerialName("slot_id")
    val slotId: String,
    @SerialName("client_id")
    val clientId: String? = null,
    @SerialName("seats_count")
    val seatsCount: Int,
    @SerialName("rental_count")
    val rentalCount: Int,
    val status: String,
    @SerialName("price_total")
    val priceTotal: Int? = null,
    @SerialName("created_at")
    val createdAt: Instant,
    @SerialName("cancelled_at")
    val cancelledAt: Instant? = null,
    val slot: SlotDto? = null,
    @SerialName("is_first_booking")
    val isFirstBooking: Boolean? = null,
)
