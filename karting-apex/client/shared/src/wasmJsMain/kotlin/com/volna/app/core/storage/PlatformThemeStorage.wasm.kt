package com.volna.app.core.storage

import kotlinx.browser.localStorage

actual object PlatformThemeStorage : ThemePreferenceStorage {
    actual override fun read(): String? = localStorage.getItem(KEY_THEME)

    actual override fun write(value: String) {
        localStorage.setItem(KEY_THEME, value)
    }

    // Этот же ключ читает прелоадер в index.html, чтобы не показывать белый экран
    // тем, кто выбрал тёмную тему. Меняя ключ, поменяй и там.
    private const val KEY_THEME = "apex_theme"
}
