package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Slot struct {
	ID               string
	RouteID          string
	RouteName        string
	RouteType        string
	RouteCapacityCap int
	RouteDurationMin int
	RouteGeometry    []byte // сырой jsonb геометрии маршрута ([[lat,lng],...] или null)
	InstructorID     string
	InstructorName   string
	StartAt          time.Time
	TotalSeats       int
	FreeSeats        int
	FreeRentalBoards int
	Price            int
	RentalPrice      int
	MeetingPoint     string
	MeetingPointLat  float64
	MeetingPointLng  float64
	Status           string
}

type SlotFilters struct {
	DateFrom      *time.Time
	DateTo        *time.Time
	RouteTypes    []string
	InstructorIDs []string
	OnlyAvailable bool
	Limit         int
	Offset        int
}

type SlotList struct {
	Items []Slot
	Total int
}

type SlotRepository struct {
	db *pgxpool.Pool
}

func NewSlotRepository(db *pgxpool.Pool) *SlotRepository {
	return &SlotRepository{db: db}
}

func (r *SlotRepository) List(ctx context.Context, filters SlotFilters) (SlotList, error) {
	where, args := slotWhere(filters)
	limit := filters.Limit
	if limit == 0 {
		limit = 20
	}
	offset := filters.Offset

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM slots s JOIN routes r ON r.id = s.route_id JOIN instructors i ON i.id = s.instructor_id`+where, args...).Scan(&total); err != nil {
		return SlotList{}, fmt.Errorf("count slots: %w", err)
	}

	queryArgs := append(args, limit, offset)
	rows, err := r.db.Query(ctx, `
SELECT
    s.id::text,
    r.id::text,
    r.name,
    r.type,
    r.capacity_cap,
    r.duration_min,
    r.geometry,
    i.id::text,
    i.name,
    s.start_at,
    s.total_seats,
    s.free_seats,
    s.free_rental_boards,
    s.price,
    s.rental_price,
    s.meeting_point,
    s.meeting_point_lat,
    s.meeting_point_lng,
    s.status
FROM slots s
JOIN routes r ON r.id = s.route_id
JOIN instructors i ON i.id = s.instructor_id
`+where+`
ORDER BY s.start_at ASC
LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), queryArgs...)
	if err != nil {
		return SlotList{}, fmt.Errorf("query slots: %w", err)
	}
	defer rows.Close()

	slots := make([]Slot, 0)
	for rows.Next() {
		var slot Slot
		if err := rows.Scan(
			&slot.ID,
			&slot.RouteID,
			&slot.RouteName,
			&slot.RouteType,
			&slot.RouteCapacityCap,
			&slot.RouteDurationMin,
			&slot.RouteGeometry,
			&slot.InstructorID,
			&slot.InstructorName,
			&slot.StartAt,
			&slot.TotalSeats,
			&slot.FreeSeats,
			&slot.FreeRentalBoards,
			&slot.Price,
			&slot.RentalPrice,
			&slot.MeetingPoint,
			&slot.MeetingPointLat,
			&slot.MeetingPointLng,
			&slot.Status,
		); err != nil {
			return SlotList{}, fmt.Errorf("scan slot: %w", err)
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return SlotList{}, fmt.Errorf("iterate slots: %w", err)
	}

	return SlotList{Items: slots, Total: total}, nil
}

func (r *SlotRepository) GetByID(ctx context.Context, id string) (Slot, bool, error) {
	var slot Slot
	err := r.db.QueryRow(ctx, `
SELECT
    s.id::text,
    r.id::text,
    r.name,
    r.type,
    r.capacity_cap,
    r.duration_min,
    r.geometry,
    i.id::text,
    i.name,
    s.start_at,
    s.total_seats,
    s.free_seats,
    s.free_rental_boards,
    s.price,
    s.rental_price,
    s.meeting_point,
    s.meeting_point_lat,
    s.meeting_point_lng,
    s.status
FROM slots s
JOIN routes r ON r.id = s.route_id
JOIN instructors i ON i.id = s.instructor_id
WHERE s.id = $1`, id).Scan(
		&slot.ID,
		&slot.RouteID,
		&slot.RouteName,
		&slot.RouteType,
		&slot.RouteCapacityCap,
		&slot.RouteDurationMin,
		&slot.RouteGeometry,
		&slot.InstructorID,
		&slot.InstructorName,
		&slot.StartAt,
		&slot.TotalSeats,
		&slot.FreeSeats,
		&slot.FreeRentalBoards,
		&slot.Price,
		&slot.RentalPrice,
		&slot.MeetingPoint,
		&slot.MeetingPointLat,
		&slot.MeetingPointLng,
		&slot.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Slot{}, false, nil
	}
	if err != nil {
		return Slot{}, false, fmt.Errorf("get slot: %w", err)
	}
	return slot, true, nil
}

