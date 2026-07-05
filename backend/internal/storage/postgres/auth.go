package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/service/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) LatestOTP(ctx context.Context, phone, purpose string) (auth.OTP, bool, error) {
	var otp auth.OTP
	err := r.db.QueryRow(ctx, `
SELECT id::text, code_hash, created_at, expires_at, consumed_at, attempt_count
FROM otp_codes
WHERE phone = $1 AND purpose = $2
ORDER BY created_at DESC
LIMIT 1`, phone, purpose).Scan(&otp.ID, &otp.CodeHash, &otp.CreatedAt, &otp.ExpiresAt, &otp.ConsumedAt, &otp.AttemptCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.OTP{}, false, nil
	}
	if err != nil {
		return auth.OTP{}, false, fmt.Errorf("query latest otp: %w", err)
	}
	return otp, true, nil
}

func (r *AuthRepository) CreateOTP(ctx context.Context, phone, purpose, codeHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO otp_codes (phone, purpose, code_hash, expires_at)
VALUES ($1, $2, $3, $4)`, phone, purpose, codeHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create otp: %w", err)
	}
	return nil
}

func (r *AuthRepository) ConsumeOTP(ctx context.Context, id string, now time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE otp_codes SET consumed_at = $2 WHERE id = $1`, id, now)
	if err != nil {
		return fmt.Errorf("consume otp: %w", err)
	}
	return nil
}

func (r *AuthRepository) IncrementOTPAttempts(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE otp_codes SET attempt_count = attempt_count + 1 WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("increment otp attempts: %w", err)
	}
	return nil
}

func (r *AuthRepository) FindClientByPhone(ctx context.Context, phone string) (auth.Client, bool, error) {
	var client auth.Client
	err := r.db.QueryRow(ctx, `
SELECT id::text, name, phone, created_at
FROM clients
WHERE phone = $1 AND deleted_at IS NULL`, phone).Scan(&client.ID, &client.Name, &client.Phone, &client.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Client{}, false, nil
	}
	if err != nil {
		return auth.Client{}, false, fmt.Errorf("find client by phone: %w", err)
	}
	return client, true, nil
}

func (r *AuthRepository) CreateClient(ctx context.Context, phone string, now time.Time) (auth.Client, error) {
	var client auth.Client
	err := r.db.QueryRow(ctx, `
INSERT INTO clients (phone, created_at)
VALUES ($1, $2)
RETURNING id::text, name, phone, created_at`, phone, now).Scan(&client.ID, &client.Name, &client.Phone, &client.CreatedAt)
	if err != nil {
		return auth.Client{}, fmt.Errorf("create client: %w", err)
	}
	return client, nil
}

// IssueSession атомарно создаёт access-сессию и связанный с ней refresh-токен (одна транзакция).
func (r *AuthRepository) IssueSession(ctx context.Context, clientID, accessHash, refreshHash string, accessExpiresAt, refreshExpiresAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin issue session: %w", err)
	}
	defer tx.Rollback(ctx)

	var sessionID string
	if err := tx.QueryRow(ctx, `
INSERT INTO auth_sessions (client_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id::text`, clientID, accessHash, accessExpiresAt).Scan(&sessionID); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO refresh_tokens (client_id, session_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)`, clientID, sessionID, refreshHash, refreshExpiresAt); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit issue session: %w", err)
	}
	return nil
}

// RotateSession атомарно (одна транзакция) гасит предъявленный refresh-токен и выдаёт новую пару
// access+refresh. Гашение через UPDATE ... RETURNING защищает от повторного использования refresh
// при гонке: только один из конкурентных запросов затронет строку (revoked_at IS NULL) и получит
// client_id; остальные вернут ok=false. Возвращает ok=false для недействительного/истёкшего/уже
// использованного refresh — без выдачи новой пары.
func (r *AuthRepository) RotateSession(ctx context.Context, oldRefreshHash, newAccessHash, newRefreshHash string, now, accessExpiresAt, refreshExpiresAt time.Time) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin rotate session: %w", err)
	}
	defer tx.Rollback(ctx)

	var clientID string
	err = tx.QueryRow(ctx, `
UPDATE refresh_tokens
SET revoked_at = $2
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2
RETURNING client_id::text`, oldRefreshHash, now).Scan(&clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("consume refresh token: %w", err)
	}

	var sessionID string
	if err := tx.QueryRow(ctx, `
INSERT INTO auth_sessions (client_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id::text`, clientID, newAccessHash, accessExpiresAt).Scan(&sessionID); err != nil {
		return false, fmt.Errorf("insert rotated session: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO refresh_tokens (client_id, session_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)`, clientID, sessionID, newRefreshHash, refreshExpiresAt); err != nil {
		return false, fmt.Errorf("insert rotated refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit rotate session: %w", err)
	}
	return true, nil
}

// RevokeSessionByAccessToken завершает ТЕКУЩУЮ сессию по её access-токену: гасит саму сессию и
// только связанные с ней refresh-токены (не трогая другие устройства клиента) — одна транзакция.
// Возвращает found=false, если живой сессии с таким токеном нет.
func (r *AuthRepository) RevokeSessionByAccessToken(ctx context.Context, accessHash string, now time.Time) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin revoke session: %w", err)
	}
	defer tx.Rollback(ctx)

	var sessionID string
	err = tx.QueryRow(ctx, `
UPDATE auth_sessions
SET revoked_at = $2
WHERE token_hash = $1 AND revoked_at IS NULL
RETURNING id::text`, accessHash, now).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("revoke session: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE refresh_tokens
SET revoked_at = $2
WHERE session_id = $1 AND revoked_at IS NULL`, sessionID, now); err != nil {
		return false, fmt.Errorf("revoke session refresh tokens: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit revoke session: %w", err)
	}
	return true, nil
}
