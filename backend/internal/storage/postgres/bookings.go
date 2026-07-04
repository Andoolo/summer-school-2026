package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/service/booking"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepository struct {
	db *pgxpool.Pool
}

func NewBookingRepository(db *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) ClientBySessionTokenHash(ctx context.Context, tokenHash string) (booking.Client, bool, error) {
	var client booking.Client
	err := r.db.QueryRow(ctx, `
SELECT c.id::text
FROM auth_sessions s
JOIN clients c ON c.id = s.client_id
WHERE s.token_hash = $1
  AND s.revoked_at IS NULL
  AND s.expires_at > now()
  AND c.deleted_at IS NULL`, tokenHash).Scan(&client.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return booking.Client{}, false, nil
	}
	if err != nil {
		return booking.Client{}, false, fmt.Errorf("query booking client by session: %w", err)
	}
	return client, true, nil
}

func (r *BookingRepository) Create(ctx context.Context, clientID string, command booking.CreateCommand, requestHash string, now time.Time) (booking.Booking, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return booking.Booking{}, fmt.Errorf("begin create booking: %w", err)
	}
	defer tx.Rollback(ctx)

	if command.IdempotencyKey != "" {
		existing, ok, err := lockIdempotencyKey(ctx, tx, clientID, command.IdempotencyKey)
		if err != nil {
			return booking.Booking{}, err
		}
		if ok {
			if existing.RequestHash != requestHash {
				return booking.Booking{}, booking.ErrIdempotencyConflict
			}
			if existing.BookingID != "" {
				created, found, err := bookingByID(ctx, tx, existing.BookingID)
				if err != nil {
					return booking.Booking{}, err
				}
				if found {
					if err := tx.Commit(ctx); err != nil {
						return booking.Booking{}, fmt.Errorf("commit idempotent booking: %w", err)
					}
					return created, nil
				}
			}
		} else if err := insertIdempotencyKey(ctx, tx, clientID, command.IdempotencyKey, requestHash, now); err != nil {
			return booking.Booking{}, err
		}
	}

	slot, err := lockSlot(ctx, tx, command.SlotID)
	if err != nil {
		return booking.Booking{}, err
	}
	if slot.Status == "cancelled" {
		return booking.Booking{}, booking.AvailabilityError{Err: booking.ErrSlotCancelled, Availability: booking.Availability{AvailableSeats: slot.FreeSeats, AvailableRentalBoards: slot.FreeRentalBoards}}
	}
	if !now.Before(slot.StartAt) {
		return booking.Booking{}, booking.ErrSlotStarted
	}
	if slot.FreeSeats < command.SeatsCount || slot.FreeRentalBoards < command.RentalCount {
		return booking.Booking{}, booking.AvailabilityError{Err: booking.ErrSlotFull, Availability: booking.Availability{AvailableSeats: slot.FreeSeats, AvailableRentalBoards: slot.FreeRentalBoards}}
	}

	var alreadyBooked bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM bookings
    WHERE client_id = $1 AND slot_id = $2 AND status = 'active'
)`, clientID, command.SlotID).Scan(&alreadyBooked); err != nil {
		return booking.Booking{}, fmt.Errorf("check double booking: %w", err)
	}
	if alreadyBooked {
		return booking.Booking{}, booking.ErrDoubleBooking
	}

	if _, err := tx.Exec(ctx, `
UPDATE slots
SET free_seats = free_seats - $2,
    free_rental_boards = free_rental_boards - $3
WHERE id = $1`, command.SlotID, command.SeatsCount, command.RentalCount); err != nil {
		return booking.Booking{}, fmt.Errorf("update slot availability: %w", err)
	}

	var bookingID string
	var createdAt time.Time
	err = tx.QueryRow(ctx, `
INSERT INTO bookings (slot_id, client_id, seats_count, rental_count, status, created_at)
VALUES ($1, $2, $3, $4, 'active', $5)
RETURNING id::text, created_at`, command.SlotID, clientID, command.SeatsCount, command.RentalCount, now).Scan(&bookingID, &createdAt)
	if err != nil {
		if isUniqueViolation(err) {
			return booking.Booking{}, booking.ErrDoubleBooking
		}
		return booking.Booking{}, fmt.Errorf("insert booking: %w", err)
	}

	created, found, err := bookingByID(ctx, tx, bookingID)
	if err != nil {
		return booking.Booking{}, err
	}
	if !found {
		return booking.Booking{}, fmt.Errorf("created booking not found")
	}
	created.CreatedAt = createdAt

	if command.IdempotencyKey != "" {
		body, err := json.Marshal(map[string]string{"booking_id": bookingID})
		if err != nil {
			return booking.Booking{}, fmt.Errorf("marshal idempotency response: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE idempotency_keys
SET response_status = 201, response_body = $4
WHERE client_id = $1 AND key = $2 AND request_hash = $3`, clientID, command.IdempotencyKey, requestHash, body); err != nil {
			return booking.Booking{}, fmt.Errorf("store idempotency response: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return booking.Booking{}, fmt.Errorf("commit create booking: %w", err)
	}
	return created, nil
}

