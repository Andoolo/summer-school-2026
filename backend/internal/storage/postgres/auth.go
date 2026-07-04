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

func (r *AuthRepository) CreateSession(ctx context.Context, clientID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO auth_sessions (client_id, token_hash, expires_at)
VALUES ($1, $2, $3)`, clientID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *AuthRepository) RevokeSession(ctx context.Context, tokenHash string, now time.Time) error {
	_, err := r.db.Exec(ctx, `
UPDATE auth_sessions
SET revoked_at = $2
WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash, now)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// SessionClientID возвращает client_id живой (не отозванной, не истёкшей) access-сессии по хешу.
func (r *AuthRepository) SessionClientID(ctx context.Context, tokenHash string) (string, bool, error) {
	var clientID string
	err := r.db.QueryRow(ctx, `
SELECT client_id::text
FROM auth_sessions
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`, tokenHash).Scan(&clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("query session client: %w", err)
	}
	return clientID, true, nil
}

func (r *AuthRepository) CreateRefreshToken(ctx context.Context, clientID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO refresh_tokens (client_id, token_hash, expires_at)
VALUES ($1, $2, $3)`, clientID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

// RefreshTokenClientID возвращает client_id действительного refresh-токена по хешу
// (не отозван, не истёк на момент now).
func (r *AuthRepository) RefreshTokenClientID(ctx context.Context, tokenHash string, now time.Time) (string, bool, error) {
	var clientID string
	err := r.db.QueryRow(ctx, `
SELECT client_id::text
FROM refresh_tokens
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2`, tokenHash, now).Scan(&clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("query refresh token: %w", err)
	}
	return clientID, true, nil
}

func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string, now time.Time) error {
	_, err := r.db.Exec(ctx, `
UPDATE refresh_tokens
SET revoked_at = $2
WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash, now)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *AuthRepository) RevokeClientRefreshTokens(ctx context.Context, clientID string, now time.Time) error {
	_, err := r.db.Exec(ctx, `
UPDATE refresh_tokens
SET revoked_at = $2
WHERE client_id = $1 AND revoked_at IS NULL`, clientID, now)
	if err != nil {
		return fmt.Errorf("revoke client refresh tokens: %w", err)
	}
	return nil
}
