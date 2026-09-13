package com.volna.app.core.ui

import kotlin.test.Test
import kotlin.test.assertEquals

class SelectionSemanticsTest {
    @Test
    fun `состояние выбора озвучивается по-русски`() {
        assertEquals("выбрано", selectedStateText(true))
        assertEquals("не выбрано", selectedStateText(false))
    }

    @Test
    fun `состояние переключателя озвучивается по-русски`() {
        assertEquals("включено", switchStateText(true))
        assertEquals("выключено", switchStateText(false))
    }

    @Test
    fun `вне веба подпись не дублирует системное объявление`() {
        // commonTest исполняется на JVM-таргете: там, как на Android и iOS, подпись
        // остаётся без суффикса — состояние объявляет сама система.
        assertEquals("Заезды", selectionLabel("Заезды", selected = true))
    }
}