// LeaderboardEntry — строка рекордов трассы: лучший круг клиента по всем его заездам.
type LeaderboardEntry struct {
	ClientName string
	BestLapMs  int
	Laps       int
}

// Leaderboard возвращает топ лучших кругов по трассе (конфигурации), found=false — трасса не найдена.
func (r *SlotRepository) Leaderboard(ctx context.Context, routeID string, limit int) (string, []LeaderboardEntry, bool, error) {
	var routeName string
	err := r.db.QueryRow(ctx, `SELECT name FROM routes WHERE id = $1`, routeID).Scan(&routeName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, fmt.Errorf("get route for leaderboard: %w", err)
	}

	rows, err := r.db.Query(ctx, `
SELECT COALESCE(c.name, 'Гонщик'), MIN(lr.lap_time_ms) AS best_lap_ms, COUNT(lr.id) AS laps
FROM lap_results lr
JOIN bookings b ON b.id = lr.booking_id
JOIN slots s ON s.id = b.slot_id
JOIN clients c ON c.id = b.client_id
WHERE s.route_id = $1
GROUP BY c.id, c.name
ORDER BY best_lap_ms ASC
LIMIT $2`, routeID, limit)
	if err != nil {
		return "", nil, false, fmt.Errorf("query leaderboard: %w", err)
	}
	defer rows.Close()

	entries := make([]LeaderboardEntry, 0)
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.ClientName, &e.BestLapMs, &e.Laps); err != nil {
			return "", nil, false, fmt.Errorf("scan leaderboard entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return "", nil, false, fmt.Errorf("iterate leaderboard: %w", err)
	}
	return routeName, entries, true, nil
}

// RoutePassport — «паспортные» характеристики трассы из БД (F5). Всё, что
// выводится из геометрии (длина круга, повороты, направление, главная прямая),
// здесь намеренно отсутствует: это считает internal/track по тому же контуру,
// который рисуется на схеме.
type RoutePassport struct {
	ID          string
	Name        string
	Type        string
	CapacityCap int
	DurationMin int
	Geometry    []byte // сырой jsonb ([[lat,lng],...] или null)

	Surface     *string
	WidthM      *float64
	ElevationM  *float64
	OpenedYear  *int
	KartModel   *string
	KartPowerHP *int

	// Рекорд круга по этой трассе, если заезды уже были.
	RecordHolder *string
	RecordLapMs  *int
}

// RoutePassportByID отдаёт паспорт трассы вместе с текущим рекордом круга.
// Второе значение — false, если трассы с таким id нет.
func (r *SlotRepository) RoutePassportByID(ctx context.Context, routeID string) (RoutePassport, bool, error) {
	var p RoutePassport
	err := r.db.QueryRow(ctx, `
SELECT
    rt.id::text, rt.name, rt.type, rt.capacity_cap, rt.duration_min, rt.geometry,
    rt.surface, rt.width_m, rt.elevation_m, rt.opened_year, rt.kart_model, rt.kart_power_hp,
    record.name, record.lap_ms
FROM routes rt
LEFT JOIN LATERAL (
    SELECT COALESCE(c.name, 'Гонщик') AS name, lr.lap_time_ms AS lap_ms
    FROM lap_results lr
    JOIN bookings b ON b.id = lr.booking_id
    JOIN slots s ON s.id = b.slot_id
    JOIN clients c ON c.id = b.client_id
    WHERE s.route_id = rt.id
    ORDER BY lr.lap_time_ms ASC
    LIMIT 1
) record ON true
WHERE rt.id = $1`, routeID).Scan(
		&p.ID, &p.Name, &p.Type, &p.CapacityCap, &p.DurationMin, &p.Geometry,
		&p.Surface, &p.WidthM, &p.ElevationM, &p.OpenedYear, &p.KartModel, &p.KartPowerHP,
		&p.RecordHolder, &p.RecordLapMs,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutePassport{}, false, nil
	}
	if err != nil {
		return RoutePassport{}, false, fmt.Errorf("get route passport: %w", err)
	}
	return p, true, nil
}

// LapEntryTarget — заезд, в который маршал вносит времена кругов (F6).
type LapEntryTarget struct {
	BookingID    string
	BookingState string
	ClientName   string
	RouteID      string
	RouteName    string
	SlotStartAt  time.Time
}

// BookingForLapEntry находит бронь и контекст, нужный для проверок перед записью
// результатов. Второе значение — false, если брони с таким id нет.
func (r *SlotRepository) BookingForLapEntry(ctx context.Context, bookingID string) (LapEntryTarget, bool, error) {
	var t LapEntryTarget
	err := r.db.QueryRow(ctx, `
SELECT b.id::text, b.status, COALESCE(c.name, 'Гонщик'), rt.id::text, rt.name, s.start_at
FROM bookings b
JOIN slots s ON s.id = b.slot_id
JOIN routes rt ON rt.id = s.route_id
JOIN clients c ON c.id = b.client_id
WHERE b.id = $1`, bookingID).Scan(
		&t.BookingID, &t.BookingState, &t.ClientName, &t.RouteID, &t.RouteName, &t.SlotStartAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LapEntryTarget{}, false, nil
	}
	if err != nil {
		return LapEntryTarget{}, false, fmt.Errorf("get booking for lap entry: %w", err)
	}
	return t, true, nil
}

