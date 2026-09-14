package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/ops"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OpsRepository struct {
	db *pgxpool.Pool
}

func NewOpsRepository(db *pgxpool.Pool) *OpsRepository {
	return &OpsRepository{db: db}
}

func (r *OpsRepository) AddCounters(ctx context.Context, hour time.Time, counts map[string]int64) error {
	names := make([]string, 0, len(counts))
	values := make([]int64, 0, len(counts))
	for name, value := range counts {
		names = append(names, name)
		values = append(values, value)
	}
	if len(names) == 0 {
		return nil
	}
	if _, err := r.db.Exec(ctx, `
INSERT INTO ops_counters (hour, name, value)
SELECT $1, name, value FROM unnest($2::text[], $3::bigint[]) AS t(name, value)
ON CONFLICT (hour, name) DO UPDATE SET value = ops_counters.value + EXCLUDED.value`, hour, names, values); err != nil {
		return fmt.Errorf("add ops counters: %w", err)
	}
	return nil
}

func (r *OpsRepository) State(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRow(ctx, `SELECT value FROM ops_state WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read ops state: %w", err)
	}
	return value, nil
}

func (r *OpsRepository) SetState(ctx context.Context, key, value string) error {
	if _, err := r.db.Exec(ctx, `
INSERT INTO ops_state (key, value, updated_at) VALUES ($1, $2, now())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, key, value); err != nil {
		return fmt.Errorf("write ops state: %w", err)
	}
	return nil
}

// DeleteOldCounters удаляет счётчики старше 30 дней.
func DeleteOldCounters(ctx context.Context, db *pgxpool.Pool, now time.Time) (int64, error) {
	tag, err := db.Exec(ctx, `DELETE FROM ops_counters WHERE hour < $1`, now.Add(-30*24*time.Hour))
	if err != nil {
		return 0, fmt.Errorf("delete old ops counters: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Snapshot — числа для /stats одним запросом на раздел. Гости (демо) в броням и клиентам
// считаются вместе со всеми, отдельно показано только их текущее число.
func (r *OpsRepository) Snapshot(ctx context.Context, now time.Time) (ops.Snapshot, error) {
	var s ops.Snapshot
	day, week := now.Add(-24*time.Hour), now.Add(-7*24*time.Hour)

	if err := r.db.QueryRow(ctx, `
SELECT
    count(*) FILTER (WHERE created_at > $1),
    count(*) FILTER (WHERE created_at > $2),
    count(*) FILTER (WHERE status <> 'active' AND cancelled_at > $1),
    count(*) FILTER (WHERE status <> 'active' AND cancelled_at > $2),
    count(*) FILTER (WHERE status = 'late_cancel' AND cancelled_at > $2)
FROM bookings
WHERE created_at > $2 OR cancelled_at > $2`, day, week).
		Scan(&s.BookingsCreated24h, &s.BookingsCreated7d, &s.Cancelled24h, &s.Cancelled7d, &s.LateCancelled7d); err != nil {
		return ops.Snapshot{}, fmt.Errorf("snapshot bookings: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
SELECT count(*), coalesce(sum(total_seats), 0), coalesce(sum(total_seats - free_seats), 0), count(*) FILTER (WHERE free_seats = 0)
FROM slots
WHERE status = 'scheduled' AND start_at > $1 AND start_at <= $2`, now, now.Add(7*24*time.Hour)).
		Scan(&s.UpcomingRaces, &s.UpcomingSeats, &s.UpcomingTaken, &s.UpcomingFull); err != nil {
		return ops.Snapshot{}, fmt.Errorf("snapshot slots: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
SELECT
    count(*) FILTER (WHERE status = 'waiting'),
    count(*) FILTER (WHERE notified_at > $1),
    count(*) FILTER (WHERE status = 'booked' AND notified_at IS NOT NULL AND closed_at > $1),
    count(*) FILTER (WHERE status = 'expired' AND notified_at IS NOT NULL AND closed_at > $1)
FROM waitlist_entries
WHERE status = 'waiting' OR notified_at > $1 OR closed_at > $1`, day).
		Scan(&s.WaitingNow, &s.Offers24h, &s.BookedFromWaitlist24h, &s.ExpiredOffers24h); err != nil {
		return ops.Snapshot{}, fmt.Errorf("snapshot waitlist: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
SELECT
    count(*) FILTER (WHERE demo_expires_at IS NULL),
    count(*) FILTER (WHERE telegram_chat_id IS NOT NULL),
    count(*) FILTER (WHERE demo_expires_at IS NULL AND created_at > $1),
    count(*) FILTER (WHERE demo_expires_at > $2)
FROM clients
WHERE deleted_at IS NULL`, week, now).
		Scan(&s.Clients, &s.TelegramClients, &s.NewClients7d, &s.ActiveGuests); err != nil {
		return ops.Snapshot{}, fmt.Errorf("snapshot clients: %w", err)
	}

	rows, err := r.db.Query(ctx, `SELECT name, sum(value) FROM ops_counters WHERE hour > $1 GROUP BY name`, now.Add(-24*time.Hour).Truncate(time.Hour))
	if err != nil {
		return ops.Snapshot{}, fmt.Errorf("snapshot counters: %w", err)
	}
	defer rows.Close()
	s.Counters24h = map[string]int64{}
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return ops.Snapshot{}, fmt.Errorf("scan counter: %w", err)
		}
		s.Counters24h[name] = value
	}
	if err := rows.Err(); err != nil {
		return ops.Snapshot{}, fmt.Errorf("iterate counters: %w", err)
	}
	return s, nil
}
