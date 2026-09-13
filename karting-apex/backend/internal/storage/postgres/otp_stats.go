package postgres

import (
	"context"
	"fmt"
	"time"

	"summer-school-2026/backend/internal/service/auth"

	"github.com/jackc/pgx/v5"
)

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// otpStats считает коды и неверные попытки по номеру за час и сутки одним запросом.
// Использует индекс otp_codes_phone_purpose_created_at_idx.
func otpStats(ctx context.Context, db queryRower, phone, purpose string, now time.Time) (auth.OTPStats, error) {
	var stats auth.OTPStats
	err := db.QueryRow(ctx, `
SELECT
    count(*) FILTER (WHERE created_at >= $3),
    count(*),
    coalesce(sum(attempt_count) FILTER (WHERE created_at >= $3), 0),
    coalesce(sum(attempt_count), 0)
FROM otp_codes
WHERE phone = $1 AND purpose = $2 AND created_at >= $4`,
		phone, purpose, now.Add(-time.Hour), now.Add(-24*time.Hour),
	).Scan(&stats.CodesLastHour, &stats.CodesLastDay, &stats.FailedLastHour, &stats.FailedLastDay)
	if err != nil {
		return auth.OTPStats{}, fmt.Errorf("query otp stats: %w", err)
	}
	return stats, nil
}

func (r *AuthRepository) OTPStats(ctx context.Context, phone, purpose string, now time.Time) (auth.OTPStats, error) {
	return otpStats(ctx, r.db, phone, purpose, now)
}

func (r *ProfileRepository) OTPStats(ctx context.Context, phone, purpose string, now time.Time) (auth.OTPStats, error) {
	return otpStats(ctx, r.db, phone, purpose, now)
}
