package com.volna.app.core.error

class AppFailureException(
    val failure: AppFailure,
) : RuntimeException(failure.toString())

fun Throwable.asAppFailure(): AppFailure = when (this) {
    is AppFailureException -> failure
    else -> AppFailure.Unknown
}
