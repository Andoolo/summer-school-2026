package handlers

import (
	"errors"
	"net/http"

	httpapi "summer-school-2026/backend/internal/http"
	bookingsapi "summer-school-2026/backend/internal/http/openapi/bookings"
	"summer-school-2026/backend/internal/service/booking"

	"github.com/google/uuid"
)

type BookingHandler struct {
	service *booking.Service
}

func NewBookingHandler(service *booking.Service) *BookingHandler {
	return &BookingHandler{service: service}
}

func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request, params bookingsapi.CreateBookingParams) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	var req bookingsapi.CreateBookingRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
		return
	}

	idempotencyKey := ""
	if params.IdempotencyKey != nil {
		idempotencyKey = params.IdempotencyKey.String()
	}
	created, err := h.service.Create(r.Context(), booking.CreateCommand{
		Token:          token,
		IdempotencyKey: idempotencyKey,
		SlotID:         req.SlotId.String(),
		SeatsCount:     req.SeatsCount,
		RentalCount:    req.RentalCount,
	})
	if err != nil {
		writeBookingError(w, err)
		return
	}
	mapped, err := bookingDTO(created)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, mapped)
}

func (h *BookingHandler) ListBookings(w http.ResponseWriter, r *http.Request, params bookingsapi.ListBookingsParams) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	limit, offset, ok := pagination(w, params.Limit, params.Offset)
	if !ok {
		return
	}
	var status *string
	if params.Status != nil {
		value := string(*params.Status)
		if value != string(bookingsapi.BookingStatusActive) && value != string(bookingsapi.BookingStatusCancelled) && value != string(bookingsapi.BookingStatusLateCancel) {
			httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
			return
		}
		status = &value
	}
	list, err := h.service.List(r.Context(), booking.ListCommand{Token: token, Status: status, Limit: limit, Offset: offset})
	if err != nil {
		writeBookingError(w, err)
		return
	}

	items := make([]bookingsapi.BookingSummary, 0, len(list.Items))
	for _, item := range list.Items {
		mapped, err := bookingSummaryDTO(item)
		if err != nil {
			httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
			return
		}
		items = append(items, mapped)
	}
	httpapi.WriteJSON(w, http.StatusOK, bookingsapi.BookingListResponse{Items: items, Meta: bookingsapi.PaginationMeta{Limit: limit, Offset: offset, Total: list.Total}})
}

func (h *BookingHandler) GetBooking(w http.ResponseWriter, r *http.Request, bookingId bookingsapi.BookingIdParam) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	found, err := h.service.Get(r.Context(), token, bookingId.String())
	if err != nil {
		writeBookingError(w, err)
		return
	}
	mapped, err := bookingDTO(found)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, mapped)
}

func (h *BookingHandler) CancelBooking(w http.ResponseWriter, r *http.Request, bookingId bookingsapi.BookingIdParam) {
	token, ok := bearerOrUnauthorized(w, r)
	if !ok {
		return
	}
	cancelled, err := h.service.Cancel(r.Context(), token, bookingId.String())
	if err != nil {
		writeBookingError(w, err)
		return
	}
	mapped, err := bookingDTO(cancelled)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, mapped)
}

func writeBookingError(w http.ResponseWriter, err error) {
	var availability booking.AvailabilityError
	switch {
	case errors.Is(err, booking.ErrUnauthorized):
		httpapi.WriteError(w, http.StatusUnauthorized, httpapi.CodeUnauthorized, "Требуется авторизация. Передайте действительный токен в заголовке Authorization.", nil)
	case errors.Is(err, booking.ErrInvalidRequest):
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
	case errors.Is(err, booking.ErrDoubleBooking):
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodeDoubleBooking, "Вы уже записаны на выбранный слот.", nil)
	case errors.Is(err, booking.ErrIdempotencyConflict):
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodeIdempotencyConflict, "Ключ идемпотентности уже использован для другого запроса.", nil)
	case errors.Is(err, booking.ErrAlreadyCancelled):
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodeAlreadyCancelled, "Бронь уже отменена.", nil)
	case errors.Is(err, booking.ErrForbidden):
		httpapi.WriteError(w, http.StatusForbidden, httpapi.CodeForbidden, "Доступ запрещён. Вы не можете обращаться к данным другого клиента.", nil)
	case errors.Is(err, booking.ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, httpapi.CodeNotFound, "Запрашиваемый ресурс не найден.", nil)
	case errors.As(err, &availability) && errors.Is(err, booking.ErrSlotFull):
		httpapi.WriteError(w, http.StatusConflict, httpapi.CodeSlotFull, "На выбранном слоте не осталось свободных мест.", availabilityDetails(availability.Availability))
	case errors.As(err, &availability) && errors.Is(err, booking.ErrSlotCancelled):
		httpapi.WriteError(w, http.StatusGone, httpapi.CodeSlotCancelled, "Слот отменён и более недоступен для бронирования.", availabilityDetails(availability.Availability))
	case errors.Is(err, booking.ErrSlotStarted):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, httpapi.CodeSlotStarted, "Слот уже стартовал, операция недоступна.", nil)
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
	}
}

