package com.volna.app.core.config

/**
 * SCR-007: app-level values supplied by build or host runtime.
 *
 * This is not an A/B or remote feature flag layer; it only carries static links/version
 * that differ between distributions or environments.
 */
data class AppConfig(
    val rulesUrl: String? = null,
    val supportUrl: String? = null,
    val appVersion: String = "0.1.0",
)