func (r *BookingRepository) List(ctx context.Context, clientID string, command booking.ListCommand) (booking.BookingList, error) {
	where := "WHERE b.client_id = $1"
	args := []any{clientID}
	if command.Status != nil {
		where += " AND b.status = $2"
		args = append(args, *command.Status)
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM bookings b `+where, args...).Scan(&total); err != nil {
		return booking.BookingList{}, fmt.Errorf("count bookings: %w", err)
	}

	queryArgs := append(args, command.Limit, command.Offset)
	// Двунаправленная сортировка под пагинацию (SCR-005, AC-002/AC-003):
	//   1) предстоящие (start_at >= now) идут раньше прошедших;
	//   2) внутри предстоящих — по возрастанию start_at (ближайшая прогулка первой, AC-002);
	//   3) внутри прошедших — по убыванию start_at (свежая первой, AC-003).
	// Раньше было плоское ORDER BY s.start_at DESC: первая страница набивалась дальними будущими
	// бронями, ближайшая предстоящая не попадала в первый экран ленивой загрузки (R-025).
	rows, err := r.db.Query(ctx, bookingSelectSQL()+`
`+where+`
ORDER BY (CASE WHEN s.start_at >= NOW() THEN 0 ELSE 1 END),
         CASE WHEN s.start_at >= NOW() THEN s.start_at END ASC,
         s.start_at DESC,
         b.created_at DESC
LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), queryArgs...)
	if err != nil {
		return booking.BookingList{}, fmt.Errorf("query bookings: %w", err)
	}
	defer rows.Close()

	items := make([]booking.Booking, 0)
	for rows.Next() {
		item, err := scanBooking(rows)
		if err != nil {
			return booking.BookingList{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return booking.BookingList{}, fmt.Errorf("iterate bookings: %w", err)
	}
	return booking.BookingList{Items: items, Total: total}, nil
}

func (r *BookingRepository) Get(ctx context.Context, clientID, bookingID string) (booking.Booking, error) {
	var ownerID string
	err := r.db.QueryRow(ctx, `SELECT client_id::text FROM bookings WHERE id = $1`, bookingID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return booking.Booking{}, booking.ErrNotFound
	}
	if err != nil {
		return booking.Booking{}, fmt.Errorf("query booking owner: %w", err)
	}
	if ownerID != clientID {
		return booking.Booking{}, booking.ErrForbidden
	}

	created, found, err := bookingByID(ctx, r.db, bookingID)
	if err != nil {
		return booking.Booking{}, err
	}
	if !found {
		return booking.Booking{}, booking.ErrNotFound
	}
	return created, nil
}

func (r *BookingRepository) Cancel(ctx context.Context, clientID, bookingID string, now time.Time) (booking.Booking, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return booking.Booking{}, fmt.Errorf("begin cancel booking: %w", err)
	}
	defer tx.Rollback(ctx)

	var locked struct {
		OwnerID     string
		SlotID      string
		SeatsCount  int
		RentalCount int
		Status      string
		StartAt     time.Time
	}
	err = tx.QueryRow(ctx, `
SELECT b.client_id::text, b.slot_id::text, b.seats_count, b.rental_count, b.status, s.start_at
FROM bookings b
JOIN slots s ON s.id = b.slot_id
WHERE b.id = $1
FOR UPDATE OF b, s`, bookingID).Scan(
		&locked.OwnerID,
		&locked.SlotID,
		&locked.SeatsCount,
		&locked.RentalCount,
		&locked.Status,
		&locked.StartAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return booking.Booking{}, booking.ErrNotFound
	}
	if err != nil {
		return booking.Booking{}, fmt.Errorf("lock booking for cancel: %w", err)
	}
	if locked.OwnerID != clientID {
		return booking.Booking{}, booking.ErrForbidden
	}
	if locked.Status != "active" {
		return booking.Booking{}, booking.ErrAlreadyCancelled
	}

	status, ok := booking.CancellationStatus(now, locked.StartAt)
	if !ok {
		return booking.Booking{}, booking.ErrSlotStarted
	}

	if _, err := tx.Exec(ctx, `
UPDATE bookings
SET status = $2, cancelled_at = $3
WHERE id = $1`, bookingID, status, now); err != nil {
		return booking.Booking{}, fmt.Errorf("update booking cancel status: %w", err)
	}
	if status == "cancelled" {
		if _, err := tx.Exec(ctx, `
UPDATE slots
SET free_seats = free_seats + $2,
    free_rental_boards = free_rental_boards + $3
WHERE id = $1`, locked.SlotID, locked.SeatsCount, locked.RentalCount); err != nil {
			return booking.Booking{}, fmt.Errorf("return slot availability: %w", err)
		}
	}

	cancelled, found, err := bookingByID(ctx, tx, bookingID)
	if err != nil {
		return booking.Booking{}, err
	}
	if !found {
		return booking.Booking{}, booking.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return booking.Booking{}, fmt.Errorf("commit cancel booking: %w", err)
	}
	return cancelled, nil
}

