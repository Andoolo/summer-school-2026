// Package telegramlogin — вход через Telegram без кода.
//
// Порядок:
//  1. Приложение вызывает Start и получает ссылку на бота, код сверки и секрет опроса.
//  2. Человек открывает бота по ссылке; бот показывает тот же код сверки и кнопку
//     «Поделиться номером».
//  3. Человек делится номером; бот проверяет, что это номер самого собеседника.
//  4. Приложение опрашивает Poll по своему секрету и получает сессию.
//
// Номер подтверждает сам Telegram: контакт с user_id, равным отправителю, — это номер,
// привязанный к аккаунту человека.
package telegramlogin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"strings"
	"time"

	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/telegram"
)

var (
	ErrNotConfigured = errors.New("telegram login is not configured")
	phonePattern     = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	startPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)
)

const (
	requestTTL = 10 * time.Minute
	// Код сверки: без похожих друг на друга символов (0/O, 1/I), чтобы его было легко
	// сравнить глазами.
	codeAlphabet = "ACEFHJKMNPRTUVWXY34679"
	codeLength   = 4
)

// Status — состояние запроса входа для приложения.
type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusExpired   Status = "expired"
)

// Request — хранимый запрос входа.
type Request struct {
	ConfirmCode string
	Status      string
	ExpiresAt   time.Time
	Phone       string
	FirstName   string
}

type Repository interface {
	CreateLoginRequest(ctx context.Context, startHash, pollHash, code string, now, expiresAt time.Time) error
	// AttachChat привязывает чат к незавершённому запросу по токену из ссылки. Запрос,
	// уже привязанный к другому чату, не перепривязывается.
	AttachChat(ctx context.Context, startHash string, chatID int64, now time.Time) (Request, bool, error)
	// ConfirmLatestForChat подтверждает последний незавершённый запрос чата номером.
	ConfirmLatestForChat(ctx context.Context, chatID int64, phone, firstName string, now time.Time) (bool, error)
	// ConsumeConfirmed атомарно гасит подтверждённый запрос и возвращает его: сессия
	// выдаётся ровно один раз, даже при параллельных опросах.
	ConsumeConfirmed(ctx context.Context, pollHash string, now time.Time) (Request, bool, error)
	RequestByPollHash(ctx context.Context, pollHash string) (Request, bool, error)
}

// Sessions — выдача сессии по номеру, уже подтверждённому Telegram.
type Sessions interface {
	LoginByVerifiedPhone(ctx context.Context, phone, name string) (auth.VerifyCodeResult, error)
}

// Bot — отправка сообщений.
type Bot interface {
	SendMessage(ctx context.Context, chatID int64, text string, markup any) error
}

type StartResult struct {
	PollToken   string
	DeepLink    string
	ConfirmCode string
	ExpiresAt   time.Time
}

type PollResult struct {
	Status  Status
	Session auth.VerifyCodeResult
}

type Service struct {
	repo     Repository
	sessions Sessions
	bot      Bot
	logger   *slog.Logger
	now      func() time.Time
	username func() string
}

// NewService создаёт сервис. username возвращает имя бота или пустую строку, пока бот
// не подключён (имя узнаётся у Telegram в фоне после старта сервера).
func NewService(repo Repository, sessions Sessions, bot Bot, username func() string, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, sessions: sessions, bot: bot, username: username, logger: logger, now: time.Now}
}

