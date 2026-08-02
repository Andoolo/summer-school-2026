package com.volna.app.booking.data

import com.volna.app.booking.IdempotencyKey
import com.volna.app.booking.IdempotencyKeyFactory
import kotlin.random.Random

class RandomIdempotencyKeyFactory : IdempotencyKeyFactory {
    override fun next(): IdempotencyKey = IdempotencyKey(
        buildString {
            appendHex(8)
            append('-')
            appendHex(4)
            append("-4")
            appendHex(3)
            append('-')
            append("89ab"[Random.nextInt(4)])
            appendHex(3)
            append('-')
            appendHex(12)
        },
    )

    private fun StringBuilder.appendHex(length: Int) {
        repeat(length) {
            append("0123456789abcdef"[Random.nextInt(16)])
        }
    }
}
