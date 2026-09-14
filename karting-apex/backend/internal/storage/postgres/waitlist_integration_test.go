package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"summer-school-2026/backend/internal/service/notify"
	"summer-school-2026/backend/internal/service/waitlist"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
)

func prepareWaitlistDB(t *testing.T) (*pgxpool.Pool, *postgres.WaitlistRepository, time.Time) {
	t.Helper()
	databaseURL := testutil.PrepareDatabase(t)
	db, err := postgres.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)
	now := time.Now().UTC().Truncate(time.Second)
	// Заезд через 5 часов, мест нет.
	exec(t, db, `UPDATE slots SET start_at = $1, free_seats = 0 WHERE id = $2`, now.Add(5*time.Hour), laterSlot)
	return db, postgres.NewWaitlistRepository(db), now
}

func TestWaitlistJoinRules(t *testing.T) {
	db, repo, now := prepareWaitlistDB(t)
	ctx := context.Background()
	anna := insertNotifyClient(t, db, "+79990011001", 101, true)

	entry, created, err := repo.Join(ctx, anna, laterSlot, 2, now)
	if err != nil || !created || entry.Status != "waiting" || entry.Position != 1 || entry.SeatsCount != 2 {
		t.Fatalf("Join() = %+v, created=%v, err=%v", entry, created, err)
	}
	// Повтор — та же запись, без дубля.
	again, created, err := repo.Join(ctx, anna, laterSlot, 1, now)
	if err != nil || created || again.ID != entry.ID {
		t.Fatalf("repeat Join() = %+v, created=%v, err=%v", again, created, err)
	}

	boris := insertNotifyClient(t, db, "+79990011002", 102, true)
	second, _, err := repo.Join(ctx, boris, laterSlot, 1, now.Add(time.Second))
	if err != nil || second.Position != 2 {
		t.Fatalf("second Join() = %+v, %v; want position 2", second, err)
	}

	// Места есть — в очередь не ставим, надо бронировать.
	exec(t, db, `UPDATE slots SET free_seats = 1 WHERE id = $1`, laterSlot)
	vera := insertNotifyClient(t, db, "+79990011003", 103, true)
	if _, _, err := repo.Join(ctx, vera, laterSlot, 1, now); !errors.Is(err, waitlist.ErrSeatsAvailable) {
		t.Fatalf("Join() with free seat error = %v, want ErrSeatsAvailable", err)
	}
	// А если мест меньше, чем нужно, — можно.
	if _, created, err := repo.Join(ctx, vera, laterSlot, 2, now.Add(2*time.Second)); err != nil || !created {
		t.Fatalf("Join(2 seats, 1 free) = %v, %v", created, err)
	}
	exec(t, db, `UPDATE slots SET free_seats = 0 WHERE id = $1`, laterSlot)

	// Уже записан — очередь не нужна.
	gleb := insertNotifyClient(t, db, "+79990011004", 104, true)
	insertNotifyBooking(t, db, gleb, laterSlot, "active", now.Add(-time.Hour), nil)
	if _, _, err := repo.Join(ctx, gleb, laterSlot, 1, now); !errors.Is(err, waitlist.ErrAlreadyBooked) {
		t.Fatalf("Join() for booked client error = %v, want ErrAlreadyBooked", err)
	}

	if _, _, err := repo.Join(ctx, anna, "00000000-0000-0000-0000-000000000000", 1, now); !errors.Is(err, waitlist.ErrSlotNotFound) {
		t.Fatalf("Join(unknown slot) error = %v", err)
	}
	exec(t, db, `UPDATE slots SET start_at = $1 WHERE id = $2`, now.Add(-time.Minute), soonSlot)
	if _, _, err := repo.Join(ctx, anna, soonSlot, 1, now); !errors.Is(err, waitlist.ErrSlotStarted) {
		t.Fatalf("Join(started slot) error = %v", err)
	}

	// Выход из очереди сдвигает остальных.
	if err := repo.Leave(ctx, anna, laterSlot, now); err != nil {
		t.Fatalf("Leave() error = %v", err)
	}
	if _, found, _ := repo.ActiveEntry(ctx, anna, laterSlot); found {
		t.Fatal("left client must not be in queue")
	}
	if moved, _, _ := repo.ActiveEntry(ctx, boris, laterSlot); moved.Position != 1 {
		t.Fatalf("boris position = %d, want 1 after anna left", moved.Position)
	}
	if err := repo.Leave(ctx, anna, laterSlot, now); err != nil {
		t.Fatalf("repeat Leave() error = %v", err)
	}
}

