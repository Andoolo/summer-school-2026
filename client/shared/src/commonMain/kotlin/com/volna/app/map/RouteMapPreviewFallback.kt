package com.volna.app.map

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.domain.model.GeoPoint
import com.volna.app.domain.model.MeetingPoint
import com.volna.app.domain.model.Route
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.min

@Composable
fun RouteMapPreviewFallback(
    route: Route,
    meetingPoint: MeetingPoint,
    onOpenExternal: () -> Unit,
) {
    val spacing = VolnaTheme.tokens.spacing
    val waterColor = Color(0xFF8BD1F1)
    val landColor = Color(0xFFF6F3ED)
    val parkColor = Color(0xFFD5F3BA)
    val streetColor = Color.White
    val roadStrokeColor = Color(0xFFE5DED6)
    val routeColor = Color(0xFF00A59D)
    val pinColor = Color(0xFF00A59D)
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = Color(0xFFF2F2F2),
                shape = androidx.compose.foundation.shape.RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .padding(spacing.md),
        verticalArrangement = Arrangement.spacedBy(spacing.sm),
    ) {
        Text(
            text = "Адрес: ${meetingPoint.title.ifBlank { "уточняется" }}",
            modifier = Modifier.fillMaxWidth(),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface,
            textAlign = TextAlign.Center,
        )
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(365.dp)
                .clip(androidx.compose.foundation.shape.RoundedCornerShape(VolnaTheme.tokens.radius.md))
                .clickable { onOpenExternal() },
        ) {
            MockRouteScreenshot(
                // Реальная геометрия маршрута из API (F1: route.geometry).
                // Пусто → рисуется декоративная заглушка-линия (fallback).
                routePoints = route.geometry?.points.orEmpty(),
                waterColor = waterColor,
                landColor = landColor,
                parkColor = parkColor,
                streetColor = streetColor,
                roadStrokeColor = roadStrokeColor,
                routeColor = routeColor,
                pinColor = pinColor,
            )
        }
    }
}

