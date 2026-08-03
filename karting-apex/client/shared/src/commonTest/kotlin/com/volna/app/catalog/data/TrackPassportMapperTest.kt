package com.volna.app.catalog.data

import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.TrackDirection
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Разбор ответа GET /routes/{id} (F5).
 *
 * Маппер — единственное место, где контракт бэкенда превращается в доменную модель,
 * поэтому расхождение контракта ломает экран трассы молча: поля просто окажутся
 * пустыми. Тесты фиксируют разбор на реальном ответе сервиса.
 */
class TrackPassportMapperTest {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun parsesFullPassportFromRealResponse() {
        // Ответ скопирован с работающего сервиса, обрезана только геометрия.
        val dto = json.decodeFromString<TrackPassportDto>(
            """
            {
              "id": "11111111-1111-1111-1111-111111111111",
              "name": "Городское кольцо",
              "type": "novice",
              "capacity_cap": 8,
              "duration_min": 20,
              "geometry": [[55.74932, 37.607844], [55.750, 37.608], [55.751, 37.609]],
              "length_m": 980,
              "corners": 10,
              "direction": "clockwise",
              "main_straight_m": 200,
              "surface": "Асфальт",
              "width_m": 8,
              "elevation_m": 4.2,
              "opened_year": 2019,
              "kart_model": "Sodi RT8",
              "kart_power_hp": 13,
              "record": {"name": "Сергей", "lap_ms": 41890}
            }
            """.trimIndent(),
        )

        val passport = dto.toDomain()

        assertEquals("Городское кольцо", passport.name)
        assertEquals(RouteType.Novice, passport.type)
        assertEquals(980, passport.lengthM)
        assertEquals(10, passport.corners)
        assertEquals(TrackDirection.Clockwise, passport.direction)
        assertEquals(200, passport.mainStraightM)
        assertEquals("Асфальт", passport.surface)
        assertEquals(8.0, passport.widthM)
        assertEquals(4.2, passport.elevationM)
        assertEquals(2019, passport.openedYear)
        assertEquals("Sodi RT8", passport.kartModel)
        assertEquals(13, passport.kartPowerHp)
        assertEquals("Сергей", passport.record?.name)
        assertEquals(41890, passport.record?.lapMs)
    }

    @Test
    fun parsesGeometryPointsInLatLngOrder() {
        // Порядок координат перепутать легко, а на схеме это даст зеркальный контур.
        val dto = json.decodeFromString<TrackPassportDto>(
            """
            {
              "id": "11111111-1111-1111-1111-111111111111",
              "name": "Т", "type": "novice", "capacity_cap": 8, "duration_min": 20,
              "geometry": [[55.75, 37.61], [55.76, 37.62]],
              "length_m": 1, "corners": 1, "direction": "clockwise", "main_straight_m": 1
            }
            """.trimIndent(),
        )

        val points = dto.toDomain().geometry?.points.orEmpty()

        assertEquals(2, points.size)
        assertEquals(55.75, points[0].lat)
        assertEquals(37.61, points[0].lng)
    }

    @Test
    fun keepsPassportFieldsNullWhenServerOmitsThem() {
        // Паспортные поля необязательные: бэкенд не отдаёт их, если они не заполнены.
        // Экран должен просто скрыть такие строки, а не показать «null».
        val dto = json.decodeFromString<TrackPassportDto>(
            """
            {
              "id": "22222222-2222-2222-2222-222222222222",
              "name": "Без паспорта", "type": "experienced",
              "capacity_cap": 12, "duration_min": 30,
              "geometry": [],
              "length_m": 1240, "corners": 16,
              "direction": "counter_clockwise", "main_straight_m": 237
            }
            """.trimIndent(),
        )

        val passport = dto.toDomain()

        assertEquals(RouteType.Experienced, passport.type)
        assertEquals(TrackDirection.CounterClockwise, passport.direction)
        assertNull(passport.surface)
        assertNull(passport.widthM)
        assertNull(passport.openedYear)
        assertNull(passport.kartModel)
        assertNull(passport.record)
        // Пустой массив координат — это отсутствие схемы, а не схема из нуля точек.
        assertNull(passport.geometry)
    }

    @Test
    fun fallsBackToCounterClockwiseOnUnknownDirection() {
        // Неизвестное значение не должно ронять разбор всего ответа.
        val dto = json.decodeFromString<TrackPassportDto>(
            """
            {
              "id": "33333333-3333-3333-3333-333333333333",
              "name": "Т", "type": "novice", "capacity_cap": 8, "duration_min": 20,
              "length_m": 1, "corners": 1, "direction": "spiral", "main_straight_m": 1
            }
            """.trimIndent(),
        )

        assertEquals(TrackDirection.CounterClockwise, dto.toDomain().direction)
    }

    @Test
    fun ignoresUnknownFieldsSoNewServerFieldsDoNotBreakOldClients() {
        val dto = json.decodeFromString<TrackPassportDto>(
            """
            {
              "id": "44444444-4444-4444-4444-444444444444",
              "name": "Т", "type": "novice", "capacity_cap": 8, "duration_min": 20,
              "length_m": 1, "corners": 1, "direction": "clockwise", "main_straight_m": 1,
              "brand_new_field_from_future": {"nested": [1, 2, 3]}
            }
            """.trimIndent(),
        )

        assertTrue(dto.toDomain().name == "Т")
    }
}
