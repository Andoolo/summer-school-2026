package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOTPLimitsAllowRequest(t *testing.T) {
	limits := DefaultOTPLimits()
	cases := []struct {
		name  string
		stats OTPStats
		want  bool
	}{
		{"нет кодов", OTPStats{}, true},
		{"последний разрешённый в час", OTPStats{CodesLastHour: 4, CodesLastDay: 4}, true},
		{"исчерпан часовой лимит", OTPStats{CodesLastHour: 5, CodesLastDay: 5}, false},
		{"исчерпан суточный лимит", OTPStats{CodesLastHour: 0, CodesLastDay: 10}, false},
	}
	for _, tc := range cases {
		if got := limits.AllowRequest(tc.stats); got != tc.want {
			t.Errorf("%s: AllowRequest(%+v) = %v, want %v", tc.name, tc.stats, got, tc.want)
		}
	}
}

func TestOTPLimitsAllowVerify(t *testing.T) {
	limits := DefaultOTPLimits()
	if !limits.AllowVerify(OTPStats{FailedLastHour: 9, FailedLastDay: 19}) {
		t.Fatal("under limits must be allowed")
	}
	if limits.AllowVerify(OTPStats{FailedLastHour: 10, FailedLastDay: 10}) {
		t.Fatal("hourly failed limit must block")
	}
	if limits.AllowVerify(OTPStats{FailedLastHour: 0, FailedLastDay: 20}) {
		t.Fatal("daily failed limit must block")
	}
}

func TestMaskPhone(t *testing.T) {
	if got := MaskPhone("+79991234567"); got != "***4567" {
		t.Fatalf("MaskPhone() = %q", got)
	}
	if got := MaskPhone("+79"); got != "****" {
		t.Fatalf("MaskPhone(short) = %q", got)
	}
}

func TestRequestCodeRejectsWhenPhoneLimitReached(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	// Прошлый код давно погашен — пауза 60 секунд не мешает, упираемся именно в лимит.
	consumed := now.Add(-50 * time.Minute)
	repo := &fakeRepo{
		latestOTP: OTP{CreatedAt: now.Add(-55 * time.Minute), ConsumedAt: &consumed},
		stats:     OTPStats{CodesLastHour: 5, CodesLastDay: 5},
	}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	_, err := service.RequestCode(context.Background(), "+79991234567")
	if !errors.Is(err, ErrTooManyRequests) {
		t.Fatalf("RequestCode() error = %v, want %v", err, ErrTooManyRequests)
	}
	if repo.created != 0 {
		t.Fatalf("code must not be created when limit is reached, created = %d", repo.created)
	}
}

func TestVerifyCodeBlockedAfterTooManyFailures(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	code := "1234"
	repo := &fakeRepo{
		latestOTP: OTP{ID: "otp", CodeHash: HashOTP("+79991234567", loginPurpose, code), CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(4 * time.Minute)},
		stats:     OTPStats{FailedLastHour: 10, FailedLastDay: 10},
	}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	// Даже правильный код не принимается: иначе лимит не останавливал бы подбор.
	_, err := service.VerifyCode(context.Background(), "+79991234567", code)
	if !errors.Is(err, ErrTooManyRequests) {
		t.Fatalf("VerifyCode() error = %v, want %v", err, ErrTooManyRequests)
	}
	if repo.attempts != 0 {
		t.Fatalf("blocked verification must not count as attempt, attempts = %d", repo.attempts)
	}
}
