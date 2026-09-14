package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/service/notify"
	"summer-school-2026/backend/internal/service/waitlist"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WaitlistRepository — очередь на заполненные заезды: вход и выход для приложения
// (waitlist.Repository) и раздача предложений для рассыльщика (notify.WaitlistRepository).
type WaitlistRepository struct {
	db *pgxpool.Pool
}

func NewWaitlistRepository(db *pgxpool.Pool) *WaitlistRepository {
	return &WaitlistRepository{db: db}
}

func (r *WaitlistRepository) ClientBySessionTokenHash(ctx context.Context, tokenHash string) (waitlist.Client, bool, error) {
	var client waitlist.Client
	err := r.db.QueryRow(ctx, `
SELECT c.id::text, c.telegram_chat_id IS NOT NULL, c.telegram_notifications
FROM auth_sessions s
JOIN clients c ON c.id = s.client_id
WHERE s.token_hash = $1
  AND s.revoked_at IS NULL
  AND s.expires_at > now()
  AND c.deleted_at IS NULL`, tokenHash).Scan(&client.ID, &client.TelegramLinked, &client.NotificationsEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return waitlist.Client{}, false, nil
	}
	if err != nil {
		return waitlist.Client{}, false, fmt.Errorf("query waitlist client by session: %w", err)
	}
	return client, true, nil
}

