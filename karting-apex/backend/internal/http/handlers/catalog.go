package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	httpapi "summer-school-2026/backend/internal/http"
	instructorsapi "summer-school-2026/backend/internal/http/openapi/instructors"
	slotsapi "summer-school-2026/backend/internal/http/openapi/slots"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/track"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SlotHandler struct {
	repo *postgres.SlotRepository
}

func NewSlotHandler(repo *postgres.SlotRepository) *SlotHandler {
	return &SlotHandler{repo: repo}
}

// leaderboardEntryDTO — строка ответа рекордов трассы.
type leaderboardEntryDTO struct {
	Position  int    `json:"position"`
	Name      string `json:"name"`
	BestLapMs int    `json:"best_lap_ms"`
	Laps      int    `json:"laps"`
}

type leaderboardResponseDTO struct {
	RouteID   string                `json:"route_id"`
	RouteName string                `json:"route_name"`
	Entries   []leaderboardEntryDTO `json:"entries"`
}

// Leaderboard — GET /routes/{routeID}/leaderboard: топ лучших кругов по трассе (картинг-фича).
// Маршрут регистрируется вручную (RouterOptions.RouteLeaderboard) — в OpenAPI-транспорте его нет.
func (h *SlotHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	routeID, err := uuid.Parse(chi.URLParam(r, "routeID"))
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}

	routeName, entries, found, err := h.repo.Leaderboard(r.Context(), routeID.String(), 10)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	if !found {
		httpapi.WriteError(w, http.StatusNotFound, httpapi.CodeNotFound, "Запрашиваемый ресурс не найден.", nil)
		return
	}

	response := leaderboardResponseDTO{RouteID: routeID.String(), RouteName: routeName, Entries: make([]leaderboardEntryDTO, 0, len(entries))}
	for i, e := range entries {
		response.Entries = append(response.Entries, leaderboardEntryDTO{
			Position:  i + 1,
			Name:      e.ClientName,
			BestLapMs: e.BestLapMs,
			Laps:      e.Laps,
		})
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

// trackRecordDTO — действующий рекорд круга на трассе.
type trackRecordDTO struct {
	Name  string `json:"name"`
	LapMs int    `json:"lap_ms"`
}

// trackPassportDTO — карточка трассы (F5).
//
// Поля length_m / corners / direction / main_straight_m считаются из geometry на
// лету (internal/track), а не берутся из БД: так витрина не может разойтись с
// контуром, который рисуется на схеме. Остальное — «паспорт» из routes.
type trackPassportDTO struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	CapacityCap int         `json:"capacity_cap"`
	DurationMin int         `json:"duration_min"`
	Geometry    [][]float64 `json:"geometry"`

	LengthM       int    `json:"length_m"`
	Corners       int    `json:"corners"`
	Direction     string `json:"direction"`
	MainStraightM int    `json:"main_straight_m"`

	Surface     *string  `json:"surface,omitempty"`
	WidthM      *float64 `json:"width_m,omitempty"`
	ElevationM  *float64 `json:"elevation_m,omitempty"`
	OpenedYear  *int     `json:"opened_year,omitempty"`
	KartModel   *string  `json:"kart_model,omitempty"`
	KartPowerHP *int     `json:"kart_power_hp,omitempty"`

	Record *trackRecordDTO `json:"record,omitempty"`
}

