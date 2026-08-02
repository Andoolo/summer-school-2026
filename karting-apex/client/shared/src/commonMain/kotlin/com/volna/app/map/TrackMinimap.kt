package com.volna.app.map

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.volna.app.domain.model.GeoPoint
import kotlin.math.PI
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.min

private val AsphaltColor = Color(0xFF2B2B2E)
private val KerbRedColor = Color(0xFFE10600)
private val CenterLineColor = Color(0x8CFFFFFF)
private val BackgroundColor = Color(0xFFF2F2F4)

/**
 * Мини-схема трассы: рисует НАСТОЯЩИЙ контур из geometry, а не декоративную заглушку.
 *
 * Раньше на карточках заезда и брони висели нарисованные «городские карты» с голубой
 * водой и зелёными парками — они игнорировали geometry и остались от «Волны». Здесь
 * контур проецируется тем же способом, что и на большой карте (RouteMapPreviewFallback):
 * долгота корректируется на cos(широты), масштаб единый по обеим осям.
 *
 * Если точек меньше двух, схему рисовать не из чего — компонент не отрисовывается вовсе,
 * чтобы не показывать пустую плашку.
 */
@Composable
fun TrackMinimap(
    points: List<GeoPoint>,
    modifier: Modifier = Modifier,
    height: Dp = 156.dp,
    cornerRadius: Dp = 12.dp,
) {
    if (points.size < 2) return

    val prepared = remember(points) { prepareMinimap(points) } ?: return

    Canvas(
        modifier = modifier
            .fillMaxWidth()
            .height(height)
            .background(BackgroundColor, RoundedCornerShape(cornerRadius)),
    ) {
        val projected = prepared.toCanvas(
            width = size.width,
            height = size.height,
            padding = 18.dp.toPx(),
        )
        if (projected.size < 2) return@Canvas

        val path = Path().apply {
            moveTo(projected.first().x, projected.first().y)
            for (i in 1 until projected.size) lineTo(projected[i].x, projected[i].y)
            // Контур трассы замкнут: последняя точка соединяется с первой.
            close()
        }

        // Полотно трассы — широкая тёмная линия, поверх неё тонкая осевая разметка.
        drawPath(
            path = path,
            color = AsphaltColor,
            style = Stroke(width = 11.dp.toPx(), cap = StrokeCap.Round, join = StrokeJoin.Round),
        )
        drawPath(
            path = path,
            color = CenterLineColor,
            style = Stroke(
                width = 1.dp.toPx(),
                cap = StrokeCap.Round,
                join = StrokeJoin.Round,
                pathEffect = PathEffect.dashPathEffect(
                    floatArrayOf(5.dp.toPx(), 7.dp.toPx()),
                ),
            ),
        )

        drawStartFinishLine(from = projected.first(), to = projected[1])
    }
}

/** Клетчатая линия старт-финиш поперёк полотна в начале круга. */
private fun DrawScope.drawStartFinishLine(from: Offset, to: Offset) {
    val headingDeg = atan2(to.y - from.y, to.x - from.x) * 180f / PI.toFloat()
    val halfWidth = 7.dp.toPx()
    val cellSize = 2.6.dp.toPx()

    rotate(degrees = headingDeg, pivot = from) {
        // Два ряда чередующихся клеток поперёк направления движения.
        var y = -halfWidth
        var row = 0
        while (y < halfWidth) {
            val cellHeight = min(cellSize, halfWidth - y)
            drawRect(
                color = if (row % 2 == 0) Color.White else Color(0xFF161616),
                topLeft = Offset(from.x - cellSize, from.y + y),
                size = androidx.compose.ui.geometry.Size(cellSize, cellHeight),
            )
            drawRect(
                color = if (row % 2 == 0) Color(0xFF161616) else Color.White,
                topLeft = Offset(from.x, from.y + y),
                size = androidx.compose.ui.geometry.Size(cellSize, cellHeight),
            )
            y += cellSize
            row++
        }
        drawRect(
            color = KerbRedColor,
            topLeft = Offset(from.x - cellSize, from.y - halfWidth - 1.2.dp.toPx()),
            size = androidx.compose.ui.geometry.Size(cellSize * 2, 1.2.dp.toPx()),
        )
    }
}

/**
 * Контур в плоских aspect-корректных координатах (x = lng·cos(lat), y = lat) плюс его bbox.
 * Считается один раз по списку точек и не зависит от размера Canvas.
 */
private class PreparedMinimap(
    val points: List<Offset>,
    val minX: Float,
    val minY: Float,
    val maxX: Float,
    val maxY: Float,
)

private fun prepareMinimap(points: List<GeoPoint>): PreparedMinimap? {
    if (points.size < 2) return null

    val midLatRad = (points.sumOf { it.lat } / points.size) * (PI / 180.0)
    val cosLat = cos(midLatRad)
    val corrected = points.map { Offset((it.lng * cosLat).toFloat(), it.lat.toFloat()) }

    return PreparedMinimap(
        points = corrected,
        minX = corrected.minOf { it.x },
        minY = corrected.minOf { it.y },
        maxX = corrected.maxOf { it.x },
        maxY = corrected.maxOf { it.y },
    )
}

/** Единый масштаб по обеим осям (форма не искажается) + центрирование; север — вверху. */
private fun PreparedMinimap.toCanvas(width: Float, height: Float, padding: Float): List<Offset> {
    val epsilon = 1e-6f
    val spanX = (maxX - minX).takeIf { it > epsilon } ?: epsilon
    val spanY = (maxY - minY).takeIf { it > epsilon } ?: epsilon

    val innerW = (width - 2 * padding).coerceAtLeast(1f)
    val innerH = (height - 2 * padding).coerceAtLeast(1f)

    val scale = min(innerW / spanX, innerH / spanY)
    val offsetX = padding + (innerW - spanX * scale) / 2f
    val offsetY = padding + (innerH - spanY * scale) / 2f

    return points.map { p ->
        Offset(
            x = offsetX + (p.x - minX) * scale,
            y = offsetY + (maxY - p.y) * scale,
        )
    }
}
