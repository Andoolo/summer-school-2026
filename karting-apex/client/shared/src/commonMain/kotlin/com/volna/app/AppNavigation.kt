package com.volna.app

import com.volna.app.domain.model.BookingId
import com.volna.app.domain.model.SlotId
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

internal enum class MainTab(val title: String) {
    Slots("Заезды"),
    Bookings("Мои записи"),
    Profile("Профиль"),
}

internal enum class RootState {
    CheckingSession,
    Ready,
}

@Serializable
@SerialName("auth")
internal data object AuthDestination

@Serializable
@SerialName("slots")
internal data object SlotsDestination

@Serializable
@SerialName("slot")
internal data class SlotDetailsDestination(val slotId: String)

@Serializable
@SerialName("slot-booking")
internal data class SlotBookingDestination(val slotId: String)

@Serializable
@SerialName("bookings")
internal data object BookingsDestination

@Serializable
@SerialName("booking")
internal data class BookingDetailsDestination(val bookingId: String)

@Serializable
@SerialName("profile")
internal data object ProfileDestination

internal fun SlotDetailsDestination.slotId(): SlotId = SlotId(slotId)

internal fun SlotBookingDestination.slotId(): SlotId = SlotId(slotId)

internal fun BookingDetailsDestination.bookingId(): BookingId = BookingId(bookingId)

internal fun MainTab.destination(): Any = when (this) {
    MainTab.Slots -> SlotsDestination
    MainTab.Bookings -> BookingsDestination
    MainTab.Profile -> ProfileDestination
}
