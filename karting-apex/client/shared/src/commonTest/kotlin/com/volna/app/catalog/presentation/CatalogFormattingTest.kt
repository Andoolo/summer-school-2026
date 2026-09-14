package com.volna.app.catalog.presentation

import com.volna.app.domain.model.RouteType
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals

/** Тексты карточек заезда и рекордов: то, что человек читает глазами. */
class CatalogFormattingTest {
    @Test
    fun lapTimeUnderMinuteShowsSecondsAndMillis() {
        assertEquals("41.890", 41890.toLapTimeText())
        assertEquals("56.771", 56771.toLapTimeText())
        // Ведущие нули в миллисекундах: 42.005, а не 42.5 — это разные времена круга.
        assertEquals("42.005", 42005.toLapTimeText())
        assertEquals("59.999", 59999.toLapTimeText())
        assertEquals("0.000", 0.toLapTimeText())
    }

    @Test
    fun lapTimeFromMinuteAddsMinutesAndPadsSeconds() {
        assertEquals("1:00.000", 60000.toLapTimeText())
        assertEquals("1:02.345", 62345.toLapTimeText())
        assertEquals("2:10.001", 130001.toLapTimeText())
    }

    @Test
    fun slotCardDateInUserTimeZone() {
        val moscow = TimeZone.of("Europe/Moscow")
        // 23 сентября 2026 — среда; 07:05 UTC = 10:05 по Москве.
        assertEquals("Ср, 23 сентября · 10:05", Instant.parse("2026-09-23T07:05:00Z").toSlotCardStartText(moscow))
        // 21:30 UTC 30 сентября — уже 1 октября по Москве: и число, и месяц, и день недели местные.
        assertEquals("Чт, 1 октября · 0:30", Instant.parse("2026-09-30T21:30:00Z").toSlotCardStartText(moscow))
    }

    @Test
    fun routeTypeTexts() {
        assertEquals(RouteType.entries.size, RouteType.entries.map { it.toTagText() }.toSet().size)
        assertEquals(RouteType.entries.size, RouteType.entries.map { it.toDetailsAudienceText() }.toSet().size)
    }
}