func (r *WaitlistRepository) Join(ctx context.Context, clientID, slotID string, seats int, now time.Time) (waitlist.Entry, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return waitlist.Entry{}, false, fmt.Errorf("begin join waitlist: %w", err)
	}
	defer tx.Rollback(ctx)

	// Та же блокировка строки заезда, что при бронировании: проверка «мест нет» и
	// постановка в очередь не разъедутся с параллельной бронью или отменой.
	var slotStatus string
	var startAt time.Time
	var freeSeats int
	err = tx.QueryRow(ctx, `SELECT status, start_at, free_seats FROM slots WHERE id = $1 FOR UPDATE`, slotID).
		Scan(&slotStatus, &startAt, &freeSeats)
	if errors.Is(err, pgx.ErrNoRows) {
		return waitlist.Entry{}, false, waitlist.ErrSlotNotFound
	}
	if err != nil {
		return waitlist.Entry{}, false, fmt.Errorf("lock slot for waitlist: %w", err)
	}
	if slotStatus == "cancelled" {
		return waitlist.Entry{}, false, waitlist.ErrSlotCancelled
	}
	if !now.Before(startAt) {
		return waitlist.Entry{}, false, waitlist.ErrSlotStarted
	}

	if existing, found, err := activeEntry(ctx, tx, clientID, slotID); err != nil {
		return waitlist.Entry{}, false, err
	} else if found {
		if err := tx.Commit(ctx); err != nil {
			return waitlist.Entry{}, false, fmt.Errorf("commit existing waitlist entry: %w", err)
		}
		return existing, false, nil
	}

	var booked bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM bookings WHERE client_id = $1 AND slot_id = $2 AND status = 'active')`, clientID, slotID).Scan(&booked); err != nil {
		return waitlist.Entry{}, false, fmt.Errorf("check booking for waitlist: %w", err)
	}
	if booked {
		return waitlist.Entry{}, false, waitlist.ErrAlreadyBooked
	}
	if freeSeats >= seats {
		return waitlist.Entry{}, false, waitlist.ErrSeatsAvailable
	}

	var active int
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM waitlist_entries w
JOIN slots s ON s.id = w.slot_id
WHERE w.client_id = $1 AND w.status IN ('waiting', 'notified') AND s.start_at > $2`, clientID, now).Scan(&active); err != nil {
		return waitlist.Entry{}, false, fmt.Errorf("count waitlist entries: %w", err)
	}
	if active >= waitlist.MaxActiveEntries {
		return waitlist.Entry{}, false, waitlist.ErrTooManyEntries
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO waitlist_entries (slot_id, client_id, seats_count, created_at)
VALUES ($1, $2, $3, $4)`, slotID, clientID, seats, now); err != nil {
		return waitlist.Entry{}, false, fmt.Errorf("insert waitlist entry: %w", err)
	}
	created, found, err := activeEntry(ctx, tx, clientID, slotID)
	if err != nil {
		return waitlist.Entry{}, false, err
	}
	if !found {
		return waitlist.Entry{}, false, fmt.Errorf("created waitlist entry not found")
	}
	if err := tx.Commit(ctx); err != nil {
		return waitlist.Entry{}, false, fmt.Errorf("commit join waitlist: %w", err)
	}
	return created, true, nil
}

func (r *WaitlistRepository) Leave(ctx context.Context, clientID, slotID string, now time.Time) error {
	if _, err := r.db.Exec(ctx, `
UPDATE waitlist_entries
SET status = 'left', closed_at = $3
WHERE client_id = $1 AND slot_id = $2 AND status IN ('waiting', 'notified')`, clientID, slotID, now); err != nil {
		return fmt.Errorf("leave waitlist: %w", err)
	}
	return nil
}

func (r *WaitlistRepository) ActiveEntry(ctx context.Context, clientID, slotID string) (waitlist.Entry, bool, error) {
	return activeEntry(ctx, r.db, clientID, slotID)
}

// activeEntry — запись клиента в очереди заезда с местом в очереди. Место считается
// только среди ждущих: у получивших предложение очередь уже подошла.
func activeEntry(ctx context.Context, db bookingQuerier, clientID, slotID string) (waitlist.Entry, bool, error) {
	var entry waitlist.Entry
	err := db.QueryRow(ctx, `
SELECT
    w.id::text,
    w.slot_id::text,
    w.seats_count,
    w.status,
    w.created_at,
    w.notified_at,
    CASE WHEN w.status = 'waiting' THEN (
        SELECT count(*) + 1
        FROM waitlist_entries ahead
        WHERE ahead.slot_id = w.slot_id
          AND ahead.status = 'waiting'
          AND (ahead.created_at, ahead.id) < (w.created_at, w.id)
    ) ELSE 0 END
FROM waitlist_entries w
WHERE w.client_id = $1 AND w.slot_id = $2 AND w.status IN ('waiting', 'notified')`, clientID, slotID).
		Scan(&entry.ID, &entry.SlotID, &entry.SeatsCount, &entry.Status, &entry.CreatedAt, &entry.NotifiedAt, &entry.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return waitlist.Entry{}, false, nil
	}
	if err != nil {
		return waitlist.Entry{}, false, fmt.Errorf("query waitlist entry: %w", err)
	}
	return entry, true, nil
}

// ClaimOffers закрывает отработавшие записи и раздаёт свободные места следующим в
// очереди. Всё — в одной транзакции под блокировкой строк заездов (SKIP LOCKED: заезд,
// который прямо сейчас бронируют, обработается следующим проходом).
func (r *WaitlistRepository) ClaimOffers(ctx context.Context, now time.Time, ttl time.Duration) ([]notify.Offer, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin waitlist offers: %w", err)
	}
	defer tx.Rollback(ctx)

	// Записался на заезд — очередь ему больше не нужна.
	if _, err := tx.Exec(ctx, `
UPDATE waitlist_entries w
SET status = 'booked', closed_at = $1
FROM bookings b
WHERE b.client_id = w.client_id AND b.slot_id = w.slot_id AND b.status = 'active'
  AND w.status IN ('waiting', 'notified')`, now); err != nil {
		return nil, fmt.Errorf("close booked waitlist entries: %w", err)
	}
	// Заезд начался или отменён, либо предложение пролежало дольше срока.
	if _, err := tx.Exec(ctx, `
UPDATE waitlist_entries w
SET status = 'expired', closed_at = $1
FROM slots s
WHERE s.id = w.slot_id
  AND w.status IN ('waiting', 'notified')
  AND (s.start_at <= $1 OR s.status = 'cancelled' OR (w.status = 'notified' AND w.notified_at <= $2))`, now, now.Add(-ttl)); err != nil {
		return nil, fmt.Errorf("expire waitlist entries: %w", err)
	}

	rows, err := tx.Query(ctx, `
WITH open_slots AS (
    SELECT s.id, s.free_seats
    FROM slots s
    WHERE s.status <> 'cancelled'
      AND s.start_at > $1
      AND s.free_seats > 0
      AND EXISTS (SELECT 1 FROM waitlist_entries w WHERE w.slot_id = s.id AND w.status = 'waiting')
    FOR UPDATE OF s SKIP LOCKED
), offered AS (
    SELECT slot_id, sum(seats_count) AS seats
    FROM waitlist_entries
    WHERE status = 'notified'
    GROUP BY slot_id
)
SELECT w.id::text, w.slot_id::text, w.seats_count, os.free_seats - coalesce(o.seats, 0)
FROM waitlist_entries w
JOIN open_slots os ON os.id = w.slot_id
LEFT JOIN offered o ON o.slot_id = w.slot_id
JOIN clients c ON c.id = w.client_id
WHERE w.status = 'waiting'
  AND c.telegram_chat_id IS NOT NULL
  AND c.telegram_notifications
  AND c.deleted_at IS NULL
ORDER BY w.slot_id, w.created_at, w.id`, now)
	if err != nil {
		return nil, fmt.Errorf("query waitlist candidates: %w", err)
	}
	// Места раздаются по порядку записи. Кому не хватает мест (ждёт троих, а свободно
	// одно), тот пропускается, но остаётся первым: освободятся ещё места — получит он.
	budget := map[string]int{}
	var picked []string
	for rows.Next() {
		var entryID, slotID string
		var seats, available int
		if err := rows.Scan(&entryID, &slotID, &seats, &available); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan waitlist candidate: %w", err)
		}
		left, seen := budget[slotID]
		if !seen {
			left = available
		}
		if seats <= left {
			picked = append(picked, entryID)
			left -= seats
		}
		budget[slotID] = left
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate waitlist candidates: %w", err)
	}

	offers := make([]notify.Offer, 0, len(picked))
	if len(picked) > 0 {
		rows, err := tx.Query(ctx, `
UPDATE waitlist_entries w
SET status = 'notified', notified_at = $1
FROM clients c, slots s, routes r, instructors i
WHERE w.id::text = ANY($2)
  AND c.id = w.client_id
  AND s.id = w.slot_id
  AND r.id = s.route_id
  AND i.id = s.instructor_id
RETURNING w.id::text, c.telegram_chat_id, w.seats_count, s.free_seats, r.name, i.name, s.start_at, s.meeting_point`, now, picked)
		if err != nil {
			return nil, fmt.Errorf("mark waitlist offers: %w", err)
		}
		for rows.Next() {
			offer := notify.Offer{ExpiresAt: now.Add(ttl)}
			if err := rows.Scan(&offer.EntryID, &offer.ChatID, &offer.SeatsWanted, &offer.FreeSeats,
				&offer.RouteName, &offer.InstructorName, &offer.StartAt, &offer.MeetingPoint); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan waitlist offer: %w", err)
			}
			offers = append(offers, offer)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate waitlist offers: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit waitlist offers: %w", err)
	}
	return offers, nil
}

// ReleaseOffer возвращает запись в очередь на прежнее место: отправить предложение не
// удалось по временной причине.
func (r *WaitlistRepository) ReleaseOffer(ctx context.Context, entryID string) error {
	if _, err := r.db.Exec(ctx, `
UPDATE waitlist_entries SET status = 'waiting', notified_at = NULL
WHERE id = $1 AND status = 'notified'`, entryID); err != nil {
		return fmt.Errorf("release waitlist offer: %w", err)
	}
	return nil
}

// ExpireOffer закрывает запись, которой предложение доставить нельзя вовсе.
func (r *WaitlistRepository) ExpireOffer(ctx context.Context, entryID string, now time.Time) error {
	if _, err := r.db.Exec(ctx, `
UPDATE waitlist_entries SET status = 'expired', closed_at = $2
WHERE id = $1 AND status = 'notified'`, entryID, now); err != nil {
		return fmt.Errorf("expire waitlist offer: %w", err)
	}
	return nil
}
