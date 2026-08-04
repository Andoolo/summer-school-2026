package com.volna.app.core.storage

import kotlinx.browser.localStorage

actual object PlatformMarshalStorage : MarshalTokenStorage {
    actual override suspend fun readToken(): String? =
        localStorage.getItem(KEY_MARSHAL_TOKEN)

    actual override suspend fun writeToken(token: String) {
        localStorage.setItem(KEY_MARSHAL_TOKEN, token)
    }

    actual override suspend fun clearToken() {
        localStorage.removeItem(KEY_MARSHAL_TOKEN)
    }

    // Ключ отличается от ключа сессии гонщика: выход из аккаунта не должен
    // сбрасывать настройку рабочего места маршала.
    private const val KEY_MARSHAL_TOKEN = "apex_marshal_token"
}
