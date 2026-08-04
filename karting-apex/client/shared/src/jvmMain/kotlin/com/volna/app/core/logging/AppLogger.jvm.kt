package com.volna.app.core.logging

/**
 * JVM-цель существует только ради запуска общих тестов (см. shared/build.gradle.kts),
 * приложение на ней не собирается. Поэтому реализации здесь — минимальные заглушки,
 * достаточные для того, чтобы тестируемый код работал.
 */
actual object AppLogger {
    actual fun d(message: String) {
        println("D: $message")
    }

    actual fun e(throwable: Throwable?, message: String) {
        println("E: $message${throwable?.let { " (${it.message})" }.orEmpty()}")
    }
}
