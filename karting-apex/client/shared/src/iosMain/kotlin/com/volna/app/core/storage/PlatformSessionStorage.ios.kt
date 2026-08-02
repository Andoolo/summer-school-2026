package com.volna.app.core.storage

import platform.Foundation.NSUserDefaults

actual object PlatformSessionStorage : SessionStorage {
    private val defaults: NSUserDefaults
        get() = NSUserDefaults.standardUserDefaults

    actual override suspend fun readToken(): String? =
        defaults.stringForKey(KEY_TOKEN)

    actual override suspend fun writeToken(token: String) {
        defaults.setObject(token, forKey = KEY_TOKEN)
    }

    actual override suspend fun clearToken() {
        defaults.removeObjectForKey(KEY_TOKEN)
    }

    private const val KEY_TOKEN = "volna_bearer_token"
}
