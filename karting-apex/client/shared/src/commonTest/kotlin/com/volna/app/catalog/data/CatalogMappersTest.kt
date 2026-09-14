package com.volna.app.catalog.data

import com.volna.app.domain.model.GeoPoint
import com.volna.app.domain.model.RouteType
import com.volna.app.domain.model.SlotStatus
import kotlinx.datetime.Instant
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

/**
 * Разбор каталога и рекордов трассы. Как и паспорт трассы, расхождение контракта здесь
 * ломает экраны молча — поэтому разбор проверяется на реальных ответах сервиса.
 */
class CatalogMappersTest {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun parsesSlotListFromRealResponse() {
        // Ответ GET /slots с работающего сервиса; геометрия обрезана до трёх точек.
        val page = json.decodeFromString<SlotListResponseDto>(
            """
            {"items": [{"free_rental_boards": 12, "free_seats": 5, "id": "99999999-9999-9999-9999-999999999999",
              "instructor": {"id": "33333333-3333-3333-3333-333333333333", "name": "Марат"},
              "price": 2500, "rental_price": 800,
              "route": {"capacity_cap": 8, "duration_min": 20,
                "geometry": [[55.74932, 37.607845], [55.74946, 37.60788], [55.749596, 37.607918]],
                "id": "11111111-1111-1111-1111-111111111111", "name": "Городское кольцо", "type": "novice"},
              "start_at": "2026-09-23T08:58:33.479254Z", "status": "scheduled", "total_seats": 8}],
             "meta": {"limit": 1, "offset": 0, "total": 4}}
            """.trimIndent(),
        ).toDomain()

        assertEquals(1, page.limit)
        assertEquals(4, page.total)
        val slot = page.items.single()
        assertEquals("99999999-9999-9999-9999-999999999999", slot.id.value)
        assertEquals(Instant.parse("2026-09-23T08:58:33.479254Z"), slot.startAt)
        assertEquals("Марат", slot.instructor.name)
        assertEquals(8, slot.totalSeats)
        assertEquals(5, slot.freeSeats)
        assertEquals(12, slot.freeRentalBoards)
        assertEquals(2500, slot.price.value)
        assertEquals(800, slot.rentalPrice.value)
        assertEquals(SlotStatus.Scheduled, slot.status)
        assertEquals("Городское кольцо", slot.route.name)
        assertEquals(RouteType.Novice, slot.route.type)
        assertEquals(20, slot.route.durationMin)
        // В геометрии порядок [широта, долгота]: перепутать — и схема повернётся на 90°.
        assertEquals(GeoPoint(lat = 55.74932, lng = 37.607845), slot.route.geometry?.points?.first())
        assertEquals(3, slot.route.geometry?.points?.size)
    }

    @Test
    fun mapsStatusAndRouteTypeWithSafeFallbacks() {
        val slot = json.decodeFromString<SlotDto>(slotJson(status = "cancelled", type = "experienced")).toDomain()
        assertEquals(SlotStatus.Cancelled, slot.status)
        assertEquals(RouteType.Experienced, slot.route.type)

        // Незнакомые значения от будущего сервера не роняют разбор: заезд считается обычным.
        val unknown = json.decodeFromString<SlotDto>(slotJson(status = "rescheduled", type = "pro")).toDomain()
        assertEquals(SlotStatus.Scheduled, unknown.status)
        assertEquals(RouteType.Novice, unknown.route.type)
    }

    @Test
    fun skipsMalformedGeometryPointsAndDropsEmptyGeometry() {
        val partial = json.decodeFromString<SlotDto>(
            slotJson(geometry = """[[55.7, 37.6], [55.8], "oops", [null, 37.6], [55.9, 37.7]]"""),
        ).toDomain()
        assertEquals(
            listOf(GeoPoint(55.7, 37.6), GeoPoint(55.9, 37.7)),
            partial.route.geometry?.points,
        )

        val garbage = json.decodeFromString<SlotDto>(slotJson(geometry = """[[], "x"]""")).toDomain()
        assertNull(garbage.route.geometry, "без валидных точек схемы нет — экран покажет заглушку, а не пустой контур")

        val absent = json.decodeFromString<SlotDto>(slotJson(geometry = null)).toDomain()
        assertNull(absent.route.geometry)
    }

    @Test
    fun meetingPointFromDetailsResponse() {
        val slot = json.decodeFromString<SlotDto>(
            slotJson(extra = """, "meeting_point": "Паддок · трасса «Апекс»", "meeting_point_lat": 55.76, "meeting_point_lng": 37.63"""),
        ).toDomain()
        assertEquals("Паддок · трасса «Апекс»", slot.meetingPoint.title)
        assertEquals(GeoPoint(55.76, 37.63), slot.meetingPoint.coordinates)
    }

    @Test
    fun parsesLeaderboardFromRealResponse() {
        // Ответ GET /routes/{id}/leaderboard с работающего сервиса.
        val response = json.decodeFromString<LeaderboardResponseDto>(
            """
            {"route_id":"22222222-2222-2222-2222-222222222222","route_name":"Спортивная трасса",
             "entries":[{"position":1,"name":"Сергей","best_lap_ms":56771,"laps":2},
                        {"position":2,"name":"Иван","best_lap_ms":58204,"laps":2}]}
            """.trimIndent(),
        )
        val entries = response.entries.map { it.toDomain() }

        assertEquals(listOf(1, 2), entries.map { it.position })
        assertEquals("Сергей", entries[0].name)
        assertEquals(56771, entries[0].bestLapMs)
        assertEquals(2, entries[0].laps)
    }

    private fun slotJson(
        status: String = "scheduled",
        type: String = "novice",
        geometry: String? = """[[55.7, 37.6], [55.8, 37.7]]""",
        extra: String = "",
    ): String {
        val geometryField = geometry?.let { """, "geometry": $it""" }.orEmpty()
        return """
            {"id": "slot-1", "start_at": "2026-09-23T09:00:00Z", "status": "$status",
             "total_seats": 8, "free_seats": 5, "free_rental_boards": 12, "price": 2500, "rental_price": 800,
             "instructor": {"id": "i-1", "name": "Марат"},
             "route": {"id": "r-1", "name": "Городское кольцо", "type": "$type", "capacity_cap": 8, "duration_min": 20$geometryField}
             $extra}
        """.trimIndent()
    }
}
