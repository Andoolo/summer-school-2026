package com.volna.app.core.network

import com.volna.app.auth.SessionRepository
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.core.logging.AppLogger
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.plugins.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.logging.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.io.IOException
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json

class VolnaApiClient(
    @PublishedApi internal val sessionRepository: SessionRepository,
    @PublishedApi internal val baseUrl: String = DEFAULT_BASE_URL,
    @PublishedApi internal val httpClient: HttpClient = defaultHttpClient(),
) {
    internal suspend inline fun <reified T> send(
        path: String,
        authorized: Boolean = false,
        crossinline block: HttpRequestBuilder.() -> Unit = {},
    ): Result<T> = runCatching {
        val response = httpClient.request(baseUrl + path) {
            header(HttpHeaders.ContentType, ContentType.Application.Json)
            if (authorized) {
                sessionRepository.token()?.let { bearerAuth(it) }
            }
            block()
        }
        response.toResult<T>()
    }.fold(
        onSuccess = { it },
        onFailure = { failure ->
            AppLogger.e(failure, "Request failed: $path")
            Result.failure(failure.toAppFailureException())
        },
    )

    suspend fun sendUnit(
        path: String,
        authorized: Boolean = false,
        block: HttpRequestBuilder.() -> Unit = {},
    ): Result<Unit> = runCatching {
        val response = httpClient.request(baseUrl + path) {
            header(HttpHeaders.ContentType, ContentType.Application.Json)
            if (authorized) {
                sessionRepository.token()?.let { bearerAuth(it) }
            }
            block()
        }
        response.toUnitResult()
    }.fold(
        onSuccess = { it },
        onFailure = { failure ->
            AppLogger.e(failure, "Unit request failed: $path")
            Result.failure(failure.toAppFailureException())
        },
    )

    @PublishedApi
    internal suspend inline fun <reified T> HttpResponse.toResult(): Result<T> {
        if (status == HttpStatusCode.Unauthorized) {
            sessionRepository.clearToken()
            return Result.failure(AppFailureException(AppFailure.Unauthorized))
        }
        if (status.value in 200..299) {
            return Result.success(body())
        }
        return Result.failure(AppFailureException(readFailure()))
    }

    private suspend fun HttpResponse.toUnitResult(): Result<Unit> {
        if (status == HttpStatusCode.Unauthorized) {
            sessionRepository.clearToken()
            return Result.failure(AppFailureException(AppFailure.Unauthorized))
        }
        if (status.value in 200..299) {
            return Result.success(Unit)
        }
        return Result.failure(AppFailureException(readFailure()))
    }

    @PublishedApi
    internal suspend fun HttpResponse.readFailure(): AppFailure {
        val text = bodyAsText()
        val apiFailure = runCatching {
            json.decodeFromString<ApiErrorDto>(text).toFailure()
        }.onFailure { failure ->
            AppLogger.e(failure, "Failed to parse API error response")
        }.getOrNull()
        return apiFailure ?: AppFailure.Unknown
    }

    companion object {
        const val DEFAULT_BASE_URL = "http://localhost:8090"

        private val json = Json {
            ignoreUnknownKeys = true
        }

        fun defaultHttpClient(): HttpClient = HttpClient {
            expectSuccess = false
            install(ContentNegotiation) {
                json(json)
            }
            install(Logging) {
                logger = object : Logger {
                    override fun log(message: String) {
                        AppLogger.d(message)
                    }
                }
                level = LogLevel.ALL
                sanitizeHeader { header -> header == HttpHeaders.Authorization }
            }
            install(HttpTimeout) {
                requestTimeoutMillis = 20_000
                connectTimeoutMillis = 10_000
                socketTimeoutMillis = 20_000
            }
        }
    }
}

@PublishedApi
internal fun Throwable.toAppFailureException(): AppFailureException = when (this) {
    is AppFailureException -> this
    is TimeoutCancellationException -> AppFailureException(AppFailure.Timeout)
    is IOException -> AppFailureException(AppFailure.NetworkUnavailable)
    is SerializationException -> AppFailureException(AppFailure.Unknown)
    else -> AppFailureException(AppFailure.Unknown)
}
