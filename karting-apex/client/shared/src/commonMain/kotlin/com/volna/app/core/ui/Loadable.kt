package com.volna.app.core.ui

import com.volna.app.core.error.AppFailure

sealed interface Loadable<out T> {
    data object Initial : Loadable<Nothing>
    data object Loading : Loadable<Nothing>
    data class Content<T>(
        val value: T,
        val refreshing: Boolean = false,
    ) : Loadable<T>
    data class Empty(val reason: EmptyReason) : Loadable<Nothing>
    data class Error(val failure: AppFailure) : Loadable<Nothing>
}

enum class ActionStatus {
    Idle,
    Submitting,
}

enum class EmptyReason {
    NoSlots,
    NoSlotsByFilters,
    NoUpcomingBookings,
    NoPastBookings,
    RouteGeometryMissing,
}
