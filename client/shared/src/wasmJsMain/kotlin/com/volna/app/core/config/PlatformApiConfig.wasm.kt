package com.volna.app.core.config

@OptIn(ExperimentalWasmJsInterop::class)
@JsFun("() => (window.__API_BASE_URL__ || '')")
private external fun jsApiBaseUrl(): String

@OptIn(ExperimentalWasmJsInterop::class)
actual fun platformApiBaseUrl(): String? = jsApiBaseUrl().ifBlank { null }
