package postgres_test

import (
	"context"
	"testing"
	"time"

	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestOTPStatsCountsCodesAndFailuresByWindow(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	now := time.Now().UTC()
	phone := "+79995550101"
	insert := func(createdAt time.Time, purpose string, attempts int, forPhone string) {
		t.Helper()
		if _, err := db.Exec(ctx, `
INSERT INTO otp_codes (phone, purpose, code_hash, created_at, expires_at, attempt_count)
VALUES ($1, $2, 'hash', $3, $4, $5)`, forPhone, purpose, createdAt, createdAt.Add(5*time.Minute), attempts); err != nil {
			t.Fatalf("insert otp: %v", err)
		}
	}
	insert(now.Add(-10*time.Minute), "login", 3, phone)          // час и сутки
	insert(now.Add(-40*time.Minute), "login", 2, phone)          // час и сутки
	insert(now.Add(-5*time.Hour), "login", 4, phone)             // только сутки
	insert(now.Add(-30*time.Hour), "login", 5, phone)            // вне окна
	insert(now.Add(-10*time.Minute), "phone_change", 1, phone)   // другая цель
	insert(now.Add(-10*time.Minute), "login", 1, "+79995550102") // другой номер

	stats, err := postgres.NewAuthRepository(db).OTPStats(ctx, phone, "login", now)
	if err != nil {
		t.Fatalf("OTPStats() error = %v", err)
	}
	if stats.CodesLastHour != 2 || stats.CodesLastDay != 3 {
		t.Fatalf("codes hour/day = %d/%d, want 2/3", stats.CodesLastHour, stats.CodesLastDay)
	}
	if stats.FailedLastHour != 5 || stats.FailedLastDay != 9 {
		t.Fatalf("failed hour/day = %d/%d, want 5/9", stats.FailedLastHour, stats.FailedLastDay)
	}

	empty, err := postgres.NewProfileRepository(db).OTPStats(ctx, "+79995550199", "phone_change", now)
	if err != nil {
		t.Fatalf("OTPStats(empty) error = %v", err)
	}
	if empty.CodesLastHour != 0 || empty.CodesLastDay != 0 || empty.FailedLastHour != 0 || empty.FailedLastDay != 0 {
		t.Fatalf("empty stats = %+v, want zeros", empty)
	}
}
