package com.volna.app.core.logging

expect object AppLogger {
    fun d(message: String)
    fun e(throwable: Throwable?, message: String)
}
