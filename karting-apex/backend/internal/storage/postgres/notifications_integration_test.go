package postgres_test

import (
	"context"
	"testing"
	"time"

	"summer-school-2026/backend/internal/service/notify"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	soonSlot  = "55555555-5555-5555-5555-555555555555"
	laterSlot = "66666666-6666-6666-6666-666666666666"
)

func TestNotificationRepositoryClaimsDueNotices(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	now := time.Now().UTC().Truncate(time.Second)
	exec(t, db, `UPDATE slots SET start_at = $1 WHERE id = $2`, now.Add(90*time.Minute), soonSlot)
	exec(t, db, `UPDATE slots SET start_at = $1 WHERE id = $2`, now.Add(5*time.Hour), laterSlot)

	anna := insertNotifyClient(t, db, "+79990007001", 777, true)
	boris := insertNotifyClient(t, db, "+79990007002", 0, true)   // без Telegram
	vera := insertNotifyClient(t, db, "+79990007003", 888, false) // отключила уведомления

	fresh := insertNotifyBooking(t, db, anna, laterSlot, "active", now.Add(-time.Minute), nil)
	insertNotifyBooking(t, db, boris, laterSlot, "active", now.Add(-time.Minute), nil)
	insertNotifyBooking(t, db, vera, laterSlot, "active", now.Add(-time.Minute), nil)
	// Бронь заранее: подтверждение устарело, но напоминание пора слать.
	early := insertNotifyBooking(t, db, anna, soonSlot, "active", now.Add(-24*time.Hour), nil)
	cancelledAt := now.Add(-5 * time.Minute)
	cancelled := insertNotifyBooking(t, db, anna, soonSlot, "late_cancel", now.Add(-48*time.Hour), &cancelledAt)

	repo := postgres.NewNotificationRepository(db)

	confirms := claim(t, repo, notify.KindConfirm, now)
	if len(confirms) != 1 || confirms[0].BookingID != fresh || confirms[0].ChatID != 777 {
		t.Fatalf("confirms = %+v, want only fresh booking of anna", confirms)
	}
	got := confirms[0]
	if got.RouteName == "" || got.InstructorName == "" || got.SeatsCount != 1 || got.PriceTotal <= 0 || !got.StartAt.Equal(now.Add(5*time.Hour)) {
		t.Fatalf("confirm details = %+v", got)
	}
	// Отмечены все, в том числе брони без Telegram и устаревшие: повторно не выбираются.
	if again := claim(t, repo, notify.KindConfirm, now); len(again) != 0 {
		t.Fatalf("second confirm claim = %+v, want none", again)
	}
	if left := count(t, db, `SELECT count(*) FROM bookings WHERE confirm_notified_at IS NULL`); left != 0 {
		t.Fatalf("unclaimed confirms left = %d, want 0", left)
	}

	// Неудачную отправку можно вернуть в очередь.
	if err := repo.Unclaim(ctx, notify.KindConfirm, fresh); err != nil {
		t.Fatalf("Unclaim() error = %v", err)
	}
	if retry := claim(t, repo, notify.KindConfirm, now); len(retry) != 1 || retry[0].BookingID != fresh {
		t.Fatalf("retry claim = %+v", retry)
	}

	cancels := claim(t, repo, notify.KindCancel, now)
	if len(cancels) != 1 || cancels[0].BookingID != cancelled || cancels[0].Status != "late_cancel" {
		t.Fatalf("cancels = %+v", cancels)
	}

	// Бронь, сделанная меньше чем за 3 часа до старта, напоминания не получает.
	exec(t, db, `UPDATE bookings SET created_at = $1 WHERE id = $2`, now.Add(-time.Hour), early)
	if reminders := claim(t, repo, notify.KindReminder, now); len(reminders) != 0 {
		t.Fatalf("reminders for last-minute booking = %+v, want none", reminders)
	}
	exec(t, db, `UPDATE bookings SET created_at = $1, reminder_notified_at = NULL WHERE id = $2`, now.Add(-24*time.Hour), early)
	reminders := claim(t, repo, notify.KindReminder, now)
	if len(reminders) != 1 || reminders[0].BookingID != early {
		t.Fatalf("reminders = %+v, want early booking", reminders)
	}
	// Заезд через 5 часов — ещё не пора.
	if pending := count(t, db, `SELECT count(*) FROM bookings WHERE slot_id = $1 AND reminder_notified_at IS NULL AND status = 'active'`, laterSlot); pending != 3 {
		t.Fatalf("later slot reminders pending = %d, want 3", pending)
	}

	// Заблокировавший бота чат отвязывается.
	if err := repo.DisableChat(ctx, 777); err != nil {
		t.Fatalf("DisableChat() error = %v", err)
	}
	if linked := count(t, db, `SELECT count(*) FROM clients WHERE telegram_chat_id = 777`); linked != 0 {
		t.Fatal("chat 777 must be unlinked")
	}
}

