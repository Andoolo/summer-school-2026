package com.volna.app.core.storage

import android.content.Context
import android.content.SharedPreferences

actual object PlatformMarshalStorage : MarshalTokenStorage {
    private var preferences: SharedPreferences? = null
    private var fallbackToken: String? = null

    fun initialize(context: Context) {
        preferences = context.applicationContext.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)
        fallbackToken?.let { token ->
            preferences?.edit()?.putString(KEY_MARSHAL_TOKEN, token)?.apply()
            fallbackToken = null
        }
    }

    actual override suspend fun readToken(): String? =
        preferences?.getString(KEY_MARSHAL_TOKEN, null) ?: fallbackToken

    actual override suspend fun writeToken(token: String) {
        val currentPreferences = preferences
        if (currentPreferences == null) {
            fallbackToken = token
        } else {
            currentPreferences.edit().putString(KEY_MARSHAL_TOKEN, token).apply()
        }
    }

    actual override suspend fun clearToken() {
        fallbackToken = null
        preferences?.edit()?.remove(KEY_MARSHAL_TOKEN)?.apply()
    }

    // Отдельное хранилище от сессии гонщика: режим маршала — настройка рабочего
    // места, она переживает выход из аккаунта.
    private const val PREFERENCES_NAME = "apex_marshal"
    private const val KEY_MARSHAL_TOKEN = "marshal_token"
}
