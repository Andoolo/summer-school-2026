package com.volna.app.core.storage

/**
 * Хранилище токена маршала (F6) — отдельно от клиентской сессии.
 *
 * Это не сессия пользователя, а настройка рабочего места: планшет маршала
 * настраивают один раз. Смешивать с [SessionStorage] нельзя — выход гонщика из
 * аккаунта не должен сбрасывать режим маршала, и наоборот.
 */
expect object PlatformMarshalStorage : MarshalTokenStorage {
    override suspend fun readToken(): String?
    override suspend fun writeToken(token: String)
    override suspend fun clearToken()
}

interface MarshalTokenStorage {
    suspend fun readToken(): String?
    suspend fun writeToken(token: String)
    suspend fun clearToken()
}
