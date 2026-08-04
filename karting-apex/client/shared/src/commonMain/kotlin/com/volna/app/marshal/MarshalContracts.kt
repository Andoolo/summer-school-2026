package com.volna.app.marshal

import com.volna.app.domain.model.BookingId
import com.volna.app.domain.model.SlotId
import kotlinx.datetime.Instant

/** Участник прошедшего заезда глазами маршала (F6). */
data class RaceParticipant(
    val bookingId: BookingId,
    val racerName: String,
    val seats: Int,
    val laps: Int,
    val bestLapMs: Int?,
)

/** Состав заезда: кого маршал видит в списке при вводе результатов. */
data class RaceRoster(
    val slotId: SlotId,
    val routeName: String,
    val startAt: Instant,
    val alreadyRaced: Boolean,
    val participants: List<RaceParticipant>,
)

/** Итог внесения результатов — по нему показываем, побит ли рекорд трассы. */
data class LapEntryResult(
    val racerName: String,
    val routeName: String,
    val laps: Int,
    val bestLapMs: Int,
    val isTrackRecord: Boolean,
)

/**
 * Операции маршала. Все требуют токен: он передаётся заголовком, а не через
 * клиентскую сессию — это отдельная модель доступа (см. F6).
 */
interface MarshalRepository {
    suspend fun raceRoster(token: String, slotId: SlotId): Result<RaceRoster>

    suspend fun submitLapResults(
        token: String,
        bookingId: BookingId,
        lapTimesMs: List<Int>,
    ): Result<LapEntryResult>
}
