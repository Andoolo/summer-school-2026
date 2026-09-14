// Package waitlist — лист ожидания на заполненные заезды.
//
// Места раздаются по очереди с паузой: когда место освобождается, предложение получает
// первый в очереди, кому хватает мест. Если за OfferTTL он не записался, предложение
// переходит к следующему. Место при этом не закрепляется — записаться может любой.
// Предложения рассылает notify.Dispatcher; этот пакет — вход в очередь и выход из неё.
//
// Встать в очередь можно только с привязанным Telegram: уведомить больше нечем.
package waitlist

import (
	"context"
	"errors"
	"time"

	"summer-school-2026/backend/internal/service/auth"
)

var (
	ErrUnauthorized          = errors.New("unauthorized")
	ErrInvalidRequest        = errors.New("invalid waitlist request")
	ErrSlotNotFound          = errors.New("slot not found")
	ErrSlotCancelled         = errors.New("slot cancelled")
	ErrSlotStarted           = errors.New("slot started")
	ErrSeatsAvailable        = errors.New("seats are available, book instead")
	ErrAlreadyBooked         = errors.New("already booked")
	ErrTelegramRequired      = errors.New("telegram is required")
	ErrNotificationsDisabled = errors.New("telegram notifications are disabled")
	ErrTooManyEntries        = errors.New("too many waitlist entries")
)

const (
	MaxSeats = 3
	// MaxActiveEntries — в скольких очередях человек может стоять одновременно.
	MaxActiveEntries = 5
	// OfferTTL — сколько предложение ждёт брони, прежде чем перейти к следующему.
	OfferTTL = 15 * time.Minute
)

type Client struct {
	ID                   string
	TelegramLinked       bool
	NotificationsEnabled bool
}

type Entry struct {
	ID         string
	SlotID     string
	SeatsCount int
	Status     string
	// Position — место в очереди, начиная с 1 (для предложенных — 0).
	Position   int
	CreatedAt  time.Time
	NotifiedAt *time.Time
}

// OfferExpiresAt — до какого момента действует предложение (nil, если его нет).
func (e Entry) OfferExpiresAt() *time.Time {
	if e.Status != "notified" || e.NotifiedAt == nil {
		return nil
	}
	expires := e.NotifiedAt.Add(OfferTTL)
	return &expires
}

// MyEntry — запись в очереди для списка «Мои очереди» в профиле.
type MyEntry struct {
	Entry
	RouteName string
	StartAt   time.Time
}

type Status struct {
	Entry                *Entry
	TelegramLinked       bool
	NotificationsEnabled bool
}

type Repository interface {
	ClientBySessionTokenHash(ctx context.Context, tokenHash string) (Client, bool, error)
	// Join ставит клиента в очередь, проверяя заезд под блокировкой его строки — так же,
	// как это делает бронирование. Если клиент уже в очереди, возвращает его запись
	// (created=false).
	Join(ctx context.Context, clientID, slotID string, seats int, now time.Time) (entry Entry, created bool, err error)
	Leave(ctx context.Context, clientID, slotID string, now time.Time) error
	ActiveEntry(ctx context.Context, clientID, slotID string) (Entry, bool, error)
	// ActiveEntriesForClient — очереди клиента на заезды, которые ещё не начались, по
	// времени старта.
	ActiveEntriesForClient(ctx context.Context, clientID string, now time.Time) ([]MyEntry, error)
}

type Service struct {
	repo     Repository
	now      func() time.Time
	onJoined func()
}

// NewService: onJoined — сигнал рассыльщику после новой записи в очередь: место могло
// освободиться, пока человек вставал в очередь. nil — без сигнала.
func NewService(repo Repository, onJoined func()) *Service {
	if onJoined == nil {
		onJoined = func() {}
	}
	return &Service{repo: repo, now: time.Now, onJoined: onJoined}
}

func (s *Service) Status(ctx context.Context, token, slotID string) (Status, error) {
	if slotID == "" {
		return Status{}, ErrSlotNotFound
	}
	client, err := s.client(ctx, token)
	if err != nil {
		return Status{}, err
	}
	status := Status{TelegramLinked: client.TelegramLinked, NotificationsEnabled: client.NotificationsEnabled}
	entry, found, err := s.repo.ActiveEntry(ctx, client.ID, slotID)
	if err != nil {
		return Status{}, err
	}
	if found {
		status.Entry = &entry
	}
	return status, nil
}

// Mine — все очереди человека, в которых он сейчас стоит.
func (s *Service) Mine(ctx context.Context, token string) ([]MyEntry, error) {
	client, err := s.client(ctx, token)
	if err != nil {
		return nil, err
	}
	return s.repo.ActiveEntriesForClient(ctx, client.ID, s.now().UTC())
}

func (s *Service) Join(ctx context.Context, token, slotID string, seats int) (Entry, bool, error) {
	if slotID == "" || seats < 1 || seats > MaxSeats {
		return Entry{}, false, ErrInvalidRequest
	}
	client, err := s.client(ctx, token)
	if err != nil {
		return Entry{}, false, err
	}
	switch {
	case !client.TelegramLinked:
		return Entry{}, false, ErrTelegramRequired
	case !client.NotificationsEnabled:
		return Entry{}, false, ErrNotificationsDisabled
	}
	entry, created, err := s.repo.Join(ctx, client.ID, slotID, seats, s.now().UTC())
	if err != nil {
		return Entry{}, false, err
	}
	if created {
		s.onJoined()
	}
	return entry, created, nil
}

func (s *Service) Leave(ctx context.Context, token, slotID string) error {
	if slotID == "" {
		return ErrSlotNotFound
	}
	client, err := s.client(ctx, token)
	if err != nil {
		return err
	}
	return s.repo.Leave(ctx, client.ID, slotID, s.now().UTC())
}

func (s *Service) client(ctx context.Context, token string) (Client, error) {
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
