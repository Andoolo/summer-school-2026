package com.volna.app.core.phone

private const val DEFAULT_MAX_PHONE_DIGITS = 10

fun sanitizePhoneInput(
    input: String,
    maxDigits: Int = DEFAULT_MAX_PHONE_DIGITS,
): String {
    val digits = input.filter(Char::isDigit)
    if (digits.length >= 11 && digits.startsWith("7")) {
        val withoutPrefix = digits.drop(1)
        return if (maxDigits > 0) withoutPrefix.take(maxDigits) else withoutPrefix
    }
    return if (maxDigits > 0) digits.take(maxDigits) else digits
}

fun normalizePhoneE164(phone: String): String {
    val digits = phone.filter(Char::isDigit)
    return when {
        digits.isBlank() -> ""
        digits.length == 10 -> "+7$digits"
        digits.length == 11 && digits.startsWith("7") -> "+$digits"
        phone.trim().startsWith("+") -> "+$digits"
        else -> "+$digits"
    }
}

fun formatPhoneNumber(
    input: String,
    maxDigits: Int = DEFAULT_MAX_PHONE_DIGITS,
): String = formatPhoneNumberWithPositions(input, maxDigits).text

internal data class PhoneFormatResult(
    val text: String,
    val digitPositions: IntArray,
)

internal fun formatPhoneNumberWithPositions(
    input: String,
    maxDigits: Int = DEFAULT_MAX_PHONE_DIGITS,
): PhoneFormatResult {
    val digits = sanitizePhoneInput(input, maxDigits)
    if (digits.isEmpty()) {
        return PhoneFormatResult("", intArrayOf())
    }

    val builder = StringBuilder()
    val positions = ArrayList<Int>(digits.length)

    builder.append("+7 ")
    for (index in digits.indices) {
        if (index == 0) {
            builder.append("(")
        }
        builder.append(digits[index])
        positions.add(builder.length - 1)

        val hasNext = index < digits.lastIndex
        when (index) {
            2 -> if (hasNext) builder.append(") ")
            5 -> if (hasNext) builder.append("-")
            7 -> if (hasNext) builder.append("-")
        }
    }

    return PhoneFormatResult(builder.toString(), positions.toIntArray())
}

fun String.isRussianPhoneInputComplete(): Boolean = sanitizePhoneInput(this).length == DEFAULT_MAX_PHONE_DIGITS
