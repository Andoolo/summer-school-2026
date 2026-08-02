package com.volna.app.core.logging

actual object AppLogger {
    actual fun d(message: String) {
        println("D: $message")
    }

    actual fun e(throwable: Throwable?, message: String) {
        println("E: $message ${throwable?.message.orEmpty()}")
    }
}
