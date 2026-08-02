package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHashOTPDependsOnPhonePurposeAndCode(t *testing.T) {
	hash := HashOTP("+79991234567", "login", "123456")
	if hash == "" {
		t.Fatal("hash must not be empty")
	}
	if hash == "123456" {
		t.Fatal("otp must not be stored as plain code")
	}
	if hash == HashOTP("+79991234567", "phone_change", "123456") {
		t.Fatal("hash must depend on purpose")
	}
}

func TestRandomDigits(t *testing.T) {
	code, err := randomDigits(otpCodeLength)
	if err != nil {
		t.Fatalf("randomDigits() error = %v", err)
	}
	if !codePattern.MatchString(code) || len(code) != otpCodeLength {
		t.Fatalf("code = %q, want %d digits", code, otpCodeLength)
	}
}

func TestRequestCodeRateLimit(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{latestOTP: OTP{CreatedAt: now.Add(-30 * time.Second), ExpiresAt: now.Add(5 * time.Minute)}}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	_, err := service.RequestCode(context.Background(), "+79991234567")
	if !errors.Is(err, ErrTooManyRequests) {
		t.Fatalf("RequestCode() error = %v, want %v", err, ErrTooManyRequests)
	}
}

func TestVerifyCodeRejectsInvalidCode(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{latestOTP: OTP{ID: "otp-1", CodeHash: HashOTP("+79991234567", "login", "123456"), CreatedAt: now, ExpiresAt: now.Add(5 * time.Minute)}}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	_, err := service.VerifyCode(context.Background(), "+79991234567", "654321")
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("VerifyCode() error = %v, want %v", err, ErrInvalidCode)
	}
	if repo.attempts != 1 {
		t.Fatalf("attempts = %d, want 1", repo.attempts)
	}
}

func TestVerifyCodeRejectsExpiredCode(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{latestOTP: OTP{ID: "otp-1", CodeHash: HashOTP("+79991234567", "login", "123456"), CreatedAt: now.Add(-10 * time.Minute), ExpiresAt: now.Add(-5 * time.Minute)}}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	_, err := service.VerifyCode(context.Background(), "+79991234567", "123456")
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("VerifyCode() error = %v, want %v", err, ErrInvalidCode)
	}
}

func TestVerifyCodeRejectsConsumedCode(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	consumedAt := now.Add(-time.Minute)
	repo := &fakeRepo{latestOTP: OTP{ID: "otp-1", CodeHash: HashOTP("+79991234567", "login", "123456"), CreatedAt: now.Add(-2 * time.Minute), ExpiresAt: now.Add(3 * time.Minute), ConsumedAt: &consumedAt}}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	_, err := service.VerifyCode(context.Background(), "+79991234567", "123456")
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("VerifyCode() error = %v, want %v", err, ErrInvalidCode)
	}
}

type fakeRepo struct {
	latestOTP OTP
	attempts  int
}

func (r *fakeRepo) LatestOTP(context.Context, string, string) (OTP, bool, error) {
	return r.latestOTP, true, nil
}

func (r *fakeRepo) CreateOTP(context.Context, string, string, string, time.Time) error { return nil }

func (r *fakeRepo) ConsumeOTP(context.Context, string, time.Time) error { return nil }

func (r *fakeRepo) IncrementOTPAttempts(context.Context, string) error {
	r.attempts++
	return nil
}

func (r *fakeRepo) FindClientByPhone(context.Context, string) (Client, bool, error) {
	return Client{}, false, nil
}

func (r *fakeRepo) CreateClient(context.Context, string, time.Time) (Client, error) {
	return Client{ID: "00000000-0000-0000-0000-000000000000", Phone: "+79991234567"}, nil
}

func (r *fakeRepo) IssueSession(context.Context, string, string, string, time.Time, time.Time) error {
	return nil
}

func (r *fakeRepo) RotateSession(context.Context, string, string, string, time.Time, time.Time, time.Time) (bool, error) {
	return false, nil
}

func (r *fakeRepo) RevokeSessionByAccessToken(context.Context, string, time.Time) (bool, error) {
	return false, nil
}
