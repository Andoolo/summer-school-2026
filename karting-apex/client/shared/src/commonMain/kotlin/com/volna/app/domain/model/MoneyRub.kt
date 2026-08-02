package com.volna.app.domain.model

import kotlin.jvm.JvmInline

@JvmInline
value class MoneyRub(val value: Int) {
    init {
        require(value >= 0) { "Money value must be non-negative." }
    }

    operator fun plus(other: MoneyRub): MoneyRub = MoneyRub(value + other.value)

    operator fun times(multiplier: Int): MoneyRub {
        require(multiplier >= 0) { "Multiplier must be non-negative." }
        return MoneyRub(value * multiplier)
    }
}
