package postgres_test

import (
	"context"
	"testing"
	"time"

	"summer-school-2026/backend/internal/ops"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestOpsRepository(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)
	repo := postgres.NewOpsRepository(db)
	now := time.Now().UTC().Truncate(time.Second)
	hour := now.Truncate(time.Hour)

	// Счётчики складываются, а не перезаписываются.
	for i := 0; i < 2; i++ {
		if err := repo.AddCounters(ctx, hour, map[string]int64{ops.TelegramSent: 3, ops.HTTPServerErrors: 1}); err != nil {
			t.Fatalf("AddCounters() error = %v", err)
		}
	}
	// Позавчерашние в сводку за сутки не попадают, а старше 30 дней удаляются уборкой.
	_ = repo.AddCounters(ctx, hour.Add(-48*time.Hour), map[string]int64{ops.TelegramSent: 100})
	_ = repo.AddCounters(ctx, hour.Add(-31*24*time.Hour), map[string]int64{ops.TelegramSent: 1})
	if removed, err := postgres.DeleteOldCounters(ctx, db, now); err != nil || removed != 1 {
		t.Fatalf("DeleteOldCounters() = %d, %v; want 1", removed, err)
	}

	if value, err := repo.State(ctx, "announced_version"); err != nil || value != "" {
		t.Fatalf("State(empty) = %q, %v", value, err)
	}
	for _, version := range []string{"v1", "v2"} {
		if err := repo.SetState(ctx, "announced_version", version); err != nil {
			t.Fatalf("SetState(%s) error = %v", version, err)
		}
	}
	if value, err := repo.State(ctx, "announced_version"); err != nil || value != "v2" {
		t.Fatalf("State() = %q, %v; want v2", value, err)
	}

	// Данные для сводки.
	exec(t, db, `UPDATE slots SET start_at = $1, total_seats = 8, free_seats = 0 WHERE id = $2`, now.Add(5*time.Hour), laterSlot)
	exec(t, db, `UPDATE slots SET start_at = $1, total_seats = 10, free_seats = 6 WHERE id = $2`, now.Add(26*time.Hour), soonSlot)
	anna := insertNotifyClient(t, db, "+79990018001", 9101, true)
	boris := insertNotifyClient(t, db, "+79990018002", 0, true)
	exec(t, db, `INSERT INTO clients (phone, demo_expires_at) VALUES ('+70001112233', $1)`, now.Add(time.Hour))
	insertNotifyBooking(t, db, anna, laterSlot, "active", now.Add(-time.Hour), nil)
	cancelledAt := now.Add(-30 * time.Minute)
	insertNotifyBooking(t, db, boris, laterSlot, "late_cancel", now.Add(-3*24*time.Hour), &cancelledAt)
	exec(t, db, `INSERT INTO waitlist_entries (slot_id, client_id, seats_count, status) VALUES ($1, $2, 1, 'waiting')`, laterSlot, boris)
	exec(t, db, `INSERT INTO waitlist_entries (slot_id, client_id, seats_count, status, notified_at, closed_at) VALUES ($1, $2, 1, 'booked', $3, $3)`, soonSlot, anna, now.Add(-2*time.Hour))

	snapshot, err := repo.Snapshot(ctx, now)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	checks := []struct {
		name      string
		got, want int
	}{
		{"created 24h", snapshot.BookingsCreated24h, 1},
		{"created 7d", snapshot.BookingsCreated7d, 2},
		{"cancelled 24h", snapshot.Cancelled24h, 1},
		{"late 7d", snapshot.LateCancelled7d, 1},
		{"races", snapshot.UpcomingRaces, 2},
		{"seats", snapshot.UpcomingSeats, 18},
		{"taken", snapshot.UpcomingTaken, 12},
		{"full", snapshot.UpcomingFull, 1},
		{"waiting", snapshot.WaitingNow, 1},
		{"offers", snapshot.Offers24h, 1},
		{"booked from waitlist", snapshot.BookedFromWaitlist24h, 1},
		{"clients without guests", snapshot.Clients, 2},
		{"telegram clients", snapshot.TelegramClients, 1},
		{"guests", snapshot.ActiveGuests, 1},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if snapshot.Counters24h[ops.TelegramSent] != 6 || snapshot.Counters24h[ops.HTTPServerErrors] != 2 {
		t.Errorf("counters 24h = %v, want tg_sent 6 and http_5xx 2", snapshot.Counters24h)
	}
}
