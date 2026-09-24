package postgres_test

import (
	"context"
	"testing"
	"time"

	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestBotActionsFindOnlyOwnBookings(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	now := time.Now().UTC().Truncate(time.Second)
	exec(t, db, `UPDATE slots SET start_at = $1, free_seats = 5 WHERE id = $2`, now.Add(5*time.Hour), laterSlot)
	owner := insertNotifyClient(t, db, "+79990017001", 8001, true)
	insertNotifyClient(t, db, "+79990017002", 8002, true)
	bookingID := insertNotifyBooking(t, db, owner, laterSlot, "active", now.Add(-time.Hour), nil)

	repo := postgres.NewBotActionsRepository(db)
	found, ok, err := repo.BookingForChat(ctx, bookingID, 8001)
	if err != nil || !ok || found.ClientID != owner || found.Status != "active" || !found.StartAt.Equal(now.Add(5*time.Hour)) {
		t.Fatalf("BookingForChat(owner) = %+v, %v, %v", found, ok, err)
	}
	// Чужой чат бронь не видит, даже зная её id.
	if _, ok, err := repo.BookingForChat(ctx, bookingID, 8002); err != nil || ok {
		t.Fatalf("BookingForChat(other chat) = %v, %v; want not found", ok, err)
	}

	// Отмена из бота — та же, что в приложении.
	cancelled, err := postgres.NewBookingRepository(db).Cancel(ctx, found.ClientID, bookingID, now)
	if err != nil || cancelled.Status != "cancelled" {
		t.Fatalf("Cancel() = %+v, %v", cancelled, err)
	}
	if seats := count(t, db, `SELECT free_seats FROM slots WHERE id = $1`, laterSlot); seats != 6 {
		t.Fatalf("free seats after cancel = %d, want 6 (seat returned)", seats)
	}
}
