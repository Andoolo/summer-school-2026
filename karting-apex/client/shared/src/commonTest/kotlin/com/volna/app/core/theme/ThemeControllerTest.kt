package com.volna.app.core.theme

import com.volna.app.core.storage.ThemePreferenceStorage
import kotlin.test.Test
import kotlin.test.assertEquals

class ThemeControllerTest {
    private class FakeStorage(var value: String? = null) : ThemePreferenceStorage {
        var writes = 0
        override fun read(): String? = value
        override fun write(value: String) {
            writes++
            this.value = value
        }
    }

    @Test
    fun `без сохранённого выбора тема системная`() {
        assertEquals(ThemeMode.System, ThemeController(FakeStorage()).mode.value)
    }

    @Test
    fun `сохранённый выбор восстанавливается при запуске`() {
        assertEquals(ThemeMode.Dark, ThemeController(FakeStorage("Dark")).mode.value)
    }

    @Test
    fun `неизвестное значение откатывается к системной теме`() {
        assertEquals(ThemeMode.System, ThemeController(FakeStorage("Sepia")).mode.value)
    }

    @Test
    fun `выбор меняет тему и сохраняется`() {
        val storage = FakeStorage()
        val controller = ThemeController(storage)

        controller.select(ThemeMode.Light)

        assertEquals(ThemeMode.Light, controller.mode.value)
        assertEquals("Light", storage.value)
        // Новый экземпляр — как перезапуск приложения.
        assertEquals(ThemeMode.Light, ThemeController(storage).mode.value)
    }

    @Test
    fun `повторный выбор той же темы не пишет в хранилище`() {
        val storage = FakeStorage("Dark")
        val controller = ThemeController(storage)

        controller.select(ThemeMode.Dark)

        assertEquals(0, storage.writes)
    }
}
