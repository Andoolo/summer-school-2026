package com.volna.app.core.storage

import platform.Foundation.NSUserDefaults

actual object PlatformMarshalStorage : MarshalTokenStorage {
    private val defaults: NSUserDefaults
        get() = NSUserDefaults.standardUserDefaults

    actual override suspend fun readToken(): String? =
        defaults.stringForKey(KEY_MARSHAL_TOKEN)

    actual override suspend fun writeToken(token: String) {
        defaults.setObject(token, forKey = KEY_MARSHAL_TOKEN)
    }

    actual override suspend fun clearToken() {
        defaults.removeObjectForKey(KEY_MARSHAL_TOKEN)
    }

    // Ключ отличается от ключа сессии гонщика: выход из аккаунта не должен
    // сбрасывать настройку рабочего места маршала.
    private const val KEY_MARSHAL_TOKEN = "apex_marshal_token"
}
