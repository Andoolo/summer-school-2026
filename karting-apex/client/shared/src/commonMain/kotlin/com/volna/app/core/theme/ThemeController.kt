package com.volna.app.core.theme

import com.volna.app.core.storage.ThemePreferenceStorage
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow

enum class ThemeMode {
    /** Как в системе телефона или браузера. Значение по умолчанию. */
    System,
    Light,
    Dark,
}

/**
 * Выбранная пользователем тема оформления.
 *
 * Чтение из хранилища синхронное и происходит при создании: тема нужна уже на
 * первом кадре. Асинхронная загрузка дала бы мигание — первый кадр в системной
 * теме, следующий в выбранной.
 *
 * Это настройка устройства, а не аккаунта: выход из аккаунта её не сбрасывает.
 */
class ThemeController(
    private val storage: ThemePreferenceStorage,
) {
    private val mutableMode = MutableStateFlow(parse(storage.read()))

    val mode: StateFlow<ThemeMode> = mutableMode

    fun select(mode: ThemeMode) {
        if (mode == mutableMode.value) return
        storage.write(mode.name)
        mutableMode.value = mode
    }

    private companion object {
        // Неизвестное значение (например, от будущей версии) — не повод падать:
        // откатываемся к системной теме.
        fun parse(raw: String?): ThemeMode =
            ThemeMode.entries.firstOrNull { it.name == raw } ?: ThemeMode.System
    }
}
