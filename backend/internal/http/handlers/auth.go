package handlers

import (
	"errors"
	"net/http"

	httpapi "summer-school-2026/backend/internal/http"
	authapi "summer-school-2026/backend/internal/http/openapi/auth"
	"summer-school-2026/backend/internal/service/auth"

	"github.com/google/uuid"
)

type AuthHandler struct {
	service *auth.Service
	// dev true → разрешено возвращать OTP-код в ответе (для ручной проверки без SMS).
	// В production код никогда не возвращается клиенту (флаг берётся из APP_ENV).
	dev bool
}

func NewAuthHandler(service *auth.Service, dev bool) *AuthHandler {
	return &AuthHandler{service: service, dev: dev}
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token, err := httpapi.BearerToken(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusUnauthorized, httpapi.CodeUnauthorized, "Требуется авторизация. Передайте действительный токен в заголовке Authorization.", nil)
		return
	}
	if err := h.service.Logout(r.Context(), token); err != nil {
		httpapi.WriteError(w, http.StatusUnauthorized, httpapi.CodeUnauthorized, "Требуется авторизация. Передайте действительный токен в заголовке Authorization.", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) RequestAuthCode(w http.ResponseWriter, r *http.Request) {
	var req authapi.RequestCodeRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}

	result, err := h.service.RequestCode(r.Context(), req.Phone)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	// OTP-код возвращается ТОЛЬКО в dev (APP_ENV != "production").
	// В production клиент получает только TTL/resend_after, без самого кода —
	// код доставляется отдельным каналом (SMS).
	response := authapi.RequestCodeResponse{
		TtlSeconds:         result.TTLSeconds,
		ResendAfterSeconds: result.ResendAfterSeconds,
	}
	if h.dev {
		response.Code = &result.Code
	}

	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) VerifyAuthCode(w http.ResponseWriter, r *http.Request) {
	var req authapi.VerifyCodeRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}

	result, err := h.service.VerifyCode(r.Context(), req.Phone, req.Code)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	clientID, err := uuid.Parse(result.Client.ID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, authapi.VerifyCodeResponse{
		Token: result.Token,
		Client: authapi.Client{
			Id:        clientID,
			Name:      result.Client.Name,
			Phone:     result.Client.Phone,
			CreatedAt: result.Client.CreatedAt,
		},
		IsNew: result.IsNew,
	})
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidPhone):
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
	case errors.Is(err, auth.ErrInvalidCode):
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeInvalidCode, "Неверный или истёкший код подтверждения.", nil)
	case errors.Is(err, auth.ErrTooManyRequests):
		httpapi.WriteError(w, http.StatusTooManyRequests, httpapi.CodeTooManyRequests, "Слишком много запросов. Повторите попытку позже.", nil)
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
	}
}
