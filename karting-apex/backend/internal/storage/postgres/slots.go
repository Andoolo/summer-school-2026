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
