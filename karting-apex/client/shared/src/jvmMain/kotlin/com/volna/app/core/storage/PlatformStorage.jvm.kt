package com.volna.app.core.storage

/**
 * Хранилища в памяти: JVM-цель нужна только для тестов, переживать перезапуск
 * процесса тут нечему. Настоящие реализации — в androidMain, iosMain и wasmJsMain.
 */
actual object PlatformSessionStorage : SessionStorage {
    private var token: String? = null

    actual override suspend fun readToken(): String? = token

    actual override suspend fun writeToken(token: String) {
        this.token = token
    }

    actual override suspend fun clearToken() {
        token = null
    }
}

actual object PlatformMarshalStorage : MarshalTokenStorage {
    private var token: String? = null

    actual override suspend fun readToken(): String? = token

    actual override suspend fun writeToken(token: String) {
        this.token = token
    }

    actual override suspend fun clearToken() {
        token = null
    }
}

actual object PlatformThemeStorage : ThemePreferenceStorage {
    private var value: String? = null

    actual override fun read(): String? = value

    actual override fun write(value: String) {
        this.value = value
    }
}
