package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/service/waitlist"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Коды ошибок листа ожидания: по ним приложение показывает свои подсказки.
const (
	CodeTelegramRequired      = "telegram_required"
	CodeNotificationsDisabled = "notifications_disabled"
	CodeSeatsAvailable        = "seats_available"
	CodeWaitlistLimit         = "waitlist_limit"
)

type WaitlistHandler struct {
	service *waitlist.Service
	logger  *slog.Logger
}

func NewWaitlistHandler(service *waitlist.Service, logger *slog.Logger) *WaitlistHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WaitlistHandler{service: service, logger: logger}
}

type waitlistEntryDTO struct {
	ID             string     `json:"id"`
	Status         string     `json:"status"`
	SeatsCount     int        `json:"seats_count"`
	Position       int        `json:"position"`
	CreatedAt      time.Time  `json:"created_at"`
	OfferExpiresAt *time.Time `json:"offer_expires_at,omitempty"`
}

type waitlistStatusDTO struct {
	Entry                *waitlistEntryDTO `json:"entry"`
	TelegramLinked       bool              `json:"telegram_linked"`
	NotificationsEnabled bool              `json:"notifications_enabled"`
}

// Status — GET /slots/{slotID}/waitlist: стоит ли человек в очереди и может ли встать.
func (h *WaitlistHandler) Status(w http.ResponseWriter, r *http.Request) {
	token, slotID, ok := h.request(w, r)
	if !ok {
		return
	}
	status, err := h.service.Status(r.Context(), token, slotID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	dto := waitlistStatusDTO{TelegramLinked: status.TelegramLinked, NotificationsEnabled: status.NotificationsEnabled}
	if status.Entry != nil {
		dto.Entry = entryDTO(*status.Entry)
	}
	httpapi.WriteJSON(w, http.StatusOK, dto)
}

// Join — POST /slots/{slotID}/waitlist {"seats_count": N}. Повторный вызов возвращает
// уже существующую запись (200 вместо 201).
func (h *WaitlistHandler) Join(w http.ResponseWriter, r *http.Request) {
	token, slotID, ok := h.request(w, r)
	if !ok {
		return
	}
	var req struct {
		SeatsCount int `json:"seats_count"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}
	entry, created, err := h.service.Join(r.Context(), token, slotID, req.SeatsCount)
	if err != nil {
		h.writeError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpapi.WriteJSON(w, status, entryDTO(entry))
}

// Leave — DELETE /slots/{slotID}/waitlist. Идемпотентно: вне очереди — тоже 204.
func (h *WaitlistHandler) Leave(w http.ResponseWriter, r *http.Request) {
	token, slotID, ok := h.request(w, r)
	if !ok {
		return
	}
	if err := h.service.Leave(r.Context(), token, slotID); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WaitlistHandler) request(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return "", "", false
	}
	slotID, err := uuid.Parse(chi.URLParam(r, "slotID"))
	if err != nil {
		httpapi.WriteError(w, http.StatusNotFound, httpapi.CodeNotFound, "Заезд не найден.", nil)
		return "", "", false
	}
	return token, slotID.String(), true
}

func entryDTO(entry waitlist.Entry) *waitlistEntryDTO {
	return &waitlistEntryDTO{
		ID:             entry.ID,
		Status:         entry.Status,
		SeatsCount:     entry.SeatsCount,
		Position:       entry.Position,
		CreatedAt:      entry.CreatedAt,
		OfferExpiresAt: entry.OfferExpiresAt(),
	}
}

func (h *WaitlistHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, waitlist.ErrUnauthorized):
		httpapi.WriteError(w, http.StatusUnauthorized, httpapi.CodeUnauthorized, "Требуется авторизация.", nil)
	case errors.Is(err, waitlist.ErrInvalidRequest):
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, fmt.Sprintf("Укажите от 1 до %d мест.", waitlist.MaxSeats), nil)
	case errors.Is(err, waitlist.ErrSlotNotFound):
		httpapi.WriteError(w, http.StatusNotFound, httpapi.CodeNotFound, "Заезд не найден.", nil)
	case errors.Is(err, waitlist.ErrSlotCancelled):
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodeSlotCancelled, "Заезд отменён.", nil)
	case errors.Is(err, waitlist.ErrSlotStarted):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, httpapi.CodeSlotStarted, "Заезд уже начался.", nil)
	case errors.Is(err, waitlist.ErrAlreadyBooked):
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodeDoubleBooking, "Вы уже записаны на этот заезд.", nil)
	case errors.Is(err, waitlist.ErrSeatsAvailable):
		httpapi.WriteError(w, http.StatusConflict, CodeSeatsAvailable, "Места уже есть — можно записаться сразу.", nil)
	case errors.Is(err, waitlist.ErrTelegramRequired):
		httpapi.WriteError(w, http.StatusForbidden, CodeTelegramRequired,
			"О свободном месте мы сообщаем в Telegram. Войдите в приложение через Telegram, чтобы встать в лист ожидания.", nil)
	case errors.Is(err, waitlist.ErrNotificationsDisabled):
		httpapi.WriteError(w, http.StatusForbidden, CodeNotificationsDisabled,
			"Уведомления в Telegram отключены. Отправьте боту /notify и попробуйте снова.", nil)
	case errors.Is(err, waitlist.ErrTooManyEntries):
		httpapi.WriteError(w, http.StatusConflict, CodeWaitlistLimit, fmt.Sprintf("Можно стоять в очереди не больше чем на %d заездов.", waitlist.MaxActiveEntries), nil)
	default:
		h.logger.Error("waitlist request failed", "error", err)
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
	}
}
