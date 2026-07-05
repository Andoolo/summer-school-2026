package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"time"
)

var (
	ErrInvalidPhone    = errors.New("invalid phone")
	ErrInvalidCode     = errors.New("invalid code")
	ErrTooManyRequests = errors.New("too many requests")
	ErrInvalidSession  = errors.New("invalid session")
	phonePattern       = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	codePattern        = regexp.MustCompile(`^\d{4,6}$`)
)

const (
	loginPurpose  = "login"
	otpCodeLength = 4
)

type Client struct {
	ID        string
	Name      *string
	Phone     string
	CreatedAt time.Time
}

type RequestCodeResult struct {
	TTLSeconds         int
	ResendAfterSeconds int
	Code               string
}

type VerifyCodeResult struct {
	Token            string // access-токен сессии (кладётся в Authorization)
	RefreshToken     string // refresh-токен для обновления сессии через /auth/refresh
	AccessTTLSeconds int    // срок жизни access-токена в секундах (для expires_in в TokenPair)
	Client           Client
	IsNew            bool
}

// RefreshResult — новая пара токенов после обмена refresh-токена (/auth/refresh).
type RefreshResult struct {
	AccessToken      string
	RefreshToken     string
	AccessTTLSeconds int
}

type Repository interface {
	LatestOTP(ctx context.Context, phone, purpose string) (OTP, bool, error)
	CreateOTP(ctx context.Context, phone, purpose, codeHash string, expiresAt time.Time) error
	ConsumeOTP(ctx context.Context, id string, now time.Time) error
	IncrementOTPAttempts(ctx context.Context, id string) error
	FindClientByPhone(ctx context.Context, phone string) (Client, bool, error)
	CreateClient(ctx context.Context, phone string, now time.Time) (Client, error)
	IssueSession(ctx context.Context, clientID, accessHash, refreshHash string, accessExpiresAt, refreshExpiresAt time.Time) error
	RotateSession(ctx context.Context, oldRefreshHash, newAccessHash, newRefreshHash string, now, accessExpiresAt, refreshExpiresAt time.Time) (bool, error)
	RevokeSessionByAccessToken(ctx context.Context, accessHash string, now time.Time) (bool, error)
}

type OTP struct {
	ID           string
	CodeHash     string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	ConsumedAt   *time.Time
	AttemptCount int
}

type Service struct {
	repo        Repository
	logger      *slog.Logger
	now         func() time.Time
	codeTTL     time.Duration
	resendAfter time.Duration
	sessionTTL  time.Duration
	refreshTTL  time.Duration
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
		sessionTTL:  24 * time.Hour,
		refreshTTL:  30 * 24 * time.Hour,
		maxAttempts: 5,
	}
}

func (s *Service) RequestCode(ctx context.Context, phone string) (RequestCodeResult, error) {
	if !phonePattern.MatchString(phone) {
		return RequestCodeResult{}, ErrInvalidPhone
	}

	now := s.now().UTC()
	latest, ok, err := s.repo.LatestOTP(ctx, phone, loginPurpose)
	if err != nil {
		return RequestCodeResult{}, err
	}
	if ok && latest.ConsumedAt == nil && now.Sub(latest.CreatedAt) < s.resendAfter {
		return RequestCodeResult{}, ErrTooManyRequests
	}

	code, err := randomDigits(otpCodeLength)
	if err != nil {
		return RequestCodeResult{}, err
	}
	if err := s.repo.CreateOTP(ctx, phone, loginPurpose, HashOTP(phone, loginPurpose, code), now.Add(s.codeTTL)); err != nil {
		return RequestCodeResult{}, err
	}

	s.logger.Info("dev otp generated", "phone", phone, "purpose", loginPurpose, "code", code)
	return RequestCodeResult{TTLSeconds: int(s.codeTTL.Seconds()), ResendAfterSeconds: int(s.resendAfter.Seconds()), Code: code}, nil
}

