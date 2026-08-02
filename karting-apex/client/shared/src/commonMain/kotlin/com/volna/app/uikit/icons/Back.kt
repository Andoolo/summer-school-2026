package com.volna.app.uikit.icons

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.path
import androidx.compose.ui.unit.dp

val Icons.Back: ImageVector
    get() {
        if (_Back != null) {
            return _Back!!
        }
        _Back = ImageVector.Builder(
            name = "Back",
            defaultWidth = 16.dp,
            defaultHeight = 16.dp,
            viewportWidth = 16f,
            viewportHeight = 16f
        ).apply {
            path(fill = SolidColor(Color(0xFF797979))) {
                moveTo(3.492f, 9.048f)
                lineTo(8.526f, 14.218f)
                curveTo(8.731f, 14.429f, 8.83f, 14.676f, 8.822f, 14.957f)
                curveTo(8.814f, 15.238f, 8.707f, 15.484f, 8.5f, 15.696f)
                curveTo(8.295f, 15.889f, 8.055f, 15.99f, 7.781f, 15.999f)
                curveTo(7.507f, 16.009f, 7.268f, 15.907f, 7.062f, 15.696f)
                lineTo(0.282f, 8.731f)
                curveTo(0.179f, 8.626f, 0.106f, 8.512f, 0.063f, 8.389f)
                curveTo(0.02f, 8.265f, -0.001f, 8.134f, 0f, 7.993f)
                curveTo(0.001f, 7.852f, 0.023f, 7.72f, 0.065f, 7.597f)
                curveTo(0.107f, 7.474f, 0.179f, 7.36f, 0.283f, 7.254f)
                lineTo(7.063f, 0.29f)
                curveTo(7.251f, 0.097f, 7.487f, 0f, 7.77f, 0f)
                curveTo(8.053f, 0f, 8.297f, 0.097f, 8.501f, 0.29f)
                curveTo(8.707f, 0.501f, 8.81f, 0.752f, 8.81f, 1.043f)
                curveTo(8.81f, 1.333f, 8.707f, 1.583f, 8.501f, 1.794f)
                lineTo(3.492f, 6.938f)
                horizontalLineTo(14.973f)
                curveTo(15.264f, 6.938f, 15.508f, 7.039f, 15.705f, 7.242f)
                curveTo(15.902f, 7.444f, 16.001f, 7.695f, 16f, 7.993f)
                curveTo(15.999f, 8.291f, 15.901f, 8.542f, 15.704f, 8.745f)
                curveTo(15.508f, 8.948f, 15.264f, 9.049f, 14.973f, 9.048f)
                horizontalLineTo(3.492f)
                close()
            }
        }.build()

        return _Back!!
    }

@Suppress("ObjectPropertyName")
private var _Back: ImageVector? = null
