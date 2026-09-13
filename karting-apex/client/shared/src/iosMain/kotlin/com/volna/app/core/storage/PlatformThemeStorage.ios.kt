package com.volna.app.core.storage

import platform.Foundation.NSUserDefaults

actual object PlatformThemeStorage : ThemePreferenceStorage {
    actual override fun read(): String? =
        NSUserDefaults.standardUserDefaults.stringForKey(KEY_THEME)

    actual override fun write(value: String) {
        NSUserDefaults.standardUserDefaults.setObject(value, forKey = KEY_THEME)
    }

    private const val KEY_THEME = "apex_theme"
}
