package handlers

import (
	"crypto/subtle"
	"net/http"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Границы правдоподобия для времени круга. Круг быстрее пяти секунд или медленнее
// десяти минут — это почти наверняка опечатка маршала (лишний ноль, секунды вместо
// миллисекунд), а не результат заезда. Пускать такое в таблицу рекордов нельзя:
// один неверный ноль навсегда сделает «рекордсменом» того, кто им не был.
const (
	minLapTimeMs   = 5_000
	maxLapTimeMs   = 600_000
	maxLapsPerRace = 100
)

type lapResultsRequestDTO struct {
	LapTimesMs []int `json:"lap_times_ms"`
}

type lapResultsResponseDTO struct {
	BookingID     string `json:"booking_id"`
	RacerName     string `json:"racer_name"`
	RouteName     string `json:"route_name"`
	Laps          int    `json:"laps"`
	BestLapMs     int    `json:"best_lap_ms"`
	IsTrackRecord bool   `json:"is_track_record"`
}

// MarshalHandler — внесение времён кругов маршалом (картинг-фича F6).
//
// Отдельный обработчик, а не часть SlotHandler: у него своя модель доступа —
// не клиентский токен, а общий секрет маршала.
type MarshalHandler struct {
	repo  *postgres.SlotRepository
	token string
	now   func() time.Time
}

func NewMarshalHandler(repo *postgres.SlotRepository, token string) *MarshalHandler {
	return &MarshalHandler{repo: repo, token: token, now: time.Now}
}

// SubmitLapResults — POST /bookings/{bookingID}/lap-results.
//
// Полностью заменяет времена кругов брони: маршал вносит результаты заезда целиком
// и может переотправить их, исправив опечатку.
func (h *MarshalHandler) SubmitLapResults(w http.ResponseWriter, r *http.Request) {
	// Сравнение постоянного времени: обычное сравнение строк завершается на первом
	// несовпавшем байте, и по времени ответа токен можно подобрать посимвольно.
	provided := r.Header.Get("X-Marshal-Token")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) != 1 {
		httpapi.WriteError(w, http.StatusForbidden, httpapi.CodeForbidden, "Недостаточно прав для выполнения операции.", nil)
		return
	}

	bookingID, err := uuid.Parse(chi.URLParam(r, "bookingID"))
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}

	var req lapResultsRequestDTO
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}
	if len(req.LapTimesMs) == 0 || len(req.LapTimesMs) > maxLapsPerRace {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Укажите от одного до 100 кругов.", nil)
		return
	}
	for _, ms := range req.LapTimesMs {
		if ms < minLapTimeMs || ms > maxLapTimeMs {
			httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Время круга должно быть от 5 секунд до 10 минут.", nil)
			return
		}
	}

	target, found, err := h.repo.BookingForLapEntry(r.Context(), bookingID.String())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	if !found {
		httpapi.WriteError(w, http.StatusNotFound, httpapi.CodeNotFound, "Запрашиваемый ресурс не найден.", nil)
		return
	}
	if target.BookingState != "active" {
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodeAlreadyCancelled, "Запись отменена — результаты для неё не вносятся.", nil)
		return
	}
	// Заезд, который ещё не начался, не мог дать кругов. Без этой проверки опечатка
	// в выборе брони наградила бы рекордом участника будущего заезда.
	if !target.SlotStartAt.Before(h.now()) {
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodeBadRequest, "Заезд ещё не состоялся.", nil)
		return
	}

	previousBest, err := h.repo.ReplaceLapResults(r.Context(), target.BookingID, target.RouteID, req.LapTimesMs)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}

	best := req.LapTimesMs[0]
	for _, ms := range req.LapTimesMs[1:] {
		if ms < best {
			best = ms
		}
	}

	httpapi.WriteJSON(w, http.StatusOK, lapResultsResponseDTO{
		BookingID:     target.BookingID,
		RacerName:     target.ClientName,
		RouteName:     target.RouteName,
		Laps:          len(req.LapTimesMs),
		BestLapMs:     best,
		IsTrackRecord: previousBest == nil || best < *previousBest,
	})
}
