package postgres_test

import (
	"context"
	"testing"
	"time"

	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestBookingConstraintsRejectInvalidCounts(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	if _, err := db.Exec(ctx, `INSERT INTO clients (id, phone) VALUES ($1, $2)`, clientID, "+79990006001"); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO auth_sessions (client_id, token_hash, expires_at) VALUES ($1, $2, $3)`, clientID, auth.HashToken("token"), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	_, err = db.Exec(ctx, `
INSERT INTO bookings (slot_id, client_id, seats_count, rental_count, status)
VALUES ('55555555-5555-5555-5555-555555555555', $1, 4, 0, 'active')`, clientID)
	if err == nil {
		t.Fatal("insert booking with invalid seats_count succeeded, want constraint error")
	}

	_, err = db.Exec(ctx, `
INSERT INTO bookings (slot_id, client_id, seats_count, rental_count, status)
VALUES ('55555555-5555-5555-5555-555555555555', $1, 1, 2, 'active')`, clientID)
	if err == nil {
		t.Fatal("insert booking with rental_count > seats_count succeeded, want constraint error")
	}
}
