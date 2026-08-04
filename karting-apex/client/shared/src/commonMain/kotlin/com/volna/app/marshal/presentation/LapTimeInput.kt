package com.volna.app.marshal.presentation

/**
 * Разбор времени круга из того, что маршал реально набирает на планшете.
 *
 * Секундомер показывает «41.890» или «1:01.234», и заставлять переводить это
 * в миллисекунды — прямой путь к опечаткам. Принимаем оба вида записи, а также
 * запятую вместо точки: на цифровой клавиатуре под рукой обычно она.
 *
 * Возвращает null, если строку разобрать нельзя. Проверку правдоподобия
 * (5 секунд…10 минут) делает бэкенд — здесь только разбор формата.
 */
internal fun parseLapTime(input: String): Int? {
    val normalized = input.trim().replace(',', '.')
    if (normalized.isEmpty()) return null

    val (minutesPart, secondsPart) = when (val colon = normalized.indexOf(':')) {
        -1 -> "0" to normalized
        else -> normalized.substring(0, colon) to normalized.substring(colon + 1)
    }

    val minutes = minutesPart.toIntOrNull() ?: return null
    if (minutes < 0) return null

    val dot = secondsPart.indexOf('.')
    val wholeSeconds: String
    val fraction: String
    if (dot == -1) {
        wholeSeconds = secondsPart
        fraction = ""
    } else {
        wholeSeconds = secondsPart.substring(0, dot)
        fraction = secondsPart.substring(dot + 1)
    }

    val seconds = wholeSeconds.toIntOrNull() ?: return null
    if (seconds < 0 || (minutes > 0 && seconds >= 60)) return null
    if (fraction.any { !it.isDigit() }) return null
    if (fraction.length > 3) return null

    // «41.5» — это 500 миллисекунд, а не 5: дописываем нули справа, а не слева.
    val millis = when (fraction.length) {
        0 -> 0
        else -> fraction.padEnd(3, '0').toIntOrNull() ?: return null
    }

    return minutes * 60_000 + seconds * 1_000 + millis
}

/** 41890 → «41.890», 61234 → «1:01.234». Обратная операция к [parseLapTime]. */
internal fun formatLapTime(ms: Int): String {
    val minutes = ms / 60_000
    val seconds = (ms % 60_000) / 1_000
    val millis = (ms % 1_000).toString().padStart(3, '0')
    return if (minutes > 0) {
        "$minutes:${seconds.toString().padStart(2, '0')}.$millis"
    } else {
        "$seconds.$millis"
    }
}
