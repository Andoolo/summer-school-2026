package handlers

import (
	"errors"
	"net/http"

	httpapi "summer-school-2026/backend/internal/http"
	profileapi "summer-school-2026/backend/internal/http/openapi/profile"
	"summer-school-2026/backend/internal/service/profile"

	"github.com/google/uuid"
)

type ProfileHandler struct {
	service *profile.Service
}

func NewProfileHandler(service *profile.Service) *ProfileHandler {
	return &ProfileHandler{service: service}
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	client, err := h.service.Current(r.Context(), token)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeProfileClient(w, client)
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	var req profileapi.UpdateProfileRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}
	client, err := h.service.UpdateName(r.Context(), token, req.Name)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeProfileClient(w, client)
}

func (h *ProfileHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteAccount(r.Context(), token); err != nil {
		writeProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProfileHandler) RequestPhoneChangeCode(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	var req profileapi.ChangePhoneRequestCodeRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}
	result, err := h.service.RequestPhoneChangeCode(r.Context(), token, req.NewPhone)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, profileapi.RequestCodeResponse{TtlSeconds: result.TTLSeconds, ResendAfterSeconds: result.ResendAfterSeconds, Code: &result.Code})
}

func (h *ProfileHandler) ConfirmPhoneChange(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	var req profileapi.ChangePhoneConfirmRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}
	client, err := h.service.ConfirmPhoneChange(r.Context(), token, req.NewPhone, req.Code)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeProfileClient(w, client)
}

func bearerOrUnauthorized(w http.ResponseWriter, r *http.Request) (string, bool) {
	token, err := httpapi.BearerToken(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusUnauthorized, httpapi.CodeUnauthorized, "Требуется авторизация. Передайте действительный токен в заголовке Authorization.", nil)
		return "", false
	}
	return token, true
}

func writeProfileClient(w http.ResponseWriter, client profile.Client) {
	clientID, err := uuid.Parse(client.ID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, profileapi.Client{Id: clientID, Name: client.Name, Phone: client.Phone, CreatedAt: client.CreatedAt})
}

func writeProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profile.ErrUnauthorized):
		httpapi.WriteError(w, http.StatusUnauthorized, httpapi.CodeUnauthorized, "Требуется авторизация. Передайте действительный токен в заголовке Authorization.", nil)
	case errors.Is(err, profile.ErrInvalidName), errors.Is(err, profile.ErrInvalidPhone), errors.Is(err, profile.ErrInvalidCode):
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
	case errors.Is(err, profile.ErrPhoneConflict):
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodePhoneConflict, "Указанный телефон уже используется другим клиентом.", nil)
	case errors.Is(err, profile.ErrTooManyRequests):
		httpapi.WriteError(w, http.StatusTooManyRequests, httpapi.CodeTooManyRequests, "Слишком много запросов. Повторите попытку позже.", nil)
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
	}
}
