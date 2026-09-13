package com.volna.app.core.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

@Immutable
data class VolnaSpacing(
    val xxs: Dp = 4.dp,
    val xs: Dp = 8.dp,
    val sm: Dp = 12.dp,
    val md: Dp = 16.dp,
    val lg: Dp = 24.dp,
    val xl: Dp = 32.dp,
)

@Immutable
data class VolnaRadius(
    val xs: Dp = 4.dp,
    val sm: Dp = 8.dp,
    val md: Dp = 14.dp,
    val lg: Dp = 16.dp,
    val pill: Dp = 32.dp,
)

@Immutable
data class VolnaSizing(
    val screenMaxWidth: Dp = 393.dp,
    val contentWidth: Dp = 360.dp,
    val topTitleY: Dp = 80.dp,
    val authLogoY: Dp = 89.dp,
    val authTitleY: Dp = 190.dp,
    val authInputY: Dp = 315.dp,
    val authTermsY: Dp = 403.dp,
    val authButtonY: Dp = 464.dp,
    val listCardTopY: Dp = 136.dp,
    val listCardSecondY: Dp = 448.dp,
    val listCardHeight: Dp = 300.dp,
    val listStateMessageY: Dp = 364.dp,
    val filterIconX: Dp = 348.dp,
    val fieldHeight: Dp = 52.dp,
    val codeInputWidth: Dp = 84.dp,
    val buttonHeight: Dp = 56.dp,
    val backButtonY: Dp = 85.dp,
    val profileInfoY: Dp = 147.dp,
    val profileLinksY: Dp = 304.dp,
    val profileLogoutY: Dp = 662.dp,
    val stateMessageY: Dp = 190.dp,
    val navWidth: Dp = 300.dp,
    val navHeight: Dp = 56.dp,
    val navBottomPadding: Dp = 42.dp,
    val navContentBottomPadding: Dp = navHeight + navBottomPadding + 8.dp,
)

data class VolnaTokens(
    val colors: VolnaColorScheme,
    val spacing: VolnaSpacing = VolnaSpacing(),
    val radius: VolnaRadius = VolnaRadius(),
    val sizing: VolnaSizing = VolnaSizing(),
)

val LocalVolnaTokens = staticCompositionLocalOf {
    VolnaTokens(colors = VolnaLightColors)
}

@Composable
fun VolnaTheme(
    mode: ThemeMode = ThemeMode.System,
    content: @Composable () -> Unit,
) {
    val darkTheme = when (mode) {
        ThemeMode.System -> isSystemInDarkTheme()
        ThemeMode.Light -> false
        ThemeMode.Dark -> true
    }
    val tokens = VolnaTokens(colors = if (darkTheme) VolnaDarkColors else VolnaLightColors)
    androidx.compose.runtime.CompositionLocalProvider(LocalVolnaTokens provides tokens) {
        MaterialTheme(
            colorScheme = tokens.colors.toMaterialColorScheme(darkTheme),
            typography = Typography(),
        ) {
            // Text без явного цвета берёт LocalContentColor, а по умолчанию это чёрный —
            // его задаёт только Surface, которым экраны не обёрнуты. В светлой теме это
            // было незаметно, в тёмной такие заголовки пропадали на графите.
            androidx.compose.runtime.CompositionLocalProvider(
                androidx.compose.material3.LocalContentColor provides tokens.colors.textPrimary,
                content = content,
            )
        }
    }
}

object VolnaTheme {
    val tokens: VolnaTokens
        @Composable get() = LocalVolnaTokens.current
}

/**
 * Переводит палитру «Апекса» в схему Material.
 *
 * Контейнерные поверхности (surfaceContainer*) заданы явно. Раньше они брались из
 * стандартной схемы Material, а там они с сиреневым оттенком: диалоги и шторки
 * получали чужой цвет, а в тёмной теме — фиолетово-серый поверх графита.
 */
private fun VolnaColorScheme.toMaterialColorScheme(darkTheme: Boolean): ColorScheme {
    val base = if (darkTheme) {
        androidx.compose.material3.darkColorScheme()
    } else {
        androidx.compose.material3.lightColorScheme()
    }
    return base.copy(
        primary = brand,
        onPrimary = onBrand,
        background = background,
        onBackground = textPrimary,
        surface = surface,
        onSurface = textPrimary,
        surfaceVariant = surfaceVariant,
        onSurfaceVariant = textSecondary,
        surfaceTint = surface,
        surfaceContainerLowest = surface,
        surfaceContainerLow = surface,
        surfaceContainer = surface,
        surfaceContainerHigh = if (darkTheme) surfaceVariant else surface,
        surfaceContainerHighest = surfaceVariant,
        inverseSurface = textPrimary,
        inverseOnSurface = background,
        outline = outline,
        outlineVariant = border,
        error = error,
        onError = onError,
    )
}
