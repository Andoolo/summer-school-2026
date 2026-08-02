package com.volna.app.uikit.icons

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.PathFillType
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.path
import androidx.compose.ui.unit.dp

val Icons.Tune: ImageVector
    get() {
        if (_Tune != null) {
            return _Tune!!
        }
        _Tune = ImageVector.Builder(
            name = "Tune",
            defaultWidth = 32.dp,
            defaultHeight = 32.dp,
            viewportWidth = 32f,
            viewportHeight = 32f
        ).apply {
            path(
                fill = SolidColor(Color(0xFF797979)),
                pathFillType = PathFillType.EvenOdd
            ) {
                moveTo(4f, 21.333f)
                curveTo(4f, 20.597f, 4.597f, 20f, 5.333f, 20f)
                horizontalLineTo(26.667f)
                curveTo(27.403f, 20f, 28f, 20.597f, 28f, 21.333f)
                curveTo(28f, 22.07f, 27.403f, 22.667f, 26.667f, 22.667f)
                horizontalLineTo(5.333f)
                curveTo(4.597f, 22.667f, 4f, 22.07f, 4f, 21.333f)
                close()
            }
            path(
                fill = SolidColor(Color(0xFF797979)),
                pathFillType = PathFillType.EvenOdd
            ) {
                moveTo(4f, 10.667f)
                curveTo(4f, 9.93f, 4.597f, 9.333f, 5.333f, 9.333f)
                horizontalLineTo(26.667f)
                curveTo(27.403f, 9.333f, 28f, 9.93f, 28f, 10.667f)
                curveTo(28f, 11.403f, 27.403f, 12f, 26.667f, 12f)
                horizontalLineTo(5.333f)
                curveTo(4.597f, 12f, 4f, 11.403f, 4f, 10.667f)
                close()
            }
            path(
                fill = SolidColor(Color(0xFF797979)),
                pathFillType = PathFillType.EvenOdd
            ) {
                moveTo(20f, 12f)
                curveTo(20.736f, 12f, 21.333f, 11.403f, 21.333f, 10.667f)
                curveTo(21.333f, 9.93f, 20.736f, 9.333f, 20f, 9.333f)
                curveTo(19.264f, 9.333f, 18.667f, 9.93f, 18.667f, 10.667f)
                curveTo(18.667f, 11.403f, 19.264f, 12f, 20f, 12f)
                close()
                moveTo(20f, 14.667f)
                curveTo(22.209f, 14.667f, 24f, 12.876f, 24f, 10.667f)
                curveTo(24f, 8.458f, 22.209f, 6.667f, 20f, 6.667f)
                curveTo(17.791f, 6.667f, 16f, 8.458f, 16f, 10.667f)
                curveTo(16f, 12.876f, 17.791f, 14.667f, 20f, 14.667f)
                close()
            }
            path(
                fill = SolidColor(Color(0xFF797979)),
                pathFillType = PathFillType.EvenOdd
            ) {
                moveTo(12f, 22.667f)
                curveTo(12.736f, 22.667f, 13.333f, 22.07f, 13.333f, 21.333f)
                curveTo(13.333f, 20.597f, 12.736f, 20f, 12f, 20f)
                curveTo(11.264f, 20f, 10.667f, 20.597f, 10.667f, 21.333f)
                curveTo(10.667f, 22.07f, 11.264f, 22.667f, 12f, 22.667f)
                close()
                moveTo(12f, 25.333f)
                curveTo(14.209f, 25.333f, 16f, 23.543f, 16f, 21.333f)
                curveTo(16f, 19.124f, 14.209f, 17.333f, 12f, 17.333f)
                curveTo(9.791f, 17.333f, 8f, 19.124f, 8f, 21.333f)
                curveTo(8f, 23.543f, 9.791f, 25.333f, 12f, 25.333f)
                close()
            }
        }.build()

        return _Tune!!
    }

@Suppress("ObjectPropertyName")
private var _Tune: ImageVector? = null
