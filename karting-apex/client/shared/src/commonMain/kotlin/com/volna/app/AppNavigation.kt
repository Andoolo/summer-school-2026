package com.volna.app

import androidx.navigation.NavBackStackEntry
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.toRoute
import com.volna.app.domain.model.BookingId
import com.volna.app.domain.model.RouteId
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
@SerialName("track")
internal data class TrackDestination(val routeId: String)

@Serializable
@SerialName("marshal")
internal data object MarshalDestination

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

internal fun TrackDestination.routeId(): RouteId = RouteId(routeId)

internal fun BookingDetailsDestination.bookingId(): BookingId = BookingId(bookingId)

internal fun MainTab.destination(): Any = when (this) {
    MainTab.Slots -> SlotsDestination
    MainTab.Bookings -> BookingsDestination
    MainTab.Profile -> ProfileDestination
}

/**
 * Экран, открытый прямой ссылкой (в вебе — адресом вида #slot/<id>, например из сообщения
 * бота), и вкладка, которая должна лежать под ним. null — это не такой экран.
 *
 * Без вкладки под ним «Назад» с такого экрана вёл на стартовый экран входа.
 */
internal data class DeepLink(val target: Any, val tab: Any)

internal fun NavBackStackEntry.deepLink(): DeepLink? = when {
    destination.hasRoute<SlotDetailsDestination>() -> DeepLink(toRoute<SlotDetailsDestination>(), SlotsDestination)
    destination.hasRoute<TrackDestination>() -> DeepLink(toRoute<TrackDestination>(), SlotsDestination)
    destination.hasRoute<BookingDetailsDestination>() -> DeepLink(toRoute<BookingDetailsDestination>(), BookingsDestination)
    else -> null
}
