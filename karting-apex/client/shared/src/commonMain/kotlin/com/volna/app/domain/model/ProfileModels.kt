package com.volna.app.domain.model

import kotlinx.datetime.Instant
import kotlin.jvm.JvmInline

@JvmInline
value class Phone(val value: String) {
    init {
        require(Regex("^\\+[1-9]\\d{1,14}$").matches(value)) { "Phone must be E.164." }
    }
}

data class Client(
    val id: ClientId,
    val name: String?,
    val phone: Phone,
    val createdAt: Instant,
)
