package com.volna.app.uikit.icons

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.path
import androidx.compose.ui.unit.dp

val Icons.Info: ImageVector
    get() {
        if (_Info != null) {
            return _Info!!
        }
        _Info = ImageVector.Builder(
            name = "Info",
            defaultWidth = 16.dp,
            defaultHeight = 16.dp,
            viewportWidth = 16f,
            viewportHeight = 16f
        ).apply {
            path(
                stroke = SolidColor(Color(0xFF797979)),
                strokeLineWidth = 1f,
                strokeLineCap = StrokeCap.Round,
                strokeLineJoin = StrokeJoin.Round
            ) {
                moveTo(8f, 10.4f)
                verticalLineTo(8f)
                moveTo(8f, 5.6f)
                horizontalLineTo(8.006f)
                moveTo(14f, 8f)
                curveTo(14f, 11.314f, 11.314f, 14f, 8f, 14f)
                curveTo(4.686f, 14f, 2f, 11.314f, 2f, 8f)
                curveTo(2f, 4.686f, 4.686f, 2f, 8f, 2f)
                curveTo(11.314f, 2f, 14f, 4.686f, 14f, 8f)
                close()
            }
        }.build()

        return _Info!!
    }

@Suppress("ObjectPropertyName")
private var _Info: ImageVector? = null
