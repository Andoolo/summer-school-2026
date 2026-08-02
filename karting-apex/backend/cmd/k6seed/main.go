package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"summer-school-2026/backend/internal/config"
	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/storage/postgres"
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

	logger.Info("k6 seed ready",
		"booking_slot_id", bookingSlotID,
		"token_prefix", "vu-token-",
		"cancel_token", "vu-token-1",
		"cancel_booking_ids", cancelBookingID,
	)
}
