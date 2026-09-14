package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
	"summer-school-2026/backend/seed"

	"github.com/jackc/pgx/v5/pgxpool"
)

// demoTestSlot создаёт отдельный заезд, чтобы не зависеть от дат в сид-данных.
func demoTestSlot(t *testing.T, ctx context.Context, db *pgxpool.Pool, startAt time.Time) string {
	t.Helper()
	var routeID, instructorID, slotID string
	if err := db.QueryRow(ctx, `INSERT INTO routes (name, type, capacity_cap, duration_min) VALUES ('Демо-трасса', 'novice', 8, 20) RETURNING id::text`).Scan(&routeID); err != nil {
		t.Fatalf("insert route: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO instructors (name) VALUES ('Демо-маршал') RETURNING id::text`).Scan(&instructorID); err != nil {
		t.Fatalf("insert instructor: %v", err)
	}
	if err := db.QueryRow(ctx, `
INSERT INTO slots (route_id, instructor_id, start_at, total_seats, free_seats, rental_boards_total, free_rental_boards, price, rental_price, meeting_point, meeting_point_lat, meeting_point_lng)
VALUES ($1, $2, $3, 8, 8, 12, 12, 2500, 500, 'Паддок', 55.7, 37.6)
RETURNING id::text`, routeID, instructorID, startAt).Scan(&slotID); err != nil {
		t.Fatalf("insert slot: %v", err)
	}
	return slotID
}

func slotAvailability(t *testing.T, ctx context.Context, db *pgxpool.Pool, slotID string) (int, int) {
	t.Helper()
	var seats, boards int
	if err := db.QueryRow(ctx, `SELECT free_seats, free_rental_boards FROM slots WHERE id = $1`, slotID).Scan(&seats, &boards); err != nil {
		t.Fatalf("read slot: %v", err)
	}
	return seats, boards
}

func holdSeats(t *testing.T, ctx context.Context, db *pgxpool.Pool, slotID, clientID, status string, seats, boards int) {
	t.Helper()
	cancelledAt := "NULL"
	if status != "active" {
		cancelledAt = "now()"
	}
	if _, err := db.Exec(ctx, `
INSERT INTO bookings (slot_id, client_id, seats_count, rental_count, status, cancelled_at)
VALUES ($1, $2, $3, $4, $5, `+cancelledAt+`)`, slotID, clientID, seats, boards, status); err != nil {
		t.Fatalf("insert booking: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE slots SET free_seats = free_seats - $2, free_rental_boards = free_rental_boards - $3 WHERE id = $1`, slotID, seats, boards); err != nil {
		t.Fatalf("hold seats: %v", err)
	}
}

func TestDemoRepositoryCreateCountAndCleanup(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	repo := postgres.NewDemoRepository(db)
	now := time.Now().UTC()
	slotID := demoTestSlot(t, ctx, db, now.Add(48*time.Hour))

	expired, err := repo.CreateDemoClient(ctx, "+70000000001", "Гость", now.Add(-25*time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("CreateDemoClient(expired) error = %v", err)
	}
	alive, err := repo.CreateDemoClient(ctx, "+70000000002", "Гость", now, now.Add(23*time.Hour))
	if err != nil {
		t.Fatalf("CreateDemoClient(alive) error = %v", err)
	}
	if alive.DemoExpiresAt == nil {
		t.Fatal("created demo client must carry its expiry")
	}
	if _, err := repo.CreateDemoClient(ctx, "+70000000002", "Гость", now, now.Add(time.Hour)); !errors.Is(err, auth.ErrDemoPhoneTaken) {
		t.Fatalf("duplicate phone error = %v, want %v", err, auth.ErrDemoPhoneTaken)
	}

	count, err := repo.CountActiveDemoClients(ctx, now)
	if err != nil || count != 1 {
		t.Fatalf("CountActiveDemoClients() = %d, %v; want 1", count, err)
	}

	// Истёкший гость держит 2 места активной бронью и 1 место поздней отменой.
	holdSeats(t, ctx, db, slotID, expired.ID, "active", 2, 1)
	holdSeats(t, ctx, db, slotID, expired.ID, "late_cancel", 1, 0)
	// Живой гость — 1 место, его уборка трогать не должна.
	holdSeats(t, ctx, db, slotID, alive.ID, "active", 1, 1)
	if seats, boards := slotAvailability(t, ctx, db, slotID); seats != 4 || boards != 10 {
		t.Fatalf("before cleanup seats/boards = %d/%d, want 4/10", seats, boards)
	}

	removed, err := postgres.CleanupExpiredDemoClients(ctx, db, now, 100)
	if err != nil || removed != 1 {
		t.Fatalf("CleanupExpiredDemoClients() = %d, %v; want 1", removed, err)
	}
	if seats, boards := slotAvailability(t, ctx, db, slotID); seats != 7 || boards != 11 {
		t.Fatalf("after cleanup seats/boards = %d/%d, want 7/11 (expired guest's seats returned)", seats, boards)
	}
	var left int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM clients WHERE id = $1`, expired.ID).Scan(&left); err != nil || left != 0 {
		t.Fatalf("expired guest rows left = %d, %v; want 0", left, err)
	}

	// Повторная уборка ничего не делает и мест не добавляет.
	if removed, err := postgres.CleanupExpiredDemoClients(ctx, db, now, 100); err != nil || removed != 0 {
		t.Fatalf("second cleanup = %d, %v; want 0", removed, err)
	}
	if seats, _ := slotAvailability(t, ctx, db, slotID); seats != 7 {
		t.Fatalf("seats after second cleanup = %d, want 7", seats)
	}
}

func TestDeleteClientAccountReturnsSeats(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	now := time.Now().UTC()
	slotID := demoTestSlot(t, ctx, db, now.Add(48*time.Hour))
	var clientID string
	if err := db.QueryRow(ctx, `INSERT INTO clients (phone, name) VALUES ('+79995550301', 'Иван') RETURNING id::text`).Scan(&clientID); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	holdSeats(t, ctx, db, slotID, clientID, "active", 3, 2)

	repo := postgres.NewProfileRepository(db)
	if err := repo.DeleteClientAccount(ctx, clientID, now); err != nil {
		t.Fatalf("DeleteClientAccount() error = %v", err)
	}
	if seats, boards := slotAvailability(t, ctx, db, slotID); seats != 8 || boards != 12 {
		t.Fatalf("after delete seats/boards = %d/%d, want 8/12: deleting an account must not leak seats", seats, boards)
	}
	// Повторный вызов (например, гонка двух запросов) не должен добавлять места сверх нормы.
	if err := repo.DeleteClientAccount(ctx, clientID, now); err != nil {
		t.Fatalf("second DeleteClientAccount() error = %v", err)
	}
	if seats, _ := slotAvailability(t, ctx, db, slotID); seats != 8 {
		t.Fatalf("seats after second delete = %d, want 8", seats)
	}
}

// Сид применяется при каждом старте сервиса. Он не должен затирать места, занятые
// настоящими бронями: иначе после перезапуска места «появлялись» заново, а уборка
// гостей и отмены могли выйти за вместимость заезда.
func TestSeedKeepsRealBookingsInSeatCount(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	const upcoming = "99999999-9999-9999-9999-999999999999" // в сиде: 8 мест, свободно 5, досок 12
	if err := postgres.SeedSQL(databaseURL, seed.KartingLapResults); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if seats, boards := slotAvailability(t, ctx, db, upcoming); seats != 5 || boards != 12 {
		t.Fatalf("after first seed seats/boards = %d/%d, want 5/12", seats, boards)
	}

	var clientID string
	if err := db.QueryRow(ctx, `INSERT INTO clients (phone, name) VALUES ('+79995550401', 'Анна') RETURNING id::text`).Scan(&clientID); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	holdSeats(t, ctx, db, upcoming, clientID, "active", 2, 1)
	holdSeats(t, ctx, db, upcoming, clientID, "cancelled", 1, 0) // отменённая вовремя — места не держит
	if _, err := db.Exec(ctx, `UPDATE slots SET free_seats = free_seats + 1 WHERE id = $1`, upcoming); err != nil {
		t.Fatalf("return cancelled seat: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := postgres.SeedSQL(databaseURL, seed.KartingLapResults); err != nil {
			t.Fatalf("reseed %d: %v", i+1, err)
		}
		if seats, boards := slotAvailability(t, ctx, db, upcoming); seats != 3 || boards != 11 {
			t.Fatalf("after reseed %d seats/boards = %d/%d, want 3/11 (real booking must stay counted)", i+1, seats, boards)
		}
	}
}

// Если счёт мест уже рассогласован (так было со старым сидом), уборка не должна падать
// на CHECK-ограничении: иначе она ломалась бы на каждом запуске и гости копились бы.
func TestDemoCleanupSurvivesInconsistentSeatCount(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	now := time.Now().UTC()
	slotID := demoTestSlot(t, ctx, db, now.Add(48*time.Hour))
	guest, err := postgres.NewDemoRepository(db).CreateDemoClient(ctx, "+70000000009", "Гость", now.Add(-25*time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("CreateDemoClient() error = %v", err)
	}
	holdSeats(t, ctx, db, slotID, guest.ID, "active", 3, 2)
	// Рассогласование: места «вернули» без отмены брони.
	if _, err := db.Exec(ctx, `UPDATE slots SET free_seats = total_seats, free_rental_boards = rental_boards_total WHERE id = $1`, slotID); err != nil {
		t.Fatalf("break seat count: %v", err)
	}

	removed, err := postgres.CleanupExpiredDemoClients(ctx, db, now, 100)
	if err != nil || removed != 1 {
		t.Fatalf("CleanupExpiredDemoClients() = %d, %v; want 1 without constraint error", removed, err)
	}
	if seats, boards := slotAvailability(t, ctx, db, slotID); seats != 8 || boards != 12 {
		t.Fatalf("seats/boards = %d/%d, want capped at 8/12", seats, boards)
	}
}