@Composable
private fun MockRouteScreenshot(
    routePoints: List<GeoPoint>,
    waterColor: Color,
    landColor: Color,
    parkColor: Color,
    streetColor: Color,
    roadStrokeColor: Color,
    routeColor: Color,
    pinColor: Color,
) {
    Canvas(
        modifier = Modifier
            .fillMaxWidth()
            .height(365.dp),
    ) {
        val corner = 12.dp.toPx()
        drawRoundRect(
            color = waterColor,
            cornerRadius = CornerRadius(corner, corner),
        )
        drawPath(
            path = androidx.compose.ui.graphics.Path().apply {
                moveTo(0f, 0f)
                lineTo(size.width * 0.16f, 0f)
                cubicTo(size.width * 0.08f, size.height * 0.18f, size.width * 0.18f, size.height * 0.32f, size.width * 0.1f, size.height * 0.48f)
                cubicTo(size.width * 0.02f, size.height * 0.64f, size.width * 0.24f, size.height * 0.78f, size.width * 0.18f, size.height)
                lineTo(0f, size.height)
                close()
            },
            color = landColor,
        )
        drawPath(
            path = androidx.compose.ui.graphics.Path().apply {
                moveTo(0f, size.height * 0.12f)
                lineTo(size.width * 0.06f, size.height * 0.08f)
                lineTo(size.width * 0.04f, size.height * 0.28f)
                lineTo(0f, size.height * 0.33f)
                close()
            },
            color = parkColor,
        )
        drawPath(
            path = androidx.compose.ui.graphics.Path().apply {
                moveTo(0f, size.height * 0.83f)
                cubicTo(size.width * 0.12f, size.height * 0.86f, size.width * 0.25f, size.height * 0.92f, size.width * 0.36f, size.height)
                lineTo(0f, size.height)
                close()
            },
            color = parkColor,
        )
        listOf(0.24f, 0.42f, 0.62f, 0.82f).forEach { y ->
            drawLine(
                color = roadStrokeColor,
                start = Offset(0f, size.height * y),
                end = Offset(size.width * 0.38f, size.height * (y - 0.08f)),
                strokeWidth = 6.dp.toPx(),
                cap = StrokeCap.Round,
            )
            drawLine(
                color = streetColor,
                start = Offset(0f, size.height * y),
                end = Offset(size.width * 0.38f, size.height * (y - 0.08f)),
                strokeWidth = 4.dp.toPx(),
                cap = StrokeCap.Round,
            )
        }
        listOf(0.12f, 0.26f).forEach { x ->
            drawLine(
                color = roadStrokeColor,
                start = Offset(size.width * x, 0f),
                end = Offset(size.width * (x + 0.16f), size.height),
                strokeWidth = 6.dp.toPx(),
                cap = StrokeCap.Round,
            )
            drawLine(
                color = streetColor,
                start = Offset(size.width * x, 0f),
                end = Offset(size.width * (x + 0.16f), size.height),
                strokeWidth = 4.dp.toPx(),
                cap = StrokeCap.Round,
            )
        }

        // Проекция реальной геометрии маршрута на канвас (F2).
        val projected = projectRoute(
            points = routePoints,
            width = size.width,
            height = size.height,
            padding = 28.dp.toPx(),
        )
        val routePath = if (projected.size >= 2) {
            androidx.compose.ui.graphics.Path().apply {
                moveTo(projected.first().x, projected.first().y)
                for (i in 1 until projected.size) {
                    lineTo(projected[i].x, projected[i].y)
                }
            }
        } else {
            // Fallback: декоративная линия, если geometry не пришла.
            androidx.compose.ui.graphics.Path().apply {
                moveTo(size.width * 0.34f, size.height * 0.57f)
                cubicTo(size.width * 0.58f, size.height * 0.58f, size.width * 0.62f, size.height * 0.48f, size.width * 0.61f, size.height * 0.43f)
                cubicTo(size.width * 0.6f, size.height * 0.36f, size.width * 0.72f, size.height * 0.32f, size.width * 0.82f, size.height * 0.24f)
                lineTo(size.width * 0.82f, 0f)
            }
        }
        drawPath(
            path = routePath,
            color = routeColor,
            style = androidx.compose.ui.graphics.drawscope.Stroke(
                width = 3.dp.toPx(),
                cap = StrokeCap.Round,
                join = StrokeJoin.Round,
            ),
        )

        // Пин в начале маршрута (первая точка geometry либо начало декоративной линии).
        val pinCenter = if (projected.isNotEmpty()) {
            projected.first()
        } else {
            Offset(size.width * 0.34f, size.height * 0.57f)
        }
        drawCircle(
            color = routeColor.copy(alpha = 0.16f),
            radius = 32.dp.toPx(),
            center = pinCenter,
        )
        drawCircle(
            color = pinColor,
            radius = 7.dp.toPx(),
            center = pinCenter,
        )
        drawCircle(
            color = Color.White,
            radius = 3.dp.toPx(),
            center = pinCenter,
        )
    }
}

/**
 * Проецирует географические координаты маршрута в точки канваса.
 *
 * - долгота корректируется на cos(широты), чтобы форма не растягивалась по горизонтали;
 * - единый масштаб по обеим осям (форма не искажается) + центрирование;
 * - ось Y инвертируется (север — вверху).
 *
 * Возвращает пустой список, если точек меньше двух.
 */
private fun projectRoute(
    points: List<GeoPoint>,
    width: Float,
    height: Float,
    padding: Float,
): List<Offset> {
    if (points.size < 2) return emptyList()

    val midLatRad = (points.sumOf { it.lat } / points.size) * (PI / 180.0)
    val cosLat = cos(midLatRad)

    // Корректированные плоские координаты: x = lng*cos(lat), y = lat.
    val xs = points.map { it.lng * cosLat }
    val ys = points.map { it.lat }
    val minX = xs.min()
    val maxX = xs.max()
    val minY = ys.min()
    val maxY = ys.max()

    val epsilon = 1e-9
    val spanX = (maxX - minX).takeIf { it > epsilon } ?: epsilon
    val spanY = (maxY - minY).takeIf { it > epsilon } ?: epsilon

    val innerW = (width - 2 * padding).coerceAtLeast(1f)
    val innerH = (height - 2 * padding).coerceAtLeast(1f)

    val scale = min(innerW / spanX, innerH / spanY)
    val drawnW = (spanX * scale).toFloat()
    val drawnH = (spanY * scale).toFloat()
    val offsetX = padding + (innerW - drawnW) / 2f
    val offsetY = padding + (innerH - drawnH) / 2f

    return points.map { p ->
        val x = offsetX + ((p.lng * cosLat - minX) * scale).toFloat()
        // инверсия Y: большая широта (север) → меньший y (верх экрана)
        val y = offsetY + ((maxY - p.lat) * scale).toFloat()
        Offset(x, y)
    }
}