// ReplaceLapResults полностью заменяет времена кругов брони на переданные.
//
// Именно замена, а не добавление: маршал вносит результаты заезда целиком и может
// исправить опечатку, отправив список заново. При добавлении повторная отправка
// удвоила бы круги и испортила таблицу рекордов.
//
// Возвращает лучший круг по трассе ДО этой записи — по нему видно, побит ли рекорд.
func (r *SlotRepository) ReplaceLapResults(ctx context.Context, bookingID, routeID string, lapTimesMs []int) (previousBestMs *int, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin replace lap results: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var best *int
	if err := tx.QueryRow(ctx, `
SELECT MIN(lr.lap_time_ms)
FROM lap_results lr
JOIN bookings b ON b.id = lr.booking_id
JOIN slots s ON s.id = b.slot_id
WHERE s.route_id = $1`, routeID).Scan(&best); err != nil {
		return nil, fmt.Errorf("read previous best lap: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM lap_results WHERE booking_id = $1`, bookingID); err != nil {
		return nil, fmt.Errorf("clear lap results: %w", err)
	}
	for i, ms := range lapTimesMs {
		if _, err := tx.Exec(ctx, `
INSERT INTO lap_results (booking_id, lap_number, lap_time_ms) VALUES ($1, $2, $3)`,
			bookingID, i+1, ms); err != nil {
			return nil, fmt.Errorf("insert lap %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit replace lap results: %w", err)
	}
	return best, nil
}

// RaceParticipant — участник прошедшего заезда глазами маршала (F6).
type RaceParticipant struct {
	BookingID  string
	RacerName  string
	SeatsCount int
	Laps       int
	BestLapMs  *int
}

// RaceRoster отдаёт список участников заезда с уже внесёнными результатами.
// Без него маршалу пришлось бы вводить идентификатор брони вручную.
// Второе значение — false, если заезда с таким id нет.
func (r *SlotRepository) RaceRoster(ctx context.Context, slotID string) (string, time.Time, []RaceParticipant, bool, error) {
	var routeName string
	var startAt time.Time
	err := r.db.QueryRow(ctx, `
SELECT rt.name, s.start_at
FROM slots s JOIN routes rt ON rt.id = s.route_id
WHERE s.id = $1`, slotID).Scan(&routeName, &startAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, nil, false, nil
	}
	if err != nil {
		return "", time.Time{}, nil, false, fmt.Errorf("get slot for roster: %w", err)
	}

	rows, err := r.db.Query(ctx, `
SELECT b.id::text, COALESCE(c.name, 'Гонщик'), b.seats_count,
       count(lr.id), MIN(lr.lap_time_ms)
FROM bookings b
JOIN clients c ON c.id = b.client_id
LEFT JOIN lap_results lr ON lr.booking_id = b.id
WHERE b.slot_id = $1 AND b.status = 'active'
GROUP BY b.id, c.name, b.seats_count, b.created_at
ORDER BY b.created_at ASC`, slotID)
	if err != nil {
		return "", time.Time{}, nil, false, fmt.Errorf("query roster: %w", err)
	}
	defer rows.Close()

	participants := make([]RaceParticipant, 0)
	for rows.Next() {
		var p RaceParticipant
		if err := rows.Scan(&p.BookingID, &p.RacerName, &p.SeatsCount, &p.Laps, &p.BestLapMs); err != nil {
			return "", time.Time{}, nil, false, fmt.Errorf("scan participant: %w", err)
		}
		participants = append(participants, p)
	}
	if err := rows.Err(); err != nil {
		return "", time.Time{}, nil, false, fmt.Errorf("iterate roster: %w", err)
	}
	return routeName, startAt, participants, true, nil
}

func slotWhere(filters SlotFilters) (string, []any) {
	conditions := make([]string, 0)
	args := make([]any, 0)
	add := func(condition string, arg any) {
		args = append(args, arg)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if filters.DateFrom != nil {
		add("s.start_at >= $%d", *filters.DateFrom)
	}
	if filters.DateTo != nil {
		add("s.start_at <= $%d", *filters.DateTo)
	}
	if len(filters.RouteTypes) > 0 {
		add("r.type = ANY($%d)", filters.RouteTypes)
	}
	if len(filters.InstructorIDs) > 0 {
		add("i.id = ANY($%d::uuid[])", filters.InstructorIDs)
	}
	if filters.OnlyAvailable {
		conditions = append(conditions, "s.free_seats > 0")
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}