func TestWaitlistLimitPerClient(t *testing.T) {
	db, repo, now := prepareWaitlistDB(t)
	ctx := context.Background()
	anna := insertNotifyClient(t, db, "+79990012001", 201, true)
	route, instructor := slotRefs(t, db, laterSlot)
	for i := 0; i < waitlist.MaxActiveEntries; i++ {
		slot := insertFullSlot(t, db, route, instructor, now.Add(time.Duration(i+6)*time.Hour))
		if _, _, err := repo.Join(ctx, anna, slot, 1, now); err != nil {
			t.Fatalf("Join #%d error = %v", i+1, err)
		}
	}
	if _, _, err := repo.Join(ctx, anna, laterSlot, 1, now); !errors.Is(err, waitlist.ErrTooManyEntries) {
		t.Fatalf("Join over limit error = %v, want ErrTooManyEntries", err)
	}
}

func TestWaitlistOffersGoInQueueOrderWithPause(t *testing.T) {
	db, repo, now := prepareWaitlistDB(t)
	ctx := context.Background()
	ttl := waitlist.OfferTTL

	anna := insertNotifyClient(t, db, "+79990013001", 301, true)
	boris := insertNotifyClient(t, db, "+79990013002", 302, true)
	vera := insertNotifyClient(t, db, "+79990013003", 303, true)
	muted := insertNotifyClient(t, db, "+79990013004", 304, false)
	for i, client := range []string{anna, boris, vera} {
		if _, _, err := repo.Join(ctx, client, laterSlot, 1, now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("join: %v", err)
		}
	}
	// Отключивший уведомления стоит первым, но предложений не получает.
	exec(t, db, `INSERT INTO waitlist_entries (slot_id, client_id, seats_count, created_at) VALUES ($1, $2, 1, $3)`, laterSlot, muted, now.Add(-time.Hour))

	// Мест нет — предлагать нечего.
	if offers := claimOffers(t, repo, now, ttl); len(offers) != 0 {
		t.Fatalf("offers without free seats = %+v", offers)
	}

	// Освободилось одно место — предложение первому (по времени записи) с Telegram.
	exec(t, db, `UPDATE slots SET free_seats = 1 WHERE id = $1`, laterSlot)
	offers := claimOffers(t, repo, now, ttl)
	if len(offers) != 1 || offers[0].ChatID != 301 || offers[0].SlotID != laterSlot || offers[0].FreeSeats != 1 || offers[0].RouteName == "" || !offers[0].ExpiresAt.Equal(now.Add(ttl)) {
		t.Fatalf("first offers = %+v, want anna", offers)
	}
	// Пока предложение действует, следующему то же место не предлагаем.
	if again := claimOffers(t, repo, now.Add(5*time.Minute), ttl); len(again) != 0 {
		t.Fatalf("offers during pause = %+v, want none", again)
	}
	// Анна не записалась за 15 минут — место предлагается Борису, Анна выбывает.
	later := now.Add(ttl + time.Minute)
	offers = claimOffers(t, repo, later, ttl)
	if len(offers) != 1 || offers[0].ChatID != 302 {
		t.Fatalf("offers after pause = %+v, want boris", offers)
	}
	if _, found, _ := repo.ActiveEntry(ctx, anna, laterSlot); found {
		t.Fatal("anna's expired offer must leave the queue")
	}

	// Борис записался — его запись закрывается, Вере ничего: место занято.
	insertNotifyBooking(t, db, boris, laterSlot, "active", later, nil)
	exec(t, db, `UPDATE slots SET free_seats = 0 WHERE id = $1`, laterSlot)
	if offers := claimOffers(t, repo, later.Add(time.Minute), ttl); len(offers) != 0 {
		t.Fatalf("offers after booking = %+v", offers)
	}
	if status := entryStatus(t, db, boris); status != "booked" {
		t.Fatalf("boris entry status = %q, want booked", status)
	}

	// Сбой отправки: запись возвращается в очередь на прежнее место.
	exec(t, db, `UPDATE slots SET free_seats = 1 WHERE id = $1`, laterSlot)
	offers = claimOffers(t, repo, later.Add(2*time.Minute), ttl)
	if len(offers) != 1 || offers[0].ChatID != 303 {
		t.Fatalf("offers for vera = %+v", offers)
	}
	if err := repo.ReleaseOffer(ctx, offers[0].EntryID); err != nil {
		t.Fatalf("ReleaseOffer() error = %v", err)
	}
	if entry, _, _ := repo.ActiveEntry(ctx, vera, laterSlot); entry.Status != "waiting" || entry.Position != 2 {
		t.Fatalf("released vera entry = %+v, want waiting behind muted client", entry)
	}
	// Недоставляемое предложение закрывается.
	offers = claimOffers(t, repo, later.Add(3*time.Minute), ttl)
	if len(offers) != 1 {
		t.Fatalf("offers retry = %+v", offers)
	}
	if err := repo.ExpireOffer(ctx, offers[0].EntryID, later.Add(3*time.Minute)); err != nil {
		t.Fatalf("ExpireOffer() error = %v", err)
	}
	if status := entryStatus(t, db, vera); status != "expired" {
		t.Fatalf("vera status = %q, want expired", status)
	}
}