func (s *Service) VerifyCode(ctx context.Context, phone, code string) (VerifyCodeResult, error) {
	if !phonePattern.MatchString(phone) || !codePattern.MatchString(code) {
		return VerifyCodeResult{}, ErrInvalidCode
	}

	now := s.now().UTC()
	otp, ok, err := s.repo.LatestOTP(ctx, phone, loginPurpose)
	if err != nil {
		return VerifyCodeResult{}, err
	}
	if !ok || otp.ConsumedAt != nil || !now.Before(otp.ExpiresAt) || otp.AttemptCount >= s.maxAttempts {
		return VerifyCodeResult{}, ErrInvalidCode
	}
	if otp.CodeHash != HashOTP(phone, loginPurpose, code) {
		_ = s.repo.IncrementOTPAttempts(ctx, otp.ID)
		return VerifyCodeResult{}, ErrInvalidCode
	}

	client, found, err := s.repo.FindClientByPhone(ctx, phone)
	if err != nil {
		return VerifyCodeResult{}, err
	}
	isNew := false
	if !found {
		client, err = s.repo.CreateClient(ctx, phone, now)
		if err != nil {
			return VerifyCodeResult{}, err
		}
		isNew = true
	}

	token, err := randomToken()
	if err != nil {
		return VerifyCodeResult{}, err
	}
	refresh, err := randomToken()
	if err != nil {
		return VerifyCodeResult{}, err
	}
	if err := s.repo.IssueSession(ctx, client.ID, HashToken(token), HashToken(refresh), now.Add(s.sessionTTL), now.Add(s.refreshTTL)); err != nil {
		return VerifyCodeResult{}, err
	}
	if err := s.repo.ConsumeOTP(ctx, otp.ID, now); err != nil {
		return VerifyCodeResult{}, err
	}

	return VerifyCodeResult{
		Token:            token,
		RefreshToken:     refresh,
		AccessTTLSeconds: int(s.sessionTTL.Seconds()),
		Client:           client,
		IsNew:            isNew,
	}, nil
}

// Refresh обменивает действительный refresh-токен на новую пару access+refresh (R-016).
// Ротация выполняется атомарно в repo (RotateSession): гашение старого refresh через
// UPDATE ... RETURNING защищает от повторного использования при гонке. Недействительный/
// просроченный/уже использованный refresh → ErrInvalidSession (→ 401 в handler).
func (s *Service) Refresh(ctx context.Context, refreshToken string) (RefreshResult, error) {
	if refreshToken == "" {
		return RefreshResult{}, ErrInvalidSession
	}
	now := s.now().UTC()

	access, err := randomToken()
	if err != nil {
		return RefreshResult{}, err
	}
	refresh, err := randomToken()
	if err != nil {
		return RefreshResult{}, err
	}

	ok, err := s.repo.RotateSession(ctx, HashToken(refreshToken), HashToken(access), HashToken(refresh), now, now.Add(s.sessionTTL), now.Add(s.refreshTTL))
	if err != nil {
		return RefreshResult{}, err
	}
	if !ok {
		return RefreshResult{}, ErrInvalidSession
	}

	return RefreshResult{
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessTTLSeconds: int(s.sessionTTL.Seconds()),
	}, nil
}

// Logout завершает текущую сессию по access-токену: гасит саму сессию и связанные с ней
// refresh-токены (только это устройство — не другие устройства клиента), R-016.
// Если живой сессии с таким токеном нет → ErrInvalidSession (→ 401).
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidSession
	}
	found, err := s.repo.RevokeSessionByAccessToken(ctx, HashToken(token), s.now().UTC())
	if err != nil {
		return err
	}
	if !found {
		return ErrInvalidSession
	}
	return nil
}

func HashOTP(phone, purpose, code string) string {
	sum := sha256.Sum256([]byte(phone + "|" + purpose + "|" + code))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(sum[:])
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

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