// TrackPassport — GET /routes/{routeID}: карточка трассы (картинг-фича F5).
// Маршрут регистрируется вручную (RouterOptions.RoutePassport) — в OpenAPI-транспорте его нет.
func (h *SlotHandler) TrackPassport(w http.ResponseWriter, r *http.Request) {
	routeID, err := uuid.Parse(chi.URLParam(r, "routeID"))
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}

	p, found, err := h.repo.RoutePassportByID(r.Context(), routeID.String())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	if !found {
		httpapi.WriteError(w, http.StatusNotFound, httpapi.CodeNotFound, "Запрашиваемый ресурс не найден.", nil)
		return
	}

	// Геометрия может быть пустой или лежать строкой-полилинией (см. slotBase) —
	// в этом случае считать по ней нечего, отдаём карточку без выведенных метрик.
	var coords [][]float64
	if len(p.Geometry) > 0 && !isJSONString(p.Geometry) {
		if err := json.Unmarshal(p.Geometry, &coords); err != nil {
			httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
			return
		}
	}

	points := make([]track.Point, 0, len(coords))
	for _, c := range coords {
		if len(c) >= 2 {
			points = append(points, track.Point{Lat: c[0], Lng: c[1]})
		}
	}
	stats := track.Compute(points)

	dto := trackPassportDTO{
		ID:            p.ID,
		Name:          p.Name,
		Type:          p.Type,
		CapacityCap:   p.CapacityCap,
		DurationMin:   p.DurationMin,
		Geometry:      coords,
		LengthM:       stats.LengthM,
		Corners:       stats.Corners,
		Direction:     string(stats.Direction),
		MainStraightM: stats.MainStraightM,
		Surface:       p.Surface,
		WidthM:        p.WidthM,
		ElevationM:    p.ElevationM,
		OpenedYear:    p.OpenedYear,
		KartModel:     p.KartModel,
		KartPowerHP:   p.KartPowerHP,
	}
	if p.RecordHolder != nil && p.RecordLapMs != nil {
		dto.Record = &trackRecordDTO{Name: *p.RecordHolder, LapMs: *p.RecordLapMs}
	}
	if dto.Geometry == nil {
		dto.Geometry = [][]float64{}
	}

	httpapi.WriteJSON(w, http.StatusOK, dto)
}

func (h *SlotHandler) ListSlots(w http.ResponseWriter, r *http.Request, params slotsapi.ListSlotsParams) {
	limit, offset, ok := pagination(w, params.Limit, params.Offset)
	if !ok {
		return
	}
	filters := postgres.SlotFilters{Limit: limit, Offset: offset}
	filters.DateFrom = params.DateFrom
	filters.DateTo = params.DateTo
	if params.RouteType != nil {
		for _, routeType := range *params.RouteType {
			if routeType != string(slotsapi.RouteTypeNovice) && routeType != string(slotsapi.RouteTypeExperienced) {
				httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
				return
			}
			filters.RouteTypes = append(filters.RouteTypes, routeType)
		}
	}
	if params.InstructorId != nil {
		for _, id := range *params.InstructorId {
			filters.InstructorIDs = append(filters.InstructorIDs, id.String())
		}
	}
	if params.OnlyAvailable != nil {
		filters.OnlyAvailable = *params.OnlyAvailable
	}

	list, err := h.repo.List(r.Context(), filters)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}

	items := make([]slotsapi.SlotSummary, 0, len(list.Items))
	for _, slot := range list.Items {
		mapped, err := slotSummary(slot)
		if err != nil {
			httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
			return
		}
		items = append(items, mapped)
	}
	httpapi.WriteJSON(w, http.StatusOK, slotsapi.SlotListResponse{Items: items, Meta: slotsapi.PaginationMeta{Limit: limit, Offset: offset, Total: list.Total}})
}

func (h *SlotHandler) GetSlot(w http.ResponseWriter, r *http.Request, slotID slotsapi.SlotIdParam) {
	slot, found, err := h.repo.GetByID(r.Context(), slotID.String())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	if !found {
		httpapi.WriteError(w, http.StatusNotFound, httpapi.CodeNotFound, "Запрашиваемый ресурс не найден.", nil)
		return
	}
	mapped, err := fullSlot(slot)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, mapped)
}

type InstructorHandler struct {
	repo *postgres.InstructorRepository
}

func NewInstructorHandler(repo *postgres.InstructorRepository) *InstructorHandler {
	return &InstructorHandler{repo: repo}
}

func (h *InstructorHandler) ListInstructors(w http.ResponseWriter, r *http.Request, params instructorsapi.ListInstructorsParams) {
	limit, offset, ok := pagination(w, params.Limit, params.Offset)
	if !ok {
		return
	}
	list, err := h.repo.List(r.Context(), limit, offset)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}

	items := make([]instructorsapi.Instructor, 0, len(list.Items))
	for _, instructor := range list.Items {
		id, err := uuid.Parse(instructor.ID)
		if err != nil {
			httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
			return
		}
		items = append(items, instructorsapi.Instructor{Id: id, Name: instructor.Name})
	}
	httpapi.WriteJSON(w, http.StatusOK, instructorsapi.InstructorListResponse{Items: items, Meta: instructorsapi.PaginationMeta{Limit: limit, Offset: offset, Total: list.Total}})
}

