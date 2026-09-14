package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	authapi "summer-school-2026/backend/internal/http/openapi/auth"
	"summer-school-2026/backend/internal/service/telegramlogin"
	"summer-school-2026/backend/internal/telegram"

	"github.com/google/uuid"
)

type TelegramHandler struct {
	service       *telegramlogin.Service
	webhookSecret string
	logger        *slog.Logger
}

func NewTelegramHandler(service *telegramlogin.Service, webhookSecret string, logger *slog.Logger) *TelegramHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TelegramHandler{service: service, webhookSecret: webhookSecret, logger: logger}
}

type telegramStartResponseDTO struct {
	PollToken   string    `json:"poll_token"`
	DeepLink    string    `json:"deep_link"`
	ConfirmCode string    `json:"confirm_code"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Start — POST /auth/telegram/start.
func (h *TelegramHandler) Start(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Start(r.Context())
	if err != nil {
		if errors.Is(err, telegramlogin.ErrNotConfigured) {
			httpapi.WriteError(w, http.StatusServiceUnavailable, httpapi.CodeInternalError, "Вход через Telegram сейчас недоступен.", nil)
			return
		}
		h.logger.Error("telegram login start failed", "error", err)
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, telegramStartResponseDTO{
		PollToken:   result.PollToken,
		DeepLink:    result.DeepLink,
		ConfirmCode: result.ConfirmCode,
		ExpiresAt:   result.ExpiresAt,
	})
}

type telegramPollResponseDTO struct {
	Status string `json:"status"`
	*verifyCodeResponseDTO
}

// Poll — POST /auth/telegram/poll. Секрет опроса передаётся в теле, а не в адресе:
// адреса попадают в логи прокси и сервера.
func (h *TelegramHandler) Poll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PollToken string `json:"poll_token"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil || req.PollToken == "" {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}
	result, err := h.service.Poll(r.Context(), req.PollToken)
	if err != nil {
		h.logger.Error("telegram login poll failed", "error", err)
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	if result.Status != telegramlogin.StatusConfirmed {
		httpapi.WriteJSON(w, http.StatusOK, telegramPollResponseDTO{Status: string(result.Status)})
		return
	}
	session := result.Session
	clientID, err := uuid.Parse(session.Client.ID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, telegramPollResponseDTO{
		Status: string(result.Status),
		verifyCodeResponseDTO: &verifyCodeResponseDTO{
			Token: session.Token,
			Tokens: tokenPairDTO{
				AccessToken:  session.Token,
				RefreshToken: session.RefreshToken,
				TokenType:    "Bearer",
				ExpiresIn:    session.AccessTTLSeconds,
			},
			Client: authapi.Client{
				Id:        clientID,
				Name:      session.Client.Name,
				Phone:     session.Client.Phone,
				CreatedAt: session.Client.CreatedAt,
			},
			IsNew: session.IsNew,
		},
	})
}

// Webhook — POST /telegram/webhook. Принимает обновления только с секретом, который
// Telegram присылает в заголовке: без проверки любой мог бы прислать поддельный
// «контакт» и подтвердить чужой вход.
func (h *TelegramHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	got := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if subtle.ConstantTimeCompare([]byte(got), []byte(h.webhookSecret)) != 1 {
		httpapi.WriteError(w, http.StatusUnauthorized, httpapi.CodeUnauthorized, "Требуется авторизация.", nil)
		return
	}
	var update telegram.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		// 200, а не 400: на ошибку Telegram повторял бы это же обновление снова и снова.
		h.logger.Warn("telegram webhook: undecodable update")
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := h.service.HandleUpdate(r.Context(), update); err != nil {
		h.logger.Error("telegram webhook: handle update failed", "error", err)
	}
	w.WriteHeader(http.StatusOK)
}
