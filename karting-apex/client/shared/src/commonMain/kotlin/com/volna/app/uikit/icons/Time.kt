package com.volna.app.uikit.icons

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.path
import androidx.compose.ui.unit.dp

val Icons.Time: ImageVector
    get() {
        if (_Time != null) {
            return _Time!!
        }
        _Time = ImageVector.Builder(
            name = "Time",
            defaultWidth = 16.dp,
            defaultHeight = 16.dp,
            viewportWidth = 16f,
            viewportHeight = 16f
        ).apply {
            path(fill = SolidColor(Color(0xFF797979))) {
                moveTo(8f, 15f)
                curveTo(4.14f, 15f, 1f, 11.86f, 1f, 8f)
                curveTo(1f, 4.14f, 4.14f, 1f, 8f, 1f)
                curveTo(11.86f, 1f, 15f, 4.14f, 15f, 8f)
                curveTo(15f, 11.86f, 11.86f, 15f, 8f, 15f)
                close()
                moveTo(8f, 2f)
                curveTo(4.69f, 2f, 2f, 4.69f, 2f, 8f)
                curveTo(2f, 11.31f, 4.69f, 14f, 8f, 14f)
                curveTo(11.31f, 14f, 14f, 11.31f, 14f, 8f)
                curveTo(14f, 4.69f, 11.31f, 2f, 8f, 2f)
                close()
            }
            path(fill = SolidColor(Color(0xFF797979))) {
                moveTo(10f, 10.5f)
                curveTo(9.91f, 10.5f, 9.82f, 10.48f, 9.74f, 10.43f)
                lineTo(7.24f, 8.93f)
                curveTo(7.166f, 8.885f, 7.105f, 8.822f, 7.063f, 8.747f)
                curveTo(7.021f, 8.671f, 6.999f, 8.586f, 7f, 8.5f)
                verticalLineTo(4.5f)
                curveTo(7f, 4.22f, 7.22f, 4f, 7.5f, 4f)
                curveTo(7.78f, 4f, 8f, 4.22f, 8f, 4.5f)
                verticalLineTo(8.22f)
                lineTo(10.26f, 9.57f)
                curveTo(10.353f, 9.627f, 10.425f, 9.713f, 10.465f, 9.815f)
                curveTo(10.505f, 9.916f, 10.51f, 10.028f, 10.481f, 10.133f)
                curveTo(10.452f, 10.238f, 10.389f, 10.331f, 10.302f, 10.397f)
                curveTo(10.215f, 10.463f, 10.109f, 10.5f, 10f, 10.5f)
                close()
            }
        }.build()

        return _Time!!
    }

@Suppress("ObjectPropertyName")
private var _Time: ImageVector? = null
