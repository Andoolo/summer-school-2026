package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"summer-school-2026/backend/internal/config"
	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/storage/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	bookingSlotID   = "55555555-5555-5555-5555-555555555555"
	cancelSlotID    = "66666666-6666-6666-6666-666666666666"
	cancelBookingID = "99999999-9999-9999-9999-999999999999"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	now := time.Now().UTC()
	if _, err := db.Exec(ctx, `
UPDATE slots
SET start_at = $2,
    free_seats = total_seats,
    free_rental_boards = rental_boards_total,
    status = 'scheduled'
WHERE id = $1`, bookingSlotID, now.Add(24*time.Hour)); err != nil {
		logger.Error("reset booking slot", "error", err)
		os.Exit(1)
	}
	if _, err := db.Exec(ctx, `
UPDATE slots
SET start_at = $2,
    free_seats = total_seats - 1,
    free_rental_boards = rental_boards_total - 1,
    status = 'scheduled'
WHERE id = $1`, cancelSlotID, now.Add(24*time.Hour)); err != nil {
		logger.Error("reset cancel slot", "error", err)
		os.Exit(1)
	}

	for i := 1; i <= 300; i++ {
		clientID := fmt.Sprintf("70000000-0000-4000-8000-%012d", i)
		phone := fmt.Sprintf("+1700000%08d", i)
		token := fmt.Sprintf("vu-token-%d", i)
		if _, err := db.Exec(ctx, `
INSERT INTO clients (id, phone, name)
VALUES ($1, $2, $3)
ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name, deleted_at = NULL`, clientID, phone, fmt.Sprintf("k6-user-%d", i)); err != nil {
			logger.Error("upsert client", "i", i, "error", err)
			os.Exit(1)
		}
		if _, err := db.Exec(ctx, `
INSERT INTO auth_sessions (client_id, token_hash, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (token_hash) DO UPDATE SET expires_at = EXCLUDED.expires_at, revoked_at = NULL`, clientID, auth.HashToken(token), now.Add(2*time.Hour)); err != nil {
			logger.Error("upsert session", "i", i, "error", err)
			os.Exit(1)
		}
	}

	if _, err := db.Exec(ctx, `DELETE FROM idempotency_keys WHERE client_id::text LIKE '70000000-0000-4000-8000-%'`); err != nil {
		logger.Error("delete old idempotency keys", "error", err)
		os.Exit(1)
	}
	if _, err := db.Exec(ctx, `DELETE FROM bookings WHERE client_id::text LIKE '70000000-0000-4000-8000-%'`); err != nil {
		logger.Error("delete old k6 bookings", "error", err)
		os.Exit(1)
	}

	if _, err := db.Exec(ctx, `DELETE FROM bookings WHERE id = $1`, cancelBookingID); err != nil {
		logger.Error("delete old cancel booking", "error", err)
		os.Exit(1)
	}
	if _, err := db.Exec(ctx, `
INSERT INTO bookings (id, slot_id, client_id, seats_count, rental_count, status, created_at)
VALUES ($1, $2, $3, 1, 1, 'active', $4)`, cancelBookingID, cancelSlotID, "70000000-0000-4000-8000-000000000001", now); err != nil {
		logger.Error("insert cancel booking", "error", err)
		os.Exit(1)
	}

	if err := seedWaitlist(ctx, db, now); err != nil {
		logger.Error("seed waitlist scenario", "error", err)
		os.Exit(1)
	}

	logger.Info("k6 seed ready",
		"booking_slot_id", bookingSlotID,
		"token_prefix", "vu-token-",
		"cancel_token", "vu-token-1",
		"cancel_booking_ids", cancelBookingID,
		"waitlist_slot_ids", strings.Join(waitlistSlotIDs, ","),
	)
}

// Сценарий k6/waitlist_300_vu.js: три заполненных заезда по 8 мест. Места держат k6-клиенты
// 1–24 (по 8 на заезд), в очередь встают 101–300. У всех k6-клиентов привязан «чат» и
// включены уведомления: без этого встать в очередь нельзя. Сообщения уходят туда, куда
// указывает TELEGRAM_API_BASE стенда (имитатор Bot API), а не в настоящий Telegram.
var waitlistSlotIDs = []string{
	"7a170000-0000-4000-8000-000000000001",
	"7a170000-0000-4000-8000-000000000002",
	"7a170000-0000-4000-8000-000000000003",
}

const (
	waitlistSeats       = 8
	waitlistChatIDShift = 900000000
)

func seedWaitlist(ctx context.Context, db *pgxpool.Pool, now time.Time) error {
	if _, err := db.Exec(ctx, `
UPDATE clients SET telegram_chat_id = $1 + substring(id::text from 25)::bigint, telegram_notifications = true
WHERE id::text LIKE '70000000-0000-4000-8000-%'`, waitlistChatIDShift); err != nil {
		return fmt.Errorf("link k6 telegram chats: %w", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM waitlist_entries WHERE client_id::text LIKE '70000000-0000-4000-8000-%'`); err != nil {
		return fmt.Errorf("delete old k6 waitlist entries: %w", err)
	}
	for i, slotID := range waitlistSlotIDs {
		if _, err := db.Exec(ctx, `DELETE FROM bookings WHERE slot_id = $1`, slotID); err != nil {
			return fmt.Errorf("delete old waitlist slot bookings: %w", err)
		}
		// Копия заезда из сида: те же трасса, маршал и цены, но свой id и полная загрузка.
		if _, err := db.Exec(ctx, `
INSERT INTO slots (id, route_id, instructor_id, start_at, total_seats, free_seats, rental_boards_total,
                   free_rental_boards, price, rental_price, meeting_point, meeting_point_lat, meeting_point_lng, status)
SELECT $1, route_id, instructor_id, $2, $3, 0, rental_boards_total, rental_boards_total,
       price, rental_price, meeting_point, meeting_point_lat, meeting_point_lng, 'scheduled'
FROM slots WHERE id = $4
ON CONFLICT (id) DO UPDATE SET start_at = EXCLUDED.start_at, total_seats = EXCLUDED.total_seats,
    free_seats = 0, free_rental_boards = EXCLUDED.free_rental_boards, status = 'scheduled'`,
			slotID, now.Add(24*time.Hour+time.Duration(i)*time.Hour), waitlistSeats, bookingSlotID); err != nil {
			return fmt.Errorf("upsert waitlist slot: %w", err)
		}
		for seat := 1; seat <= waitlistSeats; seat++ {
			holder := i*waitlistSeats + seat
			if _, err := db.Exec(ctx, `
INSERT INTO bookings (slot_id, client_id, seats_count, rental_count, status, created_at)
VALUES ($1, $2, 1, 0, 'active', $3)`, slotID, fmt.Sprintf("70000000-0000-4000-8000-%012d", holder), now); err != nil {
				return fmt.Errorf("insert holder booking: %w", err)
			}
		}
	}
	return nil
}
