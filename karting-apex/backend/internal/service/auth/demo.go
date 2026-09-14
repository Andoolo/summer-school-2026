package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

var (
	// ErrDemoUnavailable — гостевой вход временно недоступен: достигнут потолок
	// одновременно живых гостей.
	ErrDemoUnavailable = errors.New("demo login unavailable")
	// ErrDemoPhoneTaken — сгенерированный номер уже занят; вызывающий повторяет попытку.
	ErrDemoPhoneTaken = errors.New("demo phone taken")
)

const (
	// DemoPhonePrefix — номера гостей. Код +7 000 в российской нумерации не выделен
	// ни одному оператору, поэтому номер гостя не совпадёт с чьим-то настоящим.
	DemoPhonePrefix = "+7000"
	demoGuestName   = "Гость"
	demoPhoneTries  = 5
)

// DemoConfig — параметры гостевого входа.
type DemoConfig struct {
	// TTL — сколько живёт гость; вместе с ним истекают его сессии.
	TTL time.Duration
	// MaxActive — потолок одновременно живых гостей. Защищает базу от роста, если
	// кто-то будет создавать гостей в цикле с разных адресов.
	MaxActive int
}

func DefaultDemoConfig() DemoConfig {
	return DemoConfig{TTL: 24 * time.Hour, MaxActive: 300}
}

// IsDemoPhone — номер из диапазона гостей. По таким номерам нельзя войти кодом: иначе
// номер гостя можно было бы «угнать», запросив на него код.
func IsDemoPhone(phone string) bool {
	return strings.HasPrefix(phone, DemoPhonePrefix)
}

// DemoRepository — хранилище гостей. Отдельно от Repository, чтобы сервис входа по коду
// и его тестовые подделки не зависели от гостевого входа.
type DemoRepository interface {
	CountActiveDemoClients(ctx context.Context, now time.Time) (int, error)
	CreateDemoClient(ctx context.Context, phone, name string, now, expiresAt time.Time) (Client, error)
	IssueSession(ctx context.Context, clientID, accessHash, refreshHash string, accessExpiresAt, refreshExpiresAt time.Time) error
}

// DemoService выдаёт гостевые аккаунты.
type DemoService struct {
	repo   DemoRepository
	config DemoConfig
	now    func() time.Time
	// sessionTTL совпадает с сервисом входа: гость не должен жить дольше обычной сессии.
	sessionTTL  time.Duration
	randomPhone func() (string, error)
}

func NewDemoService(repo DemoRepository, config DemoConfig) *DemoService {
	return &DemoService{
		repo:        repo,
		config:      config,
		now:         time.Now,
		sessionTTL:  24 * time.Hour,
		randomPhone: randomDemoPhone,
	}
}

// Login создаёт нового гостя и сразу выдаёт ему сессию. Каждый вход — новый гость:
// гости не видят и не отменяют записи друг друга.
func (s *DemoService) Login(ctx context.Context) (VerifyCodeResult, error) {
	now := s.now().UTC()

	active, err := s.repo.CountActiveDemoClients(ctx, now)
	if err != nil {
		return VerifyCodeResult{}, err
	}
	if active >= s.config.MaxActive {
		return VerifyCodeResult{}, ErrDemoUnavailable
	}

	expiresAt := now.Add(s.config.TTL)
	var client Client
	for attempt := 0; ; attempt++ {
		phone, err := s.randomPhone()
		if err != nil {
			return VerifyCodeResult{}, err
		}
		client, err = s.repo.CreateDemoClient(ctx, phone, demoGuestName, now, expiresAt)
		if err == nil {
			break
		}
		if !errors.Is(err, ErrDemoPhoneTaken) || attempt+1 >= demoPhoneTries {
			return VerifyCodeResult{}, err
		}
	}

	token, err := randomToken()
	if err != nil {
		return VerifyCodeResult{}, err
	}
	refresh, err := randomToken()
	if err != nil {
		return VerifyCodeResult{}, err
	}
	// Ни access, ни refresh не переживают самого гостя: после уборки войти в его
	// аккаунт нечем.
	accessExpiresAt := now.Add(s.sessionTTL)
	if accessExpiresAt.After(expiresAt) {
		accessExpiresAt = expiresAt
	}
	if err := s.repo.IssueSession(ctx, client.ID, HashToken(token), HashToken(refresh), accessExpiresAt, expiresAt); err != nil {
		return VerifyCodeResult{}, err
	}

	return VerifyCodeResult{
		Token:            token,
		RefreshToken:     refresh,
		AccessTTLSeconds: int(accessExpiresAt.Sub(now).Seconds()),
		Client:           client,
		IsNew:            true,
	}, nil
}

// randomDemoPhone — +7000 и 7 случайных цифр: 10 миллионов вариантов, при потолке в
// сотни гостей коллизии редки и закрываются повтором.
func randomDemoPhone() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10_000_000))
	if err != nil {
		return "", fmt.Errorf("generate demo phone: %w", err)
	}
	return fmt.Sprintf("%s%07d", DemoPhonePrefix, n.Int64()), nil
}
