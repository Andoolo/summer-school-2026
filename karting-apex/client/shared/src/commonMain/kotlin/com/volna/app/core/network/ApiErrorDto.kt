package com.volna.app.core.network

import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.ErrorDetails
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ApiErrorDto(
    val code: String,
    val message: String,
    val details: ApiErrorDetailsDto? = null,
)

@Serializable
data class ApiErrorDetailsDto(
    @SerialName("available_seats")
    val availableSeats: Int? = null,
    @SerialName("available_rental_boards")
    val availableRentalBoards: Int? = null,
)

fun ApiErrorDto.toFailure(): AppFailure.Api = AppFailure.Api(
    code = code.toApiErrorCode(),
    message = message,
    details = details?.let {
        ErrorDetails(
            availableSeats = it.availableSeats,
            availableRentalBoards = it.availableRentalBoards,
        )
    },
)

private fun String.toApiErrorCode(): ApiErrorCode = when (this) {
    "bad_request" -> ApiErrorCode.BadRequest
    "unauthorized" -> ApiErrorCode.Unauthorized
    "forbidden" -> ApiErrorCode.Forbidden
    "not_found" -> ApiErrorCode.NotFound
    "slot_full" -> ApiErrorCode.SlotFull
    "double_booking" -> ApiErrorCode.DoubleBooking
    "slot_cancelled" -> ApiErrorCode.SlotCancelled
    "slot_started" -> ApiErrorCode.SlotStarted
    "already_cancelled" -> ApiErrorCode.AlreadyCancelled
    "invalid_code" -> ApiErrorCode.InvalidCode
    "idempotency_conflict" -> ApiErrorCode.IdempotencyConflict
    "phone_conflict" -> ApiErrorCode.PhoneConflict
    "too_many_requests" -> ApiErrorCode.TooManyRequests
    "internal_error" -> ApiErrorCode.InternalError
    else -> ApiErrorCode.Unknown
}
