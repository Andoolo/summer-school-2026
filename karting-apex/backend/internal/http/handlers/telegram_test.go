package handlers_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/http/handlers"
	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/service/botrouter"
	"summer-school-2026/backend/internal/service/telegramlogin"
)

type countingBot struct{ sent int }

func (b *countingBot) SendMessage(context.Context, int64, string, any) error {
	b.sent++
	return nil
}

type noopTelegramRepo struct{}

func (noopTelegramRepo) CreateLoginRequest(context.Context, string, string, string, time.Time, time.Time) error {
	return nil
}
func (noopTelegramRepo) AttachChat(context.Context, string, int64, time.Time) (telegramlogin.Request, bool, error) {
	return telegramlogin.Request{}, false, nil
}
func (noopTelegramRepo) ConfirmLatestForChat(context.Context, int64, string, string, time.Time) (bool, error) {
	return false, nil
}
func (noopTelegramRepo) ConsumeConfirmed(context.Context, string, time.Time) (telegramlogin.Request, bool, error) {
	return telegramlogin.Request{}, false, nil
}
func (noopTelegramRepo) RequestByPollHash(context.Context, string) (telegramlogin.Request, bool, error) {
	return telegramlogin.Request{}, false, nil
}
func (noopTelegramRepo) LinkChat(context.Context, string, int64) error { return nil }
func (noopTelegramRepo) SetNotifications(context.Context, int64, bool) (bool, error) {
	return false, nil
}

type noSessions struct{}

func (noSessions) LoginByVerifiedPhone(context.Context, string, string) (auth.VerifyCodeResult, error) {
	return auth.VerifyCodeResult{}, nil
}

func TestTelegramWebhookRequiresSecret(t *testing.T) {
	bot := &countingBot{}
	service := telegramlogin.NewService(noopTelegramRepo{}, noSessions{}, bot, func() string { return "apex_login_bot" }, nil)
	updates := botrouter.New(botrouter.Config{Login: service, Notifications: noopTelegramRepo{}, Bot: bot})
	handler := handlers.NewTelegramHandler(service, updates, "correct-secret", slog.Default())
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{TelegramWebhook: handler.Webhook})

	update := `{"update_id":1,"message":{"message_id":1,"from":{"id":7,"first_name":"A"},"chat":{"id":7,"type":"private"},"text":"привет"}}`
	send := func(secret string) int {
		req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(update))
		if secret != "" {
			req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := send(""); code != http.StatusUnauthorized {
		t.Fatalf("without secret status = %d, want 401", code)
	}
	if code := send("wrong-secret"); code != http.StatusUnauthorized {
		t.Fatalf("wrong secret status = %d, want 401", code)
	}
	if bot.sent != 0 {
		t.Fatal("rejected updates must not reach the bot")
	}
	if code := send("correct-secret"); code != http.StatusOK {
		t.Fatalf("correct secret status = %d, want 200", code)
	}
	if bot.sent != 1 {
		t.Fatalf("accepted update must be handled, sent = %d", bot.sent)
	}
}

func TestTelegramStartUnavailableWithoutBot(t *testing.T) {
	service := telegramlogin.NewService(noopTelegramRepo{}, noSessions{}, &countingBot{}, func() string { return "" }, nil)
	handler := handlers.NewTelegramHandler(service, nil, "secret", slog.Default())
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{TelegramStart: handler.Start, TelegramPoll: handler.Poll})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/telegram/start", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("start without bot status = %d, want 503", rec.Code)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/telegram/poll", strings.NewReader(`{"poll_token":"nope"}`)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"expired"`) {
		t.Fatalf("poll unknown token = %d %s, want expired", rec.Code, rec.Body.String())
	}
}
