package com.volna.app.core.theme

import androidx.compose.runtime.Immutable
import androidx.compose.ui.graphics.Color

@Immutable
data class VolnaColorScheme(
    val brand: Color,
    val onBrand: Color,
    val background: Color,
    val surface: Color,
    val surfaceVariant: Color,
    val textPrimary: Color,
    val textSecondary: Color,
    val border: Color,
    val success: Color,
    val warning: Color,
    val error: Color,
    val onError: Color,
    /**
     * Фон вокруг колонки приложения на широких экранах. Приложение спроектировано
     * под телефон; на десктопе оно остаётся колонкой, а подложка показывает, что
     * это осознанное решение, а не сжавшаяся вёрстка. На узких экранах не видна.
     */
    val backdrop: Color,
    /** «Ручка» шторки и прочие мелкие неинтерактивные элементы. */
    val handle: Color,
    /** Бейдж «Активна» у брони. */
    val successContainer: Color,
    val onSuccessContainer: Color,
    /**
     * Теги карточки заезда: тип трассы и её название. Фон и текст идут парой: в тёмной
     * теме яркий пастельный фон с тёмным текстом слепил бы, поэтому там фон
     * приглушённый, а текст светлый.
     */
    val tagRouteType: Color,
    val onTagRouteType: Color,
    val tagRouteName: Color,
    val onTagRouteName: Color,
    /** Схема трассы: плашка под схемой и само полотно. */
    val trackBackground: Color,
    val trackAsphalt: Color,
)

// Картинг-палитра «Апекс»: гоночный красный + графит (вместо бирюзы «Волны»).
val VolnaLightColors = VolnaColorScheme(
    brand = Color(0xFFE10600),
    onBrand = Color.White,
    background = Color.White,
    surface = Color.White,
    surfaceVariant = Color(0xFFF1F1F1),
    textPrimary = Color(0xFF15151E),
    textSecondary = Color(0xFF6E6E78),
    border = Color(0xFFE5E5E5),
    success = Color(0xFF237A4B),
    warning = Color(0xFF9A6400),
    error = Color(0xFFB3261E),
    onError = Color.White,
    backdrop = Color(0xFF15151E),
    handle = Color(0xFFCCCCCC),
    successContainer = Color(0xFFE4FFE5),
    onSuccessContainer = Color(0xFF007108),
    tagRouteType = Color(0xFF92FF9A),
    onTagRouteType = Color(0xFF15151E),
    tagRouteName = Color(0xFFFFF897),
    onTagRouteName = Color(0xFF15151E),
    trackBackground = Color(0xFFF2F2F4),
    trackAsphalt = Color(0xFF2B2B2E),
)

/**
 * Тёмная палитра, вариант A «Графит». Фон — тот же графит, что текст светлой темы и
 * подложка десктопа, так что новых базовых цветов не появилось.
 *
 * Карточки на ступень светлее фона: на чистом чёрном они бы с ним сливались.
 * Подложка десктопа, наоборот, темнее фона — иначе колонка приложения потерялась бы.
 *
 * Красный светлее, чем в светлой теме: исходный #E10600 на графите даёт контраст
 * 3.65:1 и выглядит тусклым, #FF2A1F — 4.84:1. Цена — белая надпись на красной
 * кнопке: 3.75:1, это норма только для крупного/жирного текста.
 */
val VolnaDarkColors = VolnaColorScheme(
    brand = Color(0xFFFF2A1F),
    onBrand = Color.White,
    background = Color(0xFF15151E),
    surface = Color(0xFF15151E),
    surfaceVariant = Color(0xFF1F1F2A),
    textPrimary = Color(0xFFF2F2F5),
    textSecondary = Color(0xFF9A9AA6),
    border = Color(0xFF2E2E3A),
    success = Color(0xFF4ADE80),
    warning = Color(0xFFFBBF24),
    error = Color(0xFFFF8A80),
    onError = Color(0xFF15151E),
    backdrop = Color(0xFF0B0B10),
    handle = Color(0xFF3A3A48),
    successContainer = Color(0xFF12291C),
    onSuccessContainer = Color(0xFF4ADE80),
    tagRouteType = Color(0xFF16301E),
    onTagRouteType = Color(0xFF86EFAC),
    tagRouteName = Color(0xFF332E14),
    onTagRouteName = Color(0xFFFDE68A),
    trackBackground = Color(0xFF2A2A36),
    trackAsphalt = Color(0xFF565664),
)