func pagination(w http.ResponseWriter, limitParam, offsetParam *int) (int, int, bool) {
	limit := 20
	if limitParam != nil {
		limit = *limitParam
	}
	offset := 0
	if offsetParam != nil {
		offset = *offsetParam
	}
	if limit < 1 || limit > 100 || offset < 0 {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return 0, 0, false
	}
	return limit, offset, true
}

func slotSummary(slot postgres.Slot) (slotsapi.SlotSummary, error) {
	base, err := slotBase(slot)
	if err != nil {
		return slotsapi.SlotSummary{}, err
	}
	return slotsapi.SlotSummary{
		Id:               base.id,
		Route:            base.route,
		Instructor:       base.instructor,
		StartAt:          slot.StartAt,
		TotalSeats:       slot.TotalSeats,
		FreeSeats:        slot.FreeSeats,
		FreeRentalBoards: slot.FreeRentalBoards,
		Price:            slot.Price,
		RentalPrice:      slot.RentalPrice,
		Status:           slotsapi.SlotStatus(slot.Status),
	}, nil
}

func fullSlot(slot postgres.Slot) (slotsapi.Slot, error) {
	base, err := slotBase(slot)
	if err != nil {
		return slotsapi.Slot{}, err
	}
	return slotsapi.Slot{
		Id:               base.id,
		Route:            base.route,
		Instructor:       base.instructor,
		StartAt:          slot.StartAt,
		TotalSeats:       slot.TotalSeats,
		FreeSeats:        slot.FreeSeats,
		FreeRentalBoards: slot.FreeRentalBoards,
		Price:            slot.Price,
		RentalPrice:      slot.RentalPrice,
		MeetingPoint:     slot.MeetingPoint,
		MeetingPointLat:  float32(slot.MeetingPointLat),
		MeetingPointLng:  float32(slot.MeetingPointLng),
		Status:           slotsapi.SlotStatus(slot.Status),
	}, nil
}

type slotBaseDTO struct {
	id         uuid.UUID
	route      slotsapi.Route
	instructor slotsapi.Instructor
}

// isJSONString сообщает, что сырой jsonb — это JSON-строка (первый значащий байт — кавычка),
// а не массив/объект. Используется, чтобы отличить encoded-polyline от массива координат.
func isJSONString(raw []byte) bool {
	for _, b := range raw {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '"':
			return true
		default:
			return false
		}
	}
	return false
}

func slotBase(slot postgres.Slot) (slotBaseDTO, error) {
	slotID, err := uuid.Parse(slot.ID)
	if err != nil {
		return slotBaseDTO{}, err
	}
	routeID, err := uuid.Parse(slot.RouteID)
	if err != nil {
		return slotBaseDTO{}, err
	}
	instructorID, err := uuid.Parse(slot.InstructorID)
	if err != nil {
		return slotBaseDTO{}, err
	}
	route := slotsapi.Route{
		Id:          routeID,
		Name:        slot.RouteName,
		Type:        slotsapi.RouteType(slot.RouteType),
		CapacityCap: slot.RouteCapacityCap,
		DurationMin: slot.RouteDurationMin,
	}
	// geometry — обязательное поле Route по контракту (маршрут для карты, SCR-003).
	// По контракту это oneOf: массив координат [[lat,lng],...] ИЛИ строка (encoded polyline).
	// Различаем форму jsonb, чтобы строка не роняла всю выдачу /slots в 500.
	if len(slot.RouteGeometry) > 0 {
		var geometry slotsapi.Geometry
		if isJSONString(slot.RouteGeometry) {
			var polyline string
			if err := json.Unmarshal(slot.RouteGeometry, &polyline); err != nil {
				return slotBaseDTO{}, fmt.Errorf("decode route geometry (polyline): %w", err)
			}
			if err := geometry.FromGeometry1(polyline); err != nil {
				return slotBaseDTO{}, fmt.Errorf("encode route geometry (polyline): %w", err)
			}
		} else {
			var coords [][]float32
			if err := json.Unmarshal(slot.RouteGeometry, &coords); err != nil {
				return slotBaseDTO{}, fmt.Errorf("decode route geometry (coords): %w", err)
			}
			if err := geometry.FromGeometry0(coords); err != nil {
				return slotBaseDTO{}, fmt.Errorf("encode route geometry (coords): %w", err)
			}
		}
		route.Geometry = &geometry
	}
	return slotBaseDTO{
		id:         slotID,
		route:      route,
		instructor: slotsapi.Instructor{Id: instructorID, Name: slot.InstructorName},
	}, nil
}