func availabilityDetails(availability booking.Availability) map[string]int {
	return map[string]int{
		"available_seats":         availability.AvailableSeats,
		"available_rental_boards": availability.AvailableRentalBoards,
	}
}

func bookingDTO(value booking.Booking) (bookingsapi.Booking, error) {
	id, err := uuid.Parse(value.ID)
	if err != nil {
		return bookingsapi.Booking{}, err
	}
	slotID, err := uuid.Parse(value.SlotID)
	if err != nil {
		return bookingsapi.Booking{}, err
	}
	clientID, err := uuid.Parse(value.ClientID)
	if err != nil {
		return bookingsapi.Booking{}, err
	}
	slot, err := bookingSlotDTO(value.Slot)
	if err != nil {
		return bookingsapi.Booking{}, err
	}
	priceTotal := value.PriceTotal
	return bookingsapi.Booking{
		Id:          id,
		SlotId:      slotID,
		ClientId:    clientID,
		SeatsCount:  value.SeatsCount,
		RentalCount: value.RentalCount,
		Status:      bookingsapi.BookingStatus(value.Status),
		PriceTotal:  &priceTotal,
		CreatedAt:   value.CreatedAt,
		CancelledAt: value.CancelledAt,
		Slot:        &slot,
	}, nil
}

func bookingSummaryDTO(value booking.Booking) (bookingsapi.BookingSummary, error) {
	id, err := uuid.Parse(value.ID)
	if err != nil {
		return bookingsapi.BookingSummary{}, err
	}
	slotID, err := uuid.Parse(value.SlotID)
	if err != nil {
		return bookingsapi.BookingSummary{}, err
	}
	slot, err := bookingSlotSummaryDTO(value.Slot)
	if err != nil {
		return bookingsapi.BookingSummary{}, err
	}
	priceTotal := value.PriceTotal
	return bookingsapi.BookingSummary{
		Id:          id,
		SlotId:      slotID,
		SeatsCount:  value.SeatsCount,
		RentalCount: value.RentalCount,
		Status:      bookingsapi.BookingStatus(value.Status),
		PriceTotal:  &priceTotal,
		CreatedAt:   value.CreatedAt,
		CancelledAt: value.CancelledAt,
		Slot:        &slot,
	}, nil
}

func bookingSlotDTO(value booking.Slot) (bookingsapi.Slot, error) {
	slotID, err := uuid.Parse(value.ID)
	if err != nil {
		return bookingsapi.Slot{}, err
	}
	routeID, err := uuid.Parse(value.RouteID)
	if err != nil {
		return bookingsapi.Slot{}, err
	}
	instructorID, err := uuid.Parse(value.InstructorID)
	if err != nil {
		return bookingsapi.Slot{}, err
	}
	return bookingsapi.Slot{
		Id:               slotID,
		Route:            bookingsapi.Route{Id: routeID, Name: value.RouteName, Type: bookingsapi.RouteType(value.RouteType), CapacityCap: value.RouteCapacityCap, DurationMin: value.RouteDurationMin},
		Instructor:       bookingsapi.Instructor{Id: instructorID, Name: value.InstructorName},
		StartAt:          value.StartAt,
		TotalSeats:       value.TotalSeats,
		FreeSeats:        value.FreeSeats,
		FreeRentalBoards: value.FreeRentalBoards,
		Price:            value.Price,
		RentalPrice:      value.RentalPrice,
		MeetingPoint:     value.MeetingPoint,
		MeetingPointLat:  float32(value.MeetingPointLat),
		MeetingPointLng:  float32(value.MeetingPointLng),
		Status:           bookingsapi.SlotStatus(value.Status),
	}, nil
}

func bookingSlotSummaryDTO(value booking.Slot) (bookingsapi.SlotSummary, error) {
	slot, err := bookingSlotDTO(value)
	if err != nil {
		return bookingsapi.SlotSummary{}, err
	}
	return bookingsapi.SlotSummary{
		Id:               slot.Id,
		Route:            slot.Route,
		Instructor:       slot.Instructor,
		StartAt:          slot.StartAt,
		TotalSeats:       slot.TotalSeats,
		FreeSeats:        slot.FreeSeats,
		FreeRentalBoards: slot.FreeRentalBoards,
		Price:            slot.Price,
		RentalPrice:      slot.RentalPrice,
		Status:           slot.Status,
	}, nil
}