func (s *Service) Start(ctx context.Context) (StartResult, error) {
	username := s.username()
	if username == "" {
		return StartResult{}, ErrNotConfigured
	}
	startToken, err := randomToken(24)
	if err != nil {
		return StartResult{}, err
	}
	pollToken, err := randomToken(32)
	if err != nil {
		return StartResult{}, err
	}
	code, err := randomCode()
	if err != nil {
		return StartResult{}, err
	}
	now := s.now().UTC()
	expiresAt := now.Add(requestTTL)
	if err := s.repo.CreateLoginRequest(ctx, auth.HashToken(startToken), auth.HashToken(pollToken), code, now, expiresAt); err != nil {
		return StartResult{}, err
	}
	return StartResult{
		PollToken:   pollToken,
		DeepLink:    "https://t.me/" + username + "?start=" + startToken,
		ConfirmCode: code,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *Service) Poll(ctx context.Context, pollToken string) (PollResult, error) {
	if pollToken == "" {
		return PollResult{Status: StatusExpired}, nil
	}
	now := s.now().UTC()
	pollHash := auth.HashToken(pollToken)

	request, ok, err := s.repo.ConsumeConfirmed(ctx, pollHash, now)
	if err != nil {
		return PollResult{}, err
	}
	if ok {
		session, err := s.sessions.LoginByVerifiedPhone(ctx, request.Phone, request.FirstName)
		if err != nil {
			return PollResult{}, err
		}
		return PollResult{Status: StatusConfirmed, Session: session}, nil
	}

	request, ok, err = s.repo.RequestByPollHash(ctx, pollHash)
	if err != nil {
		return PollResult{}, err
	}
	if !ok || request.Status != "pending" || !now.Before(request.ExpiresAt) {
		return PollResult{Status: StatusExpired}, nil
	}
	return PollResult{Status: StatusPending}, nil
}

const (
	textHelp = "Это бот для входа в приложение «Апекс».\n\n" +
		"Откройте приложение и нажмите «Войти через Telegram» — бот пришлёт кнопку для входа."
	textLinkExpired = "Ссылка для входа устарела или уже использована.\n\n" +
		"Вернитесь в приложение и нажмите «Войти через Telegram» ещё раз."
	textForeignContact = "Для входа нужен ваш собственный номер. Нажмите кнопку «Поделиться номером» ниже."
	textNoRequest      = "Запрос на вход не найден или устарел. Начните вход в приложении заново."
	textBadPhone       = "Не получилось войти с этим номером. Начните вход в приложении заново."
	textDone           = "Готово! Вернитесь в приложение — вход выполнится автоматически."
	shareButtonText    = "📱 Поделиться номером"
)

func textConfirm(code string) string {
	return "Вход в «Апекс»\n\n" +
		"Код сверки: " + code + "\n" +
		"Убедитесь, что в приложении показан этот же код.\n\n" +
		"Если вход начали не вы — ничего не нажимайте: кто-то пытается войти под вашим номером.\n\n" +
		"Чтобы войти, нажмите «Поделиться номером» ниже."
}

// HandleUpdate обрабатывает сообщение из вебхука. Ошибки отправки сообщения только
// логируются: вебхук должен ответить Telegram быстро, иначе тот начнёт повторы.
func (s *Service) HandleUpdate(ctx context.Context, update telegram.Update) error {
	msg := update.Message
	if msg == nil || msg.From == nil || msg.From.IsBot || msg.Chat.Type != "private" {
		return nil
	}
	now := s.now().UTC()

	switch {
	case msg.Contact != nil:
		return s.handleContact(ctx, msg, now)
	case strings.HasPrefix(msg.Text, "/start"):
		return s.handleStart(ctx, msg, now)
	default:
		s.send(ctx, msg.Chat.ID, textHelp, nil)
		return nil
	}
}

func (s *Service) handleStart(ctx context.Context, msg *telegram.Message, now time.Time) error {
	param := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/start"))
	if !startPattern.MatchString(param) {
		s.send(ctx, msg.Chat.ID, textHelp, nil)
		return nil
	}
	request, ok, err := s.repo.AttachChat(ctx, auth.HashToken(param), msg.Chat.ID, now)
	if err != nil {
		return err
	}
	if !ok {
		s.send(ctx, msg.Chat.ID, textLinkExpired, telegram.RemoveKeyboard{RemoveKeyboard: true})
		return nil
	}
	s.send(ctx, msg.Chat.ID, textConfirm(request.ConfirmCode), telegram.ReplyKeyboard{
		Keyboard:        [][]telegram.KeyboardButton{{{Text: shareButtonText, RequestContact: true}}},
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
	})
	return nil
}

func (s *Service) handleContact(ctx context.Context, msg *telegram.Message, now time.Time) error {
	// Контакт можно переслать чужой. Номер подтверждён Telegram только тогда, когда
	// контакт принадлежит самому отправителю.
	if msg.Contact.UserID == 0 || msg.Contact.UserID != msg.From.ID {
		s.send(ctx, msg.Chat.ID, textForeignContact, nil)
		return nil
	}
	phone, ok := NormalizePhone(msg.Contact.PhoneNumber)
	if !ok {
		s.send(ctx, msg.Chat.ID, textBadPhone, telegram.RemoveKeyboard{RemoveKeyboard: true})
		return nil
	}
	name := strings.TrimSpace(msg.Contact.FirstName)
	if name == "" {
		name = strings.TrimSpace(msg.From.FirstName)
	}
	confirmed, err := s.repo.ConfirmLatestForChat(ctx, msg.Chat.ID, phone, truncateRunes(name, 100), now)
	if err != nil {
		return err
	}
	if !confirmed {
		s.send(ctx, msg.Chat.ID, textNoRequest, telegram.RemoveKeyboard{RemoveKeyboard: true})
		return nil
	}
	s.logger.Info("telegram login confirmed", "phone", auth.MaskPhone(phone))
	s.send(ctx, msg.Chat.ID, textDone, telegram.RemoveKeyboard{RemoveKeyboard: true})
	return nil
}

func (s *Service) send(ctx context.Context, chatID int64, text string, markup any) {
	if err := s.bot.SendMessage(ctx, chatID, text, markup); err != nil {
		s.logger.Error("telegram send message failed", "error", err)
	}
}

// NormalizePhone приводит номер из Telegram к E.164. Telegram присылает цифры без «+»
// (иногда с «+»); номера гостей (+7000…) сюда попасть не должны.
func NormalizePhone(raw string) (string, bool) {
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	phone := "+" + digits.String()
	if !phonePattern.MatchString(phone) || auth.IsDemoPhone(phone) {
		return "", false
	}
	return phone, true
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate telegram login token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomCode() (string, error) {
	var code strings.Builder
	for i := 0; i < codeLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeAlphabet))))
		if err != nil {
			return "", fmt.Errorf("generate confirm code: %w", err)
		}
		code.WriteByte(codeAlphabet[n.Int64()])
	}
	return code.String(), nil
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}
