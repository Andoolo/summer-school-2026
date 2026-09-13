package com.volna.app.core.storage

/**
 * Хранилище выбранной темы оформления.
 *
 * В отличие от токенов, методы синхронные: значение нужно до первого кадра, а все
 * платформенные хранилища (localStorage, SharedPreferences, NSUserDefaults) и так
 * отдают строку мгновенно.
 */
expect object PlatformThemeStorage : ThemePreferenceStorage {
    override fun read(): String?
    override fun write(value: String)
}

interface ThemePreferenceStorage {
    fun read(): String?
    fun write(value: String)
}
