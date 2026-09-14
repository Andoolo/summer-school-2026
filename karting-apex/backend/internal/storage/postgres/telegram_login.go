package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/service/telegramlogin"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TelegramLoginRepository struct {
	db *pgxpool.Pool
}

func NewTelegramLoginRepository(db *pgxpool.Pool) *TelegramLoginRepository {
	return &TelegramLoginRepository{db: db}
}

func (r *TelegramLoginRepository) CreateLoginRequest(ctx context.Context, startHash, pollHash, code string, now, expiresAt time.Time) error {
	if _, err := r.db.Exec(ctx, `
INSERT INTO telegram_login_requests (start_token_hash, poll_token_hash, confirm_code, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5)`, startHash, pollHash, code, now, expiresAt); err != nil {
		return fmt.Errorf("create telegram login request: %w", err)
	}
	return nil
}

func (r *TelegramLoginRepository) AttachChat(ctx context.Context, startHash string, chatID int64, now time.Time) (telegramlogin.Request, bool, error) {
	var request telegramlogin.Request
	err := r.db.QueryRow(ctx, `
UPDATE telegram_login_requests
SET chat_id = $2
WHERE start_token_hash = $1
  AND status = 'pending'
  AND expires_at > $3
  AND (chat_id IS NULL OR chat_id = $2)
RETURNING confirm_code, status, expires_at`, startHash, chatID, now).Scan(&request.ConfirmCode, &request.Status, &request.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return telegramlogin.Request{}, false, nil
	}
	if err != nil {
		return telegramlogin.Request{}, false, fmt.Errorf("attach telegram chat: %w", err)
	}
	return request, true, nil
}

func (r *TelegramLoginRepository) ConfirmLatestForChat(ctx context.Context, chatID int64, phone, firstName string, now time.Time) (bool, error) {
	tag, err := r.db.Exec(ctx, `
UPDATE telegram_login_requests
SET status = 'confirmed', phone = $2, first_name = NULLIF($3, '')
WHERE id = (
    SELECT id FROM telegram_login_requests
    WHERE chat_id = $1 AND status = 'pending' AND expires_at > $4
    ORDER BY created_at DESC
    LIMIT 1
    FOR UPDATE
)`, chatID, phone, firstName, now)
	if err != nil {
		return false, fmt.Errorf("confirm telegram login: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ConsumeConfirmed гасит запрос одним UPDATE … RETURNING: из параллельных опросов строку
// изменит только один, остальные получат false — сессия выдаётся один раз.
func (r *TelegramLoginRepository) ConsumeConfirmed(ctx context.Context, pollHash string, now time.Time) (telegramlogin.Request, bool, error) {
	var request telegramlogin.Request
	var firstName *string
	err := r.db.QueryRow(ctx, `
UPDATE telegram_login_requests
SET status = 'consumed'
WHERE poll_token_hash = $1 AND status = 'confirmed' AND expires_at > $2
RETURNING confirm_code, status, expires_at, phone, first_name, chat_id`, pollHash, now).
		Scan(&request.ConfirmCode, &request.Status, &request.ExpiresAt, &request.Phone, &firstName, &request.ChatID)
	if errors.Is(err, pgx.ErrNoRows) {
		return telegramlogin.Request{}, false, nil
	}
	if err != nil {
		return telegramlogin.Request{}, false, fmt.Errorf("consume telegram login: %w", err)
	}
	if firstName != nil {
		request.FirstName = *firstName
	}
	return request, true, nil
}

func (r *TelegramLoginRepository) RequestByPollHash(ctx context.Context, pollHash string) (telegramlogin.Request, bool, error) {
	var request telegramlogin.Request
	err := r.db.QueryRow(ctx, `
SELECT confirm_code, status, expires_at
FROM telegram_login_requests
WHERE poll_token_hash = $1`, pollHash).Scan(&request.ConfirmCode, &request.Status, &request.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return telegramlogin.Request{}, false, nil
	}
	if err != nil {
		return telegramlogin.Request{}, false, fmt.Errorf("query telegram login: %w", err)
	}
	return request, true, nil
}

// LinkChat переносит чат на клиента с номером: сначала отвязывает его у прежнего
// клиента (человек сменил номер в Telegram), затем привязывает — в одной транзакции,
// чтобы не нарушить уникальность чата.
func (r *TelegramLoginRepository) LinkChat(ctx context.Context, phone string, chatID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin link telegram chat: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
UPDATE clients SET telegram_chat_id = NULL
WHERE telegram_chat_id = $2 AND phone <> $1`, phone, chatID); err != nil {
		return fmt.Errorf("unlink previous telegram chat owner: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE clients SET telegram_chat_id = $2
WHERE phone = $1 AND deleted_at IS NULL`, phone, chatID); err != nil {
		return fmt.Errorf("link telegram chat: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit link telegram chat: %w", err)
	}
	return nil
}

func (r *TelegramLoginRepository) SetNotifications(ctx context.Context, chatID int64, enabled bool) (bool, error) {
	tag, err := r.db.Exec(ctx, `
UPDATE clients SET telegram_notifications = $2
WHERE telegram_chat_id = $1 AND deleted_at IS NULL`, chatID, enabled)
	if err != nil {
		return false, fmt.Errorf("set telegram notifications: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteStaleTelegramLogins удаляет запросы входа, истёкшие больше суток назад: в них
// хранятся номера телефонов, держать их дольше незачем.
func DeleteStaleTelegramLogins(ctx context.Context, db *pgxpool.Pool, now time.Time) (int64, error) {
	tag, err := db.Exec(ctx, `DELETE FROM telegram_login_requests WHERE expires_at < $1`, now.Add(-24*time.Hour))
	if err != nil {
		return 0, fmt.Errorf("delete stale telegram logins: %w", err)
	}
	return tag.RowsAffected(), nil
}
