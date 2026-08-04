package com.volna.app.marshal

import com.volna.app.marshal.presentation.formatLapTime
import com.volna.app.marshal.presentation.parseLapTime
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class LapTimeInputTest {

    @Test
    fun `разбирает секунды с миллисекундами`() {
        assertEquals(41890, parseLapTime("41.890"))
        assertEquals(9500, parseLapTime("9.5"))
        assertEquals(42000, parseLapTime("42"))
    }

    @Test
    fun `разбирает запись с минутами`() {
        assertEquals(61234, parseLapTime("1:01.234"))
        assertEquals(60000, parseLapTime("1:00"))
        assertEquals(125500, parseLapTime("2:05.5"))
    }

    @Test
    fun `запятая работает как разделитель дроби`() {
        // На цифровой клавиатуре под рукой обычно запятая, а не точка.
        assertEquals(parseLapTime("41.890"), parseLapTime("41,890"))
    }

    @Test
    fun `дробная часть дополняется нулями справа`() {
        // «41.5» — это пятьсот миллисекунд, а не пять: иначе маршал внёс бы
        // время в сто раз меньше и получил бы фиктивный рекорд трассы.
        assertEquals(41500, parseLapTime("41.5"))
        assertEquals(41050, parseLapTime("41.05"))
        assertEquals(41005, parseLapTime("41.005"))
    }

    @Test
    fun `лишние пробелы не мешают`() {
        assertEquals(41890, parseLapTime("  41.890  "))
    }

    @Test
    fun `отвергает то, что не является временем круга`() {
        for (input in listOf("", "   ", "абв", "41.8x", "-41.890", "41.8901", "1:75.000", "::", "1:")) {
            assertNull(parseLapTime(input), "ожидали null для «$input»")
        }
    }

    @Test
    fun `формат и разбор обратны друг другу`() {
        for (ms in listOf(9_500, 41_890, 59_999, 60_000, 61_234, 125_500, 599_999)) {
            assertEquals(ms, parseLapTime(formatLapTime(ms)), "не сошлось на $ms мс")
        }
    }

    @Test
    fun `формат показывает минуты только когда они есть`() {
        assertEquals("41.890", formatLapTime(41_890))
        assertEquals("1:01.234", formatLapTime(61_234))
        assertEquals("2:05.500", formatLapTime(125_500))
    }
}
