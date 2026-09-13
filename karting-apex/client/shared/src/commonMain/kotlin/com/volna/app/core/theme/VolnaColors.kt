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
    /** Тонкие разделители. Для границ элементов управления — [outline]. */
    val border: Color,
    /**
     * Граница элементов управления: полей ввода, переключателя, контурных кнопок.
     * Отдельно от [border], потому что WCAG 1.4.11 требует для неё контраст 3:1,
     * а разделители с таким контрастом выглядели бы тяжело.
     */
    val outline: Color,
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
    // #DC0600, а не #E10600: красная ссылка на серой карточке давала 4.40:1 при норме 4.5.
    brand = Color(0xFFDC0600),
    onBrand = Color.White,
    background = Color.White,
    surface = Color.White,
    surfaceVariant = Color(0xFFF1F1F1),
    textPrimary = Color(0xFF15151E),
    // #66666F: прежний #6E6E78 на карточке #F1F1F1 давал 4.46:1 при норме 4.5.
    textSecondary = Color(0xFF66666F),
    border = Color(0xFFE5E5E5),
    outline = Color(0xFF85858F),
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
 * Красный светлее, чем в светлой теме: красный текст должен читаться и на фоне, и на
 * карточке (#FF4136: 5.23:1 и 4.71:1). Но на таком красном белая надпись даёт лишь
 * 3.46:1 — ниже нормы 4.5, причём общего решения нет: чем светлее красный для
 * ссылок, тем хуже на нём белый. Поэтому надписи на красных кнопках в тёмной теме
 * графитовые (5.23:1), как принято в тёмных схемах Material.
 */
val VolnaDarkColors = VolnaColorScheme(
    brand = Color(0xFFFF4136),
    onBrand = Color(0xFF15151E),
    background = Color(0xFF15151E),
    surface = Color(0xFF15151E),
    surfaceVariant = Color(0xFF1F1F2A),
    textPrimary = Color(0xFFF2F2F5),
    textSecondary = Color(0xFF9A9AA6),
    border = Color(0xFF2E2E3A),
    outline = Color(0xFF72727F),
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
