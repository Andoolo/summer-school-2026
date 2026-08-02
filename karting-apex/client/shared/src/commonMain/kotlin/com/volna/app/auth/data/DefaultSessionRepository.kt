package com.volna.app.auth.data

import com.volna.app.auth.SessionRepository
import com.volna.app.core.storage.SessionStorage

class DefaultSessionRepository(
    private val storage: SessionStorage,
) : SessionRepository {
    override suspend fun token(): String? = storage.readToken()

    override suspend fun saveToken(token: String) {
        storage.writeToken(token)
    }

    override suspend fun clearToken() {
        storage.clearToken()
    }
}
