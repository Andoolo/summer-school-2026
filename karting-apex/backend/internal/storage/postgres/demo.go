package postgres

import (
	"context"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/service/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DemoRepository — гостевые аккаунты: создание и уборка истёкших.
type DemoRepository struct {
	*AuthRepository
	db *pgxpool.Pool
}

func NewDemoRepository(db *pgxpool.Pool) *DemoRepository {
	return &DemoRepository{AuthRepository: NewAuthRepository(db), db: db}
}

func (r *DemoRepository) CountActiveDemoClients(ctx context.Context, now time.Time) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `
SELECT count(*) FROM clients
WHERE demo_expires_at IS NOT NULL AND demo_expires_at > $1 AND deleted_at IS NULL`, now).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active demo clients: %w", err)
	}
	return count, nil
}

func (r *DemoRepository) CreateDemoClient(ctx context.Context, phone, name string, now, expiresAt time.Time) (auth.Client, error) {
	var client auth.Client
	err := r.db.QueryRow(ctx, `
INSERT INTO clients (phone, name, created_at, demo_expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id::text, name, phone, created_at, demo_expires_at`, phone, name, now, expiresAt).
		Scan(&client.ID, &client.Name, &client.Phone, &client.CreatedAt, &client.DemoExpiresAt)
	if err != nil {
		if isUniqueViolation(err) {
			return auth.Client{}, auth.ErrDemoPhoneTaken
		}
		return auth.Client{}, fmt.Errorf("create demo client: %w", err)
	}
	return client, nil
}

// CleanupExpiredDemoClients удаляет истёкших гостей и возвращает места их броней в заезды.
// Удаление полное, а не мягкое: гость ничего не значит после истечения, а без удаления
// таблица клиентов росла бы с каждым входом. За один вызов — не больше batch гостей,
// чтобы не держать долгую транзакцию; остаток уйдёт в следующий запуск.
func CleanupExpiredDemoClients(ctx context.Context, db *pgxpool.Pool, now time.Time, batch int) (int, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin demo cleanup: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
SELECT id FROM clients
WHERE demo_expires_at IS NOT NULL AND demo_expires_at <= $1
ORDER BY demo_expires_at
LIMIT $2
FOR UPDATE SKIP LOCKED`, now, batch)
	if err != nil {
		return 0, fmt.Errorf("select expired demo clients: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, fmt.Errorf("collect expired demo clients: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	if err := releaseFutureSeats(ctx, tx, ids, now); err != nil {
		return 0, err
	}
	// Результаты кругов удаляются каскадом от броней, сессии и ключи идемпотентности —
	// каскадом от клиента.
	if _, err := tx.Exec(ctx, `DELETE FROM bookings WHERE client_id = ANY($1::uuid[])`, ids); err != nil {
		return 0, fmt.Errorf("delete demo bookings: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM clients WHERE id = ANY($1::uuid[])`, ids); err != nil {
		return 0, fmt.Errorf("delete demo clients: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit demo cleanup: %w", err)
	}
	return len(ids), nil
}

// releaseFutureSeats возвращает в заезды места, которые занимали брони клиентов, если
// заезд ещё не начался. Отменённая вовремя бронь места уже вернула; поздняя отмена
// (late_cancel) места по правилам удерживает — но клиента больше нет, и держать место
// за пустотой нельзя.
//
// Возврат ограничен вместимостью заезда. Если счёт мест уже рассогласован (например,
// старым сидом), без ограничения UPDATE нарушил бы CHECK, уборка падала бы на каждом
// запуске и гости копились бы до потолка — демо-вход перестал бы работать.
func releaseFutureSeats(ctx context.Context, tx pgx.Tx, clientIDs []string, now time.Time) error {
	if _, err := tx.Exec(ctx, `
UPDATE slots s
SET free_seats = LEAST(s.total_seats, s.free_seats + held.seats),
    free_rental_boards = LEAST(s.rental_boards_total, s.free_rental_boards + held.boards)
FROM (
    SELECT b.slot_id, sum(b.seats_count)::int AS seats, sum(b.rental_count)::int AS boards
    FROM bookings b
    WHERE b.client_id = ANY($1::uuid[]) AND b.status IN ('active', 'late_cancel')
    GROUP BY b.slot_id
) held
WHERE s.id = held.slot_id AND s.start_at > $2`, clientIDs, now); err != nil {
		return fmt.Errorf("release held seats: %w", err)
	}
	return nil
}
