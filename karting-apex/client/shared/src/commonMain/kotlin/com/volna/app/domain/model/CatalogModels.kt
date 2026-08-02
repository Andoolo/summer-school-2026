package com.volna.app.domain.model

import kotlinx.datetime.Instant

enum class RouteType {
    Novice,
    Experienced,
}

enum class SlotStatus {
    Scheduled,
    Cancelled,
}

data class RouteGeometry(
    val points: List<GeoPoint>,
)

data class Route(
    val id: RouteId,
    val name: String,
    val type: RouteType,
    val capacityCap: Int,
    val durationMin: Int,
    val geometry: RouteGeometry? = null,
)

data class Instructor(
    val id: InstructorId,
    val name: String,
)

data class GeoPoint(
    val lat: Double,
    val lng: Double,
)

/** Строка рекордов трассы: лучший круг гонщика по всем его заездам (картинг-фича). */
data class LeaderboardEntry(
    val position: Int,
    val name: String,
    val bestLapMs: Int,
    val laps: Int,
)

/** Направление прохождения круга. */
enum class TrackDirection {
    Clockwise,
    CounterClockwise,
}

/** Действующий рекорд круга на трассе. */
data class TrackRecord(
    val name: String,
    val lapMs: Int,
)

/**
 * Карточка трассы (картинг-фича F5, GET /routes/{id}).
 *
 * [lengthM], [corners], [direction] и [mainStraightM] считает бэкенд из той же
 * геометрии, которая рисуется на схеме, — поэтому цифры не могут разойтись с
 * контуром. Остальное — «паспорт» трассы из БД, он может быть не заполнен.
 */
data class TrackPassport(
    val id: RouteId,
    val name: String,
    val type: RouteType,
    val capacityCap: Int,
    val durationMin: Int,
    val geometry: RouteGeometry?,
    val lengthM: Int,
    val corners: Int,
    val direction: TrackDirection,
    val mainStraightM: Int,
    val surface: String? = null,
    val widthM: Double? = null,
    val elevationM: Double? = null,
    val openedYear: Int? = null,
    val kartModel: String? = null,
    val kartPowerHp: Int? = null,
    val record: TrackRecord? = null,
)

data class MeetingPoint(
    val title: String,
    val coordinates: GeoPoint,
)

data class Slot(
    val id: SlotId,
    val startAt: Instant,
    val route: Route,
    val instructor: Instructor,
    val totalSeats: Int,
    val freeSeats: Int,
    val freeRentalBoards: Int,
    val price: MoneyRub,
    val rentalPrice: MoneyRub,
    val meetingPoint: MeetingPoint,
    val status: SlotStatus,
)
