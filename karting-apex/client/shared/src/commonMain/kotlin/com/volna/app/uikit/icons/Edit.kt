package com.volna.app.uikit.icons

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.path
import androidx.compose.ui.unit.dp

val Icons.Edit: ImageVector
    get() {
        if (_Edit != null) {
            return _Edit!!
        }
        _Edit = ImageVector.Builder(
            name = "Edit",
            defaultWidth = 24.dp,
            defaultHeight = 24.dp,
            viewportWidth = 24f,
            viewportHeight = 24f
        ).apply {
            path(
                stroke = SolidColor(Color(0xFF797979)),
                strokeLineWidth = 1.5f,
                strokeLineCap = StrokeCap.Round,
                strokeLineJoin = StrokeJoin.Round
            ) {
                moveTo(12f, 19.06f)
                horizontalLineTo(20f)
                moveTo(16f, 4.605f)
                curveTo(16.354f, 4.256f, 16.833f, 4.061f, 17.333f, 4.061f)
                curveTo(17.581f, 4.061f, 17.826f, 4.109f, 18.055f, 4.202f)
                curveTo(18.284f, 4.295f, 18.492f, 4.432f, 18.667f, 4.605f)
                curveTo(18.842f, 4.777f, 18.981f, 4.982f, 19.075f, 5.208f)
                curveTo(19.17f, 5.433f, 19.219f, 5.675f, 19.219f, 5.919f)
                curveTo(19.219f, 6.163f, 19.17f, 6.405f, 19.075f, 6.63f)
                curveTo(18.981f, 6.856f, 18.842f, 7.061f, 18.667f, 7.233f)
                lineTo(7.556f, 18.184f)
                lineTo(4f, 19.06f)
                lineTo(4.889f, 15.556f)
                lineTo(16f, 4.605f)
                close()
            }
        }.build()

        return _Edit!!
    }

@Suppress("ObjectPropertyName")
private var _Edit: ImageVector? = null
