package com.volna.app.catalog

import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.domain.model.SlotId
import kotlinx.datetime.Instant

/**
 * Лист ожидания на заполненный заезд. Когда место освобождается, первому в очереди
 * приходит сообщение в Telegram; на запись у него [OfferMinutes] минут, потом место
 * предлагается следующему. Место при этом не закреплено — записаться может любой.
 */
enum class WaitlistEntryStatus {
    /** Стоит в очереди. */
    Waiting,

    /** Место освободилось, предложение отправлено. */
    Offered,
}

data class WaitlistEntry(
    val status: WaitlistEntryStatus,
    val seatsCount: Int,
    /** Место в очереди с 1; у получивших предложение — 0. */
    val position: Int,
    val offerExpiresAt: Instant?,
)

data class WaitlistStatus(
    val entry: WaitlistEntry?,
    val telegramLinked: Boolean,
    val notificationsEnabled: Boolean,
)

interface WaitlistRepository {
    suspend fun status(slotId: SlotId): Result<WaitlistStatus>
    suspend fun join(slotId: SlotId, seatsCount: Int): Result<WaitlistEntry>
    suspend fun leave(slotId: SlotId): Result<Unit>
}

const val OfferMinutes = 15

/** Лист ожидания выключен (например, в тестах, где он не нужен): секция не показывается. */
object DisabledWaitlistRepository : WaitlistRepository {
    private fun <T> unavailable(): Result<T> =
        Result.failure(AppFailureException(AppFailure.Api(ApiErrorCode.NotFound, "Лист ожидания недоступен")))

    override suspend fun status(slotId: SlotId): Result<WaitlistStatus> = unavailable()
    override suspend fun join(slotId: SlotId, seatsCount: Int): Result<WaitlistEntry> = unavailable()
    override suspend fun leave(slotId: SlotId): Result<Unit> = unavailable()
}
