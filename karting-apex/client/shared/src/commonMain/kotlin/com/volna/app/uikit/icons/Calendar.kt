package com.volna.app.uikit.icons

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.path
import androidx.compose.ui.unit.dp

val Icons.Calendar: ImageVector
    get() {
        if (_Calendar != null) {
            return _Calendar!!
        }
        _Calendar = ImageVector.Builder(
            name = "Calendar",
            defaultWidth = 32.dp,
            defaultHeight = 32.dp,
            viewportWidth = 32f,
            viewportHeight = 32f
        ).apply {
            path(fill = SolidColor(Color(0xFF797979))) {
                moveTo(22.667f, 6.667f)
                verticalLineTo(5.333f)
                curveTo(22.667f, 4.533f, 22.133f, 4f, 21.333f, 4f)
                curveTo(20.533f, 4f, 20f, 4.533f, 20f, 5.333f)
                verticalLineTo(6.667f)
                horizontalLineTo(12f)
                verticalLineTo(5.333f)
                curveTo(12f, 4.533f, 11.467f, 4f, 10.667f, 4f)
                curveTo(9.867f, 4f, 9.333f, 4.533f, 9.333f, 5.333f)
                verticalLineTo(6.667f)
                curveTo(6.4f, 6.667f, 4f, 9.067f, 4f, 12f)
                verticalLineTo(22.667f)
                curveTo(4f, 25.6f, 6.4f, 28f, 9.333f, 28f)
                horizontalLineTo(22.667f)
                curveTo(25.6f, 28f, 28f, 25.6f, 28f, 22.667f)
                verticalLineTo(12f)
                curveTo(28f, 9.067f, 25.6f, 6.667f, 22.667f, 6.667f)
                close()
                moveTo(25.333f, 22.667f)
                curveTo(25.333f, 24.133f, 24.133f, 25.333f, 22.667f, 25.333f)
                horizontalLineTo(9.333f)
                curveTo(7.867f, 25.333f, 6.667f, 24.133f, 6.667f, 22.667f)
                verticalLineTo(14.667f)
                horizontalLineTo(25.333f)
                verticalLineTo(22.667f)
                close()
                moveTo(6.667f, 12f)
                curveTo(6.667f, 10.533f, 7.867f, 9.333f, 9.333f, 9.333f)
                horizontalLineTo(22.667f)
                curveTo(24.133f, 9.333f, 25.333f, 10.533f, 25.333f, 12f)
                horizontalLineTo(6.667f)
                close()
            }
            path(fill = SolidColor(Color(0xFF797979))) {
                moveTo(22.667f, 24f)
                curveTo(23.403f, 24f, 24f, 23.403f, 24f, 22.667f)
                curveTo(24f, 21.93f, 23.403f, 21.333f, 22.667f, 21.333f)
                curveTo(21.93f, 21.333f, 21.333f, 21.93f, 21.333f, 22.667f)
                curveTo(21.333f, 23.403f, 21.93f, 24f, 22.667f, 24f)
                close()
            }
            path(fill = SolidColor(Color(0xFF797979))) {
                moveTo(18.667f, 24f)
                curveTo(19.403f, 24f, 20f, 23.403f, 20f, 22.667f)
                curveTo(20f, 21.93f, 19.403f, 21.333f, 18.667f, 21.333f)
                curveTo(17.93f, 21.333f, 17.333f, 21.93f, 17.333f, 22.667f)
                curveTo(17.333f, 23.403f, 17.93f, 24f, 18.667f, 24f)
                close()
            }
        }.build()

        return _Calendar!!
    }

@Suppress("ObjectPropertyName")
private var _Calendar: ImageVector? = null