func TestWaitlistOfferSkipsThoseWhoNeedMoreSeats(t *testing.T) {
	db, repo, now := prepareWaitlistDB(t)
	ctx := context.Background()
	big := insertNotifyClient(t, db, "+79990014001", 401, true)
	small := insertNotifyClient(t, db, "+79990014002", 402, true)
	if _, _, err := repo.Join(ctx, big, laterSlot, 3, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.Join(ctx, small, laterSlot, 1, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	exec(t, db, `UPDATE slots SET free_seats = 1 WHERE id = $1`, laterSlot)
	offers := claimOffers(t, repo, now, waitlist.OfferTTL)
	if len(offers) != 1 || offers[0].ChatID != 402 {
		t.Fatalf("offers = %+v, want the one who needs a single seat", offers)
	}
	if entry, _, _ := repo.ActiveEntry(ctx, big, laterSlot); entry.Status != "waiting" || entry.Position != 1 {
		t.Fatalf("big group entry = %+v, want still first in queue", entry)
	}
}

func TestWaitlistClosesWhenRaceStartsOrAccountDeleted(t *testing.T) {
	db, repo, now := prepareWaitlistDB(t)
	ctx := context.Background()
	anna := insertNotifyClient(t, db, "+79990015001", 501, true)
	boris := insertNotifyClient(t, db, "+79990015002", 502, true)
	if _, _, err := repo.Join(ctx, anna, laterSlot, 1, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.Join(ctx, boris, laterSlot, 1, now); err != nil {
		t.Fatal(err)
	}

	if err := postgres.NewProfileRepository(db).DeleteClientAccount(ctx, boris, now); err != nil {
		t.Fatalf("DeleteClientAccount() error = %v", err)
	}
	if status := entryStatus(t, db, boris); status != "left" {
		t.Fatalf("deleted client entry = %q, want left", status)
	}

	exec(t, db, `UPDATE slots SET start_at = $1, free_seats = 2 WHERE id = $2`, now.Add(-time.Minute), laterSlot)
	if offers := claimOffers(t, repo, now, waitlist.OfferTTL); len(offers) != 0 {
		t.Fatalf("offers for started race = %+v", offers)
	}
	if status := entryStatus(t, db, anna); status != "expired" {
		t.Fatalf("entry of started race = %q, want expired", status)
	}
}

func claimOffers(t *testing.T, repo *postgres.WaitlistRepository, now time.Time, ttl time.Duration) []notify.Offer {
	t.Helper()
	offers, err := repo.ClaimOffers(context.Background(), now, ttl)
	if err != nil {
		t.Fatalf("ClaimOffers() error = %v", err)
	}
	return offers
}

func entryStatus(t *testing.T, db *pgxpool.Pool, clientID string) string {
	t.Helper()
	var status string
	if err := db.QueryRow(context.Background(), `
SELECT status FROM waitlist_entries WHERE client_id = $1 ORDER BY created_at DESC LIMIT 1`, clientID).Scan(&status); err != nil {
		t.Fatalf("query entry status: %v", err)
	}
	return status
}

func slotRefs(t *testing.T, db *pgxpool.Pool, slotID string) (string, string) {
	t.Helper()
	var route, instructor string
	if err := db.QueryRow(context.Background(), `SELECT route_id::text, instructor_id::text FROM slots WHERE id = $1`, slotID).Scan(&route, &instructor); err != nil {
		t.Fatalf("query slot refs: %v", err)
	}
	return route, instructor
}

func insertFullSlot(t *testing.T, db *pgxpool.Pool, routeID, instructorID string, startAt time.Time) string {
	t.Helper()
	var id string
	if err := db.QueryRow(context.Background(), `
INSERT INTO slots (route_id, instructor_id, start_at, total_seats, free_seats, rental_boards_total, free_rental_boards,
                   price, rental_price, meeting_point, meeting_point_lat, meeting_point_lng, status)
SELECT route_id, instructor_id, $3, total_seats, 0, rental_boards_total, free_rental_boards,
       price, rental_price, meeting_point, meeting_point_lat, meeting_point_lng, status
FROM slots WHERE route_id = $1 AND instructor_id = $2 LIMIT 1
RETURNING id::text`, routeID, instructorID, startAt).Scan(&id); err != nil {
		t.Fatalf("insert slot: %v", err)
	}
	return id
}

// Человек выходит из очереди ровно в момент раздачи мест: выборка кандидатов уже видела
// его запись 'waiting', но отмечать её предложенной нельзя.
func TestWaitlistOfferSkipsEntryLeftDuringClaim(t *testing.T) {
	db, repo, now := prepareWaitlistDB(t)
	ctx := context.Background()
	anna := insertNotifyClient(t, db, "+79990019001", 601, true)
	if _, _, err := repo.Join(ctx, anna, laterSlot, 1, now); err != nil {
		t.Fatal(err)
	}
	exec(t, db, `UPDATE slots SET free_seats = 1 WHERE id = $1`, laterSlot)

	// Выход из очереди в открытой транзакции: строка заблокирована, изменение ещё не видно.
	leaving, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer leaving.Rollback(ctx)
	if _, err := leaving.Exec(ctx, `UPDATE waitlist_entries SET status = 'left', closed_at = now() WHERE client_id = $1`, anna); err != nil {
		t.Fatal(err)
	}

	type result struct {
		offers []notify.Offer
		err    error
	}
	done := make(chan result, 1)
	go func() {
		offers, err := repo.ClaimOffers(ctx, now, waitlist.OfferTTL)
		done <- result{offers, err}
	}()

	// Ждём, пока раздача упрётся в блокировку строки, и только тогда выходим из очереди.
	deadline := time.Now().Add(10 * time.Second)
	blocked := `SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock' AND query LIKE '%SET status = ''notified''%'`
	for count(t, db, blocked) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("ClaimOffers did not block on the row lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := leaving.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	got := <-done
	if got.err != nil {
		t.Fatalf("ClaimOffers() error = %v", got.err)
	}
	if len(got.offers) != 0 {
		t.Fatalf("offers = %+v, want none: the person left the queue", got.offers)
	}
	if status := entryStatus(t, db, anna); status != "left" {
		t.Fatalf("entry status = %q, want left", status)
	}
}