type idempotencyRecord struct {
	RequestHash string
	BookingID   string
}

func lockIdempotencyKey(ctx context.Context, tx pgx.Tx, clientID, key string) (idempotencyRecord, bool, error) {
	var record idempotencyRecord
	var body []byte
	err := tx.QueryRow(ctx, `
SELECT request_hash, response_body
FROM idempotency_keys
WHERE client_id = $1 AND key = $2
FOR UPDATE`, clientID, key).Scan(&record.RequestHash, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return idempotencyRecord{}, false, nil
	}
	if err != nil {
		return idempotencyRecord{}, false, fmt.Errorf("lock idempotency key: %w", err)
	}
	if len(body) > 0 {
		var payload struct {
			BookingID string `json:"booking_id"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return idempotencyRecord{}, false, fmt.Errorf("unmarshal idempotency response: %w", err)
		}
		record.BookingID = payload.BookingID
	}
	return record, true, nil
}

func insertIdempotencyKey(ctx context.Context, tx pgx.Tx, clientID, key, requestHash string, now time.Time) error {
	_, err := tx.Exec(ctx, `
INSERT INTO idempotency_keys (client_id, key, request_hash, expires_at)
VALUES ($1, $2, $3, $4)`, clientID, key, requestHash, now.Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("insert idempotency key: %w", err)
	}
	return nil
}

func lockSlot(ctx context.Context, tx pgx.Tx, slotID string) (booking.Slot, error) {
	var slot booking.Slot
	err := tx.QueryRow(ctx, `
SELECT
    s.id::text,
    r.id::text,
    r.name,
    r.type,
    r.capacity_cap,
    r.duration_min,
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
WHERE s.id = $1
FOR UPDATE OF s`, slotID).Scan(
		&slot.ID,
		&slot.RouteID,
		&slot.RouteName,
		&slot.RouteType,
		&slot.RouteCapacityCap,
		&slot.RouteDurationMin,
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
		return booking.Slot{}, booking.ErrSlotStarted
	}
	if err != nil {
		return booking.Slot{}, fmt.Errorf("lock slot: %w", err)
	}
	return slot, nil
}

type bookingQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func bookingByID(ctx context.Context, db bookingQuerier, id string) (booking.Booking, bool, error) {
	var b booking.Booking
	err := db.QueryRow(ctx, bookingSelectSQL()+`
WHERE b.id = $1`, id).Scan(bookingScanDest(&b)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return booking.Booking{}, false, nil
	}
	if err != nil {
		return booking.Booking{}, false, fmt.Errorf("query booking by id: %w", err)
	}
	b.PriceTotal = b.Slot.Price*b.SeatsCount + b.Slot.RentalPrice*b.RentalCount
	return b, true, nil
}

func bookingSelectSQL() string {
	return `
SELECT
    b.id::text,
    b.slot_id::text,
    b.client_id::text,
    b.seats_count,
    b.rental_count,
    b.status,
    b.created_at,
    b.cancelled_at,
    s.id::text,
    r.id::text,
    r.name,
    r.type,
    r.capacity_cap,
    r.duration_min,
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
FROM bookings b
JOIN slots s ON s.id = b.slot_id
JOIN routes r ON r.id = s.route_id
JOIN instructors i ON i.id = s.instructor_id
`
}

type bookingScanner interface {
	Scan(...any) error
}

func scanBooking(scanner bookingScanner) (booking.Booking, error) {
	var b booking.Booking
	if err := scanner.Scan(bookingScanDest(&b)...); err != nil {
		return booking.Booking{}, fmt.Errorf("scan booking: %w", err)
	}
	b.PriceTotal = b.Slot.Price*b.SeatsCount + b.Slot.RentalPrice*b.RentalCount
	return b, nil
}

func bookingScanDest(b *booking.Booking) []any {
	return []any{
		&b.ID,
		&b.SlotID,
		&b.ClientID,
		&b.SeatsCount,
		&b.RentalCount,
		&b.Status,
		&b.CreatedAt,
		&b.CancelledAt,
		&b.Slot.ID,
		&b.Slot.RouteID,
		&b.Slot.RouteName,
		&b.Slot.RouteType,
		&b.Slot.RouteCapacityCap,
		&b.Slot.RouteDurationMin,
		&b.Slot.InstructorID,
		&b.Slot.InstructorName,
		&b.Slot.StartAt,
		&b.Slot.TotalSeats,
		&b.Slot.FreeSeats,
		&b.Slot.FreeRentalBoards,
		&b.Slot.Price,
		&b.Slot.RentalPrice,
		&b.Slot.MeetingPoint,
		&b.Slot.MeetingPointLat,
		&b.Slot.MeetingPointLng,
		&b.Slot.Status,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
