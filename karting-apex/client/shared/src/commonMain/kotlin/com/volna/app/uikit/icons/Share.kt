package com.volna.app.uikit.icons

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.path
import androidx.compose.ui.unit.dp

val Icons.Share: ImageVector
    get() {
        if (_Share != null) {
            return _Share!!
        }
        _Share = ImageVector.Builder(
            name = "Share",
            defaultWidth = 16.dp,
            defaultHeight = 16.dp,
            viewportWidth = 16f,
            viewportHeight = 16f
        ).apply {
            path(fill = SolidColor(Color(0xFF797979))) {
                moveTo(12f, 16f)
                curveTo(11.333f, 16f, 10.767f, 15.767f, 10.3f, 15.3f)
                curveTo(9.833f, 14.833f, 9.6f, 14.267f, 9.6f, 13.6f)
                curveTo(9.6f, 13.52f, 9.62f, 13.333f, 9.66f, 13.04f)
                lineTo(4.04f, 9.76f)
                curveTo(3.827f, 9.96f, 3.58f, 10.117f, 3.3f, 10.23f)
                curveTo(3.02f, 10.344f, 2.72f, 10.401f, 2.4f, 10.4f)
                curveTo(1.733f, 10.4f, 1.167f, 10.167f, 0.7f, 9.7f)
                curveTo(0.233f, 9.233f, 0f, 8.667f, 0f, 8f)
                curveTo(0f, 7.333f, 0.233f, 6.767f, 0.7f, 6.3f)
                curveTo(1.167f, 5.833f, 1.733f, 5.6f, 2.4f, 5.6f)
                curveTo(2.72f, 5.6f, 3.02f, 5.657f, 3.3f, 5.77f)
                curveTo(3.58f, 5.884f, 3.827f, 6.041f, 4.04f, 6.24f)
                lineTo(9.66f, 2.96f)
                curveTo(9.633f, 2.867f, 9.617f, 2.777f, 9.61f, 2.69f)
                curveTo(9.604f, 2.604f, 9.601f, 2.507f, 9.6f, 2.4f)
                curveTo(9.6f, 1.733f, 9.833f, 1.167f, 10.3f, 0.7f)
                curveTo(10.767f, 0.233f, 11.333f, 0f, 12f, 0f)
                curveTo(12.667f, 0f, 13.233f, 0.233f, 13.7f, 0.7f)
                curveTo(14.167f, 1.167f, 14.4f, 1.733f, 14.4f, 2.4f)
                curveTo(14.4f, 3.067f, 14.167f, 3.633f, 13.7f, 4.1f)
                curveTo(13.233f, 4.567f, 12.667f, 4.8f, 12f, 4.8f)
                curveTo(11.68f, 4.8f, 11.38f, 4.743f, 11.1f, 4.63f)
                curveTo(10.82f, 4.516f, 10.573f, 4.359f, 10.36f, 4.16f)
                lineTo(4.74f, 7.44f)
                curveTo(4.767f, 7.533f, 4.783f, 7.623f, 4.79f, 7.71f)
                curveTo(4.797f, 7.797f, 4.801f, 7.894f, 4.8f, 8f)
                curveTo(4.799f, 8.106f, 4.796f, 8.203f, 4.79f, 8.29f)
                curveTo(4.785f, 8.378f, 4.768f, 8.468f, 4.74f, 8.56f)
                lineTo(10.36f, 11.84f)
                curveTo(10.573f, 11.64f, 10.82f, 11.483f, 11.1f, 11.37f)
                curveTo(11.38f, 11.257f, 11.68f, 11.2f, 12f, 11.2f)
                curveTo(12.667f, 11.2f, 13.233f, 11.433f, 13.7f, 11.9f)
                curveTo(14.167f, 12.367f, 14.4f, 12.933f, 14.4f, 13.6f)
                curveTo(14.4f, 14.267f, 14.167f, 14.833f, 13.7f, 15.3f)
                curveTo(13.233f, 15.767f, 12.667f, 16f, 12f, 16f)
                close()
            }
        }.build()

        return _Share!!
    }

@Suppress("ObjectPropertyName")
private var _Share: ImageVector? = null
