package com.volna.app.core.error

data class ErrorDetails(
    val availableSeats: Int? = null,
    val availableRentalBoards: Int? = null,
)

enum class ApiErrorCode {
    BadRequest,
    Unauthorized,
    Forbidden,
    NotFound,
    SlotFull,
    DoubleBooking,
    SlotCancelled,
    SlotStarted,
    AlreadyCancelled,
    InvalidCode,
    IdempotencyConflict,
    PhoneConflict,
    TooManyRequests,
    InternalError,
    Unknown,
}

sealed interface AppFailure {
    data object Unauthorized : AppFailure
    data object NetworkUnavailable : AppFailure
    data object Timeout : AppFailure
    data object Unknown : AppFailure
    data class Api(
        val code: ApiErrorCode,
        val message: String,
        val details: ErrorDetails? = null,
    ) : AppFailure
}
