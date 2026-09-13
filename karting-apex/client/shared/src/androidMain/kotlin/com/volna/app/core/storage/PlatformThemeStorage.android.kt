package com.volna.app.core.storage

import android.content.Context
import android.content.SharedPreferences

actual object PlatformThemeStorage : ThemePreferenceStorage {
    private var preferences: SharedPreferences? = null

    fun initialize(context: Context) {
        preferences = context.applicationContext.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)
    }

    // До initialize читать неоткуда — будет системная тема, это безопасный откат.
    actual override fun read(): String? = preferences?.getString(KEY_THEME, null)

    actual override fun write(value: String) {
        preferences?.edit()?.putString(KEY_THEME, value)?.apply()
    }

    private const val PREFERENCES_NAME = "apex_settings"
    private const val KEY_THEME = "theme"
}
