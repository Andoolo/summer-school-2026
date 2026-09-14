package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	authapi "summer-school-2026/backend/internal/http/openapi/auth"
	"summer-school-2026/backend/internal/service/auth"

	"github.com/google/uuid"
)

// demoLoginResponseDTO — ответ гостевого входа: тот же формат, что у verify-code, плюс
// признак гостя и срок его жизни, чтобы приложение сразу показало «демо-режим».
type demoLoginResponseDTO struct {
	verifyCodeResponseDTO
	IsDemo        bool      `json:"is_demo"`
	DemoExpiresAt time.Time `json:"demo_expires_at"`
}

// DemoLoginHandler — POST /auth/demo: создаёт гостя и выдаёт ему сессию.
func DemoLoginHandler(service *auth.DemoService, logger *slog.Logger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Login(r.Context())
		if err != nil {
			if errors.Is(err, auth.ErrDemoUnavailable) {
				logger.Warn("demo login capacity reached")
				httpapi.WriteError(w, http.StatusServiceUnavailable, httpapi.CodeTooManyRequests, "Демо-режим сейчас переполнен. Попробуйте позже.", nil)
				return
			}
			logger.Error("demo login failed", "error", err)
			httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
			return
		}
		clientID, err := uuid.Parse(result.Client.ID)
		if err != nil || result.Client.DemoExpiresAt == nil {
			httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
			return
		}
		httpapi.WriteJSON(w, http.StatusCreated, demoLoginResponseDTO{
			verifyCodeResponseDTO: verifyCodeResponseDTO{
				Token: result.Token,
				Tokens: tokenPairDTO{
					AccessToken:  result.Token,
					RefreshToken: result.RefreshToken,
					TokenType:    "Bearer",
					ExpiresIn:    result.AccessTTLSeconds,
				},
				Client: authapi.Client{
					Id:        clientID,
					Name:      result.Client.Name,
					Phone:     result.Client.Phone,
					CreatedAt: result.Client.CreatedAt,
				},
				IsNew: true,
			},
			IsDemo:        true,
			DemoExpiresAt: *result.Client.DemoExpiresAt,
		})
	}
}

// AuthMethods — какие способы входа сейчас доступны. Приложение показывает только их:
// так форма входа по SMS не появляется на сайте, где SMS не отправляются, а кнопка
// Telegram — пока бот не настроен.
type AuthMethods struct {
	// SMS — вход по коду. Коды реально не отправляются, поэтому только в dev, где код
	// приходит в ответе API.
	SMS  bool
	Demo bool
	// TelegramBotUsername возвращает имя бота без @; пусто — вход через Telegram
	// выключен. Функция, а не строка: бот подключается в фоне уже после старта.
	TelegramBotUsername func() string
}

type authMethodsDTO struct {
	SMS      bool               `json:"sms"`
	Demo     bool               `json:"demo"`
	Telegram *telegramMethodDTO `json:"telegram"`
}

type telegramMethodDTO struct {
	BotUsername string `json:"bot_username"`
}

// AuthMethodsHandler — GET /auth/methods.
func AuthMethodsHandler(methods AuthMethods) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := authMethodsDTO{SMS: methods.SMS, Demo: methods.Demo}
		if methods.TelegramBotUsername != nil {
			if username := methods.TelegramBotUsername(); username != "" {
				body.Telegram = &telegramMethodDTO{BotUsername: username}
			}
		}
		httpapi.WriteJSON(w, http.StatusOK, body)
	}
}
