package com.volna.app.core.config

/**
 * Платформенный override базового URL API.
 *
 * На wasmJs читается из `window.__API_BASE_URL__` (задаётся в index.html при деплое —
 * см. client/webApp/src/wasmJsMain/resources/index.html). На остальных платформах — всегда
 * null, используется [com.volna.app.core.network.VolnaApiClient.DEFAULT_BASE_URL].
 */
expect fun platformApiBaseUrl(): String?
