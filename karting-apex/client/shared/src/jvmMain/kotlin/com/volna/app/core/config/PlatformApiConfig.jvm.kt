package com.volna.app.core.config

/** На JVM (цель существует только ради тестов) адрес API не переопределяется. */
actual fun platformApiBaseUrl(): String? = null
