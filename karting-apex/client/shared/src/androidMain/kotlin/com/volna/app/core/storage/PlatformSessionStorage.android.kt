package com.volna.app.core.storage

import android.content.Context
import android.content.SharedPreferences

actual object PlatformSessionStorage : SessionStorage {
    private var preferences: SharedPreferences? = null
    private var fallbackToken: String? = null

    fun initialize(context: Context) {
        preferences = context.applicationContext.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)
        fallbackToken?.let { token ->
            preferences?.edit()?.putString(KEY_TOKEN, token)?.apply()
            fallbackToken = null
        }
    }

    actual override suspend fun readToken(): String? =
        preferences?.getString(KEY_TOKEN, null) ?: fallbackToken

    actual override suspend fun writeToken(token: String) {
        val currentPreferences = preferences
        if (currentPreferences == null) {
            fallbackToken = token
        } else {
            currentPreferences.edit().putString(KEY_TOKEN, token).apply()
        }
    }

    actual override suspend fun clearToken() {
        fallbackToken = null
        preferences?.edit()?.remove(KEY_TOKEN)?.apply()
    }

    private const val PREFERENCES_NAME = "volna_session"
    private const val KEY_TOKEN = "bearer_token"
}
