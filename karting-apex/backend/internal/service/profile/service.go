package profile

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"strings"
	"time"

	"summer-school-2026/backend/internal/service/auth"
)

var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidName     = errors.New("invalid name")
	ErrInvalidPhone    = errors.New("invalid phone")
	ErrInvalidCode     = errors.New("invalid code")
	ErrPhoneConflict   = errors.New("phone conflict")
	ErrTooManyRequests = errors.New("too many requests")
	phonePattern       = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	codePattern        = regexp.MustCompile(`^\d{4,6}$`)
)

const (
	phoneChangePurpose = "phone_change"
	otpCodeLength      = 4
)

type Client = auth.Client

type RequestPhoneCodeResult struct {
	TTLSeconds         int
	ResendAfterSeconds int
	Code               string
}

type Repository interface {
	ClientBySessionTokenHash(ctx context.Context, tokenHash string) (Client, bool, error)
	UpdateClientName(ctx context.Context, clientID, name string) (Client, error)
	FindClientByPhone(ctx context.Context, phone string) (Client, bool, error)
	LatestOTP(ctx context.Context, phone, purpose string) (auth.OTP, bool, error)
	CreateOTP(ctx context.Context, phone, purpose, codeHash string, expiresAt time.Time) error
	ConsumeOTP(ctx context.Context, id string, now time.Time) error
	IncrementOTPAttempts(ctx context.Context, id string) error
	ChangeClientPhone(ctx context.Context, clientID, newPhone, otpID string, now time.Time) (Client, error)
	DeleteClientAccount(ctx context.Context, clientID string, now time.Time) error
}

type Service struct {
	repo        Repository
	logger      *slog.Logger
	now         func() time.Time
	codeTTL     time.Duration
	resendAfter time.Duration
	maxAttempts int
}

func NewService(repo Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:        repo,
		logger:      logger,
		now:         time.Now,
		codeTTL:     5 * time.Minute,
		resendAfter: time.Minute,
		maxAttempts: 5,
	}
}

func (s *Service) Current(ctx context.Context, token string) (Client, error) {
	if token == "" {
		return Client{}, ErrUnauthorized
	}
	client, ok, err := s.repo.ClientBySessionTokenHash(ctx, auth.HashToken(token))
	if err != nil {
		return Client{}, err
	}
	if !ok {
		return Client{}, ErrUnauthorized
	}
	return client, nil
}

func (s *Service) UpdateName(ctx context.Context, token, name string) (Client, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 100 {
		return Client{}, ErrInvalidName
	}
	client, err := s.Current(ctx, token)
	if err != nil {
		return Client{}, err
	}
	return s.repo.UpdateClientName(ctx, client.ID, name)
}

func (s *Service) RequestPhoneChangeCode(ctx context.Context, token, newPhone string) (RequestPhoneCodeResult, error) {
	if !phonePattern.MatchString(newPhone) {
		return RequestPhoneCodeResult{}, ErrInvalidPhone
	}
	client, err := s.Current(ctx, token)
	if err != nil {
		return RequestPhoneCodeResult{}, err
	}
	if client.Phone == newPhone {
		return RequestPhoneCodeResult{}, ErrPhoneConflict
	}
	if _, found, err := s.repo.FindClientByPhone(ctx, newPhone); err != nil {
		return RequestPhoneCodeResult{}, err
	} else if found {
		return RequestPhoneCodeResult{}, ErrPhoneConflict
	}

	now := s.now().UTC()
	latest, ok, err := s.repo.LatestOTP(ctx, newPhone, phoneChangePurpose)
	if err != nil {
		return RequestPhoneCodeResult{}, err
	}
	if ok && latest.ConsumedAt == nil && now.Sub(latest.CreatedAt) < s.resendAfter {
		return RequestPhoneCodeResult{}, ErrTooManyRequests
	}

	code, err := randomDigits(otpCodeLength)
	if err != nil {
		return RequestPhoneCodeResult{}, err
	}
	if err := s.repo.CreateOTP(ctx, newPhone, phoneChangePurpose, auth.HashOTP(newPhone, phoneChangePurpose, code), now.Add(s.codeTTL)); err != nil {
		return RequestPhoneCodeResult{}, err
	}
	s.logger.Info("dev otp generated", "phone", newPhone, "purpose", phoneChangePurpose, "code", code)

	return RequestPhoneCodeResult{TTLSeconds: int(s.codeTTL.Seconds()), ResendAfterSeconds: int(s.resendAfter.Seconds()), Code: code}, nil
}

func (s *Service) ConfirmPhoneChange(ctx context.Context, token, newPhone, code string) (Client, error) {
	if !phonePattern.MatchString(newPhone) || !codePattern.MatchString(code) {
		return Client{}, ErrInvalidCode
	}
	client, err := s.Current(ctx, token)
	if err != nil {
		return Client{}, err
	}
	if client.Phone == newPhone {
		return Client{}, ErrPhoneConflict
	}
	if _, found, err := s.repo.FindClientByPhone(ctx, newPhone); err != nil {
		return Client{}, err
	} else if found {
		return Client{}, ErrPhoneConflict
	}

	now := s.now().UTC()
	otp, ok, err := s.repo.LatestOTP(ctx, newPhone, phoneChangePurpose)
	if err != nil {
		return Client{}, err
	}
	if !ok || otp.ConsumedAt != nil || !now.Before(otp.ExpiresAt) || otp.AttemptCount >= s.maxAttempts {
		return Client{}, ErrInvalidCode
	}
	if otp.CodeHash != auth.HashOTP(newPhone, phoneChangePurpose, code) {
		_ = s.repo.IncrementOTPAttempts(ctx, otp.ID)
		return Client{}, ErrInvalidCode
	}

	return s.repo.ChangeClientPhone(ctx, client.ID, newPhone, otp.ID, now)
}

func (s *Service) DeleteAccount(ctx context.Context, token string) error {
	client, err := s.Current(ctx, token)
	if err != nil {
		return err
	}
	return s.repo.DeleteClientAccount(ctx, client.ID, s.now().UTC())
}

func randomDigits(length int) (string, error) {
	buf := make([]byte, length)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate otp digit: %w", err)
		}
		buf[i] = byte('0' + n.Int64())
	}
	return string(buf), nil
}
