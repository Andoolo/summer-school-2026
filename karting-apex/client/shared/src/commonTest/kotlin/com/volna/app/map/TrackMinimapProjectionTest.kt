package com.volna.app.map

import com.volna.app.domain.model.GeoPoint
import kotlin.math.abs
import kotlin.math.cos
import kotlin.math.PI
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Проекция схемы трассы (F5): одна и та же математика рисует мини-схему на карточке и
 * крупную схему в шторке. Ошибка в ней не падает, а тихо искажает контур: сплющивает
 * трассу, переворачивает её или уводит за край.
 */
class TrackMinimapProjectionTest {
    private val lat = 55.75 // Москва: здесь градус долготы короче градуса широты почти вдвое

    /** Квадрат со стороной ~100 м в метрах на местности. */
    private fun squareTrack(): List<GeoPoint> {
        val dLat = 100.0 / 111_320.0
        val dLng = dLat / cos(lat * PI / 180.0)
        return listOf(
            GeoPoint(lat, 37.6),
            GeoPoint(lat, 37.6 + dLng),
            GeoPoint(lat + dLat, 37.6 + dLng),
            GeoPoint(lat + dLat, 37.6),
        )
    }

    @Test
    fun keepsRealProportionsAtHighLatitude() {
        val canvas = assertNotNull(prepareMinimap(squareTrack())).toCanvas(width = 400f, height = 200f, padding = 20f)
        val width = canvas.maxOf { it.x } - canvas.minOf { it.x }
        val height = canvas.maxOf { it.y } - canvas.minOf { it.y }
        // Квадрат на местности остаётся квадратом. Без поправки на широту он вышел бы
        // прямоугольником ~1,8 : 1.
        assertTrue(abs(width - height) < 0.5f, "width=$width height=$height")
    }

    @Test
    fun northIsUpAndEastIsRight() {
        val points = squareTrack()
        val canvas = assertNotNull(prepareMinimap(points)).toCanvas(400f, 400f, 20f)
        // points[0] — юго-запад, points[2] — северо-восток.
        assertTrue(canvas[2].y < canvas[0].y, "север должен быть выше: ${canvas[2]} vs ${canvas[0]}")
        assertTrue(canvas[2].x > canvas[0].x, "восток должен быть правее: ${canvas[2]} vs ${canvas[0]}")
    }

    @Test
    fun fitsInsidePaddingAndIsCentered() {
        val width = 390f
        val height = 156f
        val padding = 18f
        val canvas = assertNotNull(prepareMinimap(squareTrack())).toCanvas(width, height, padding)

        assertTrue(canvas.all { it.x >= padding - 0.01f && it.x <= width - padding + 0.01f })
        assertTrue(canvas.all { it.y >= padding - 0.01f && it.y <= height - padding + 0.01f })
        // Упор в высоту: контур занимает её целиком и стоит по центру по горизонтали.
        assertEquals(height - 2 * padding, canvas.maxOf { it.y } - canvas.minOf { it.y }, 0.5f)
        val left = canvas.minOf { it.x }
        val right = width - canvas.maxOf { it.x }
        assertEquals(left, right, 0.5f)
    }

    @Test
    fun degenerateInputDoesNotProduceNaN() {
        assertNull(prepareMinimap(emptyList()))
        assertNull(prepareMinimap(listOf(GeoPoint(lat, 37.6))), "из одной точки контур не нарисовать")

        // Все точки на одной широте (прямая с запада на восток): высота охвата нулевая.
        val line = listOf(GeoPoint(lat, 37.60), GeoPoint(lat, 37.61), GeoPoint(lat, 37.62))
        val canvas = assertNotNull(prepareMinimap(line)).toCanvas(300f, 150f, 10f)
        assertTrue(canvas.none { it.x.isNaN() || it.y.isNaN() || it.x.isInfinite() || it.y.isInfinite() }, "$canvas")
    }
}