func TestReminderIsNotSentForStartedRace(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	now := time.Now().UTC().Truncate(time.Second)
	// Сервис проспал напоминание: заезд уже идёт.
	exec(t, db, `UPDATE slots SET start_at = $1 WHERE id = $2`, now.Add(-10*time.Minute), soonSlot)
	client := insertNotifyClient(t, db, "+79990007010", 999, true)
	booking := insertNotifyBooking(t, db, client, soonSlot, "active", now.Add(-24*time.Hour), nil)

	repo := postgres.NewNotificationRepository(db)
	if reminders := claim(t, repo, notify.KindReminder, now); len(reminders) != 0 {
		t.Fatalf("reminders = %+v, want none for started race", reminders)
	}
	if marked := count(t, db, `SELECT count(*) FROM bookings WHERE id = $1 AND reminder_notified_at IS NOT NULL`, booking); marked != 1 {
		t.Fatal("started race booking must still be marked to leave the partial index")
	}
}

func TestTelegramChatLinking(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	oldPhone := insertNotifyClient(t, db, "+79990008001", 0, true)
	newPhone := insertNotifyClient(t, db, "+79990008002", 0, true)
	repo := postgres.NewTelegramLoginRepository(db)

	if found, err := repo.SetNotifications(ctx, 555, false); err != nil || found {
		t.Fatalf("SetNotifications(unlinked) = %v, %v; want false", found, err)
	}
	if err := repo.LinkChat(ctx, "+79990008001", 555); err != nil {
		t.Fatalf("LinkChat() error = %v", err)
	}
	// Человек сменил номер в Telegram и вошёл заново: чат переезжает на новый аккаунт.
	if err := repo.LinkChat(ctx, "+79990008002", 555); err != nil {
		t.Fatalf("LinkChat(new phone) error = %v", err)
	}
	if chat := chatOf(t, db, oldPhone); chat != nil {
		t.Fatalf("old client chat = %v, want unlinked", *chat)
	}
	if chat := chatOf(t, db, newPhone); chat == nil || *chat != 555 {
		t.Fatalf("new client chat = %v, want 555", chat)
	}

	if found, err := repo.SetNotifications(ctx, 555, false); err != nil || !found {
		t.Fatalf("SetNotifications(false) = %v, %v", found, err)
	}
	if enabled := count(t, db, `SELECT count(*) FROM clients WHERE id = $1 AND telegram_notifications`, newPhone); enabled != 0 {
		t.Fatal("notifications must be disabled")
	}

	// Удаление аккаунта отвязывает чат.
	profiles := postgres.NewProfileRepository(db)
	if err := profiles.DeleteClientAccount(ctx, newPhone, time.Now().UTC()); err != nil {
		t.Fatalf("DeleteAccount() error = %v", err)
	}
	if chat := chatOf(t, db, newPhone); chat != nil {
		t.Fatalf("deleted client chat = %v, want unlinked", *chat)
	}
}

func claim(t *testing.T, repo *postgres.NotificationRepository, kind notify.Kind, now time.Time) []notify.Notice {
	t.Helper()
	notices, err := repo.ClaimDue(context.Background(), kind, now, 50)
	if err != nil {
		t.Fatalf("ClaimDue(%s) error = %v", kind, err)
	}
	return notices
}

func exec(t *testing.T, db *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := db.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func count(t *testing.T, db *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", sql, err)
	}
	return n
}

func chatOf(t *testing.T, db *pgxpool.Pool, clientID string) *int64 {
	t.Helper()
	var chat *int64
	if err := db.QueryRow(context.Background(), `SELECT telegram_chat_id FROM clients WHERE id = $1`, clientID).Scan(&chat); err != nil {
		t.Fatalf("query chat: %v", err)
	}
	return chat
}

func insertNotifyClient(t *testing.T, db *pgxpool.Pool, phone string, chatID int64, notifications bool) string {
	t.Helper()
	var chat *int64
	if chatID != 0 {
		chat = &chatID
	}
	var id string
	if err := db.QueryRow(context.Background(), `
INSERT INTO clients (phone, telegram_chat_id, telegram_notifications) VALUES ($1, $2, $3)
RETURNING id::text`, phone, chat, notifications).Scan(&id); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	return id
}

func insertNotifyBooking(t *testing.T, db *pgxpool.Pool, clientID, slotID, status string, createdAt time.Time, cancelledAt *time.Time) string {
	t.Helper()
	var id string
	if err := db.QueryRow(context.Background(), `
INSERT INTO bookings (slot_id, client_id, seats_count, rental_count, status, created_at, cancelled_at)
VALUES ($1, $2, 1, 0, $3, $4, $5)
RETURNING id::text`, slotID, clientID, status, createdAt, cancelledAt).Scan(&id); err != nil {
		t.Fatalf("insert booking: %v", err)
	}
	return id
}
