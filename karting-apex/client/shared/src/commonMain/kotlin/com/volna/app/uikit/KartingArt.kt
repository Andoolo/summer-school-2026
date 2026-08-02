package com.volna.app.uikit

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * Картинг-обложка «Апекс»: тёмный асфальт с лёгким градиентом, красная гоночная полоса
 * и клетчатая лента финишного флага снизу. Используется как hero-плейсхолдер карточек
 * заезда (вместо пастельного градиента «Волны»).
 */
@Composable
fun KartingHeroArt(
    height: Dp,
    modifier: Modifier = Modifier,
    shape: Shape = RectangleShape,
) {
    Canvas(
        modifier = modifier
            .fillMaxWidth()
            .height(height)
            .clip(shape),
    ) {
        // Асфальт
        drawRect(
            brush = Brush.horizontalGradient(
                colors = listOf(Color(0xFF1C1C24), Color(0xFF2E2E38), Color(0xFF1C1C24)),
            ),
        )
        // Красная гоночная полоса по диагонали
        val stripe = androidx.compose.ui.graphics.Path().apply {
            moveTo(size.width * 0.58f, 0f)
            lineTo(size.width * 0.74f, 0f)
            lineTo(size.width * 0.46f, size.height)
            lineTo(size.width * 0.30f, size.height)
            close()
        }
        drawPath(stripe, Color(0xFFE10600))
        val stripeThin = androidx.compose.ui.graphics.Path().apply {
            moveTo(size.width * 0.80f, 0f)
            lineTo(size.width * 0.86f, 0f)
            lineTo(size.width * 0.58f, size.height)
            lineTo(size.width * 0.52f, size.height)
            close()
        }
        drawPath(stripeThin, Color(0xFFE10600).copy(alpha = 0.55f))

        // Клетчатая лента финишного флага (2 ряда) внизу
        val cell = 10.dp.toPx()
        val rows = 2
        val bandTop = size.height - cell * rows
        var col = 0
        var x = 0f
        while (x < size.width) {
            for (row in 0 until rows) {
                val isWhite = (col + row) % 2 == 0
                drawRect(
                    color = if (isWhite) Color(0xFFF2F2F2) else Color(0xFF15151E),
                    topLeft = Offset(x, bandTop + row * cell),
                    size = Size(minOf(cell, size.width - x), cell),
                )
            }
            x += cell
            col++
        }
    }
}
