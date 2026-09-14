package postgres

import (
	"context"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/service/notify"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// notificationQuery — что считать «пора отметить» и что из отмеченного отправлять.
//
// Отмечаются все брони без отметки, в том числе устаревшие и брони клиентов без
// Telegram: иначе они навсегда остались бы в частичных индексах и попадали бы в каждую
// выборку. Отправляются только актуальные.
type notificationQuery struct {
	column string
	due    string
	order  string
	send   string
	args   func(now time.Time, limit int) []any
}

func notificationQueryFor(kind notify.Kind) (notificationQuery, error) {
	switch kind {
	case notify.KindConfirm:
		// $1 — сейчас, $2 — лимит, $3 — с какого момента подтверждение ещё актуально.
		return notificationQuery{
			column: "confirm_notified_at",
			due:    "b.confirm_notified_at IS NULL",
			order:  "b.created_at",
			send:   "claimed.created_at > $3 AND claimed.status = 'active' AND s.status <> 'cancelled' AND s.start_at > $1",
			args: func(now time.Time, limit int) []any {
				return []any{now, limit, now.Add(-notify.FreshWindow)}
			},
		}, nil
	case notify.KindCancel:
		return notificationQuery{
			column: "cancel_notified_at",
			due:    "b.cancel_notified_at IS NULL AND b.status <> 'active'",
			order:  "b.cancelled_at",
			send:   "claimed.cancelled_at > $3",
			args: func(now time.Time, limit int) []any {
				return []any{now, limit, now.Add(-notify.FreshWindow)}
			},
		}, nil
	case notify.KindReminder:
		// $3 — до какого старта пора напоминать, $4 — минимальный запас (секунды) между
		// бронированием и стартом. Прошедшие заезды тоже отмечаются, но не отправляются.
		return notificationQuery{
			column: "reminder_notified_at",
			due:    "b.reminder_notified_at IS NULL AND b.status = 'active' AND s.start_at <= $3",
			order:  "s.start_at",
			send: "s.start_at > $1 AND s.status <> 'cancelled' " +
				"AND claimed.created_at <= s.start_at - $4 * interval '1 second'",
			args: func(now time.Time, limit int) []any {
				return []any{now, limit, now.Add(notify.ReminderLead), int64(notify.ReminderMinAdvance / time.Second)}
			},
		}, nil
	default:
		return notificationQuery{}, fmt.Errorf("unknown notification kind %q", kind)
	}
}

// ClaimDue отмечает и возвращает уведомления одним запросом. SKIP LOCKED: параллельный
// рассыльщик (или второй экземпляр сервиса) пропускает уже захваченные брони, а не ждёт.
func (r *NotificationRepository) ClaimDue(ctx context.Context, kind notify.Kind, now time.Time, limit int) ([]notify.Notice, error) {
	q, err := notificationQueryFor(kind)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
WITH due AS (
    SELECT b.id
    FROM bookings b
    JOIN slots s ON s.id = b.slot_id
    WHERE `+q.due+`
    ORDER BY `+q.order+`
    LIMIT $2
    FOR UPDATE OF b SKIP LOCKED
), claimed AS (
    UPDATE bookings b
    SET `+q.column+` = $1
    FROM due
    WHERE b.id = due.id
    RETURNING b.id, b.slot_id, b.client_id, b.status, b.seats_count, b.rental_count, b.created_at, b.cancelled_at
)
SELECT
    claimed.id::text,
    c.telegram_chat_id,
    claimed.status,
    claimed.seats_count,
    claimed.rental_count,
    s.price * claimed.seats_count + s.rental_price * claimed.rental_count,
    claimed.created_at,
    r.name,
    i.name,
    s.start_at,
    s.meeting_point
FROM claimed
JOIN clients c ON c.id = claimed.client_id
JOIN slots s ON s.id = claimed.slot_id
JOIN routes r ON r.id = s.route_id
JOIN instructors i ON i.id = s.instructor_id
WHERE c.telegram_chat_id IS NOT NULL
  AND c.telegram_notifications
  AND c.deleted_at IS NULL
  AND `+q.send, q.args(now, limit)...)
	if err != nil {
		return nil, fmt.Errorf("claim %s notifications: %w", kind, err)
	}
	defer rows.Close()

	notices := make([]notify.Notice, 0)
	for rows.Next() {
		n := notify.Notice{Kind: kind}
		if err := rows.Scan(&n.BookingID, &n.ChatID, &n.Status, &n.SeatsCount, &n.RentalCount, &n.PriceTotal,
			&n.CreatedAt, &n.RouteName, &n.InstructorName, &n.StartAt, &n.MeetingPoint); err != nil {
			return nil, fmt.Errorf("scan %s notification: %w", kind, err)
		}
		notices = append(notices, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s notifications: %w", kind, err)
	}
	return notices, nil
}

func (r *NotificationRepository) Unclaim(ctx context.Context, kind notify.Kind, bookingID string) error {
	q, err := notificationQueryFor(kind)
	if err != nil {
		return err
	}
	if _, err := r.db.Exec(ctx, `UPDATE bookings SET `+q.column+` = NULL WHERE id = $1`, bookingID); err != nil {
		return fmt.Errorf("unclaim %s notification: %w", kind, err)
	}
	return nil
}

func (r *NotificationRepository) DisableChat(ctx context.Context, chatID int64) error {
	if _, err := r.db.Exec(ctx, `UPDATE clients SET telegram_chat_id = NULL WHERE telegram_chat_id = $1`, chatID); err != nil {
		return fmt.Errorf("unlink blocked telegram chat: %w", err)
	}
	return nil
}
