package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	CodeBadRequest          = "bad_request"
	CodeUnauthorized        = "unauthorized"
	CodeForbidden           = "forbidden"
	CodeNotFound            = "not_found"
	CodeSlotFull            = "slot_full"
	CodeDoubleBooking       = "double_booking"
	CodeSlotCancelled       = "slot_cancelled"
	CodeSlotStarted         = "slot_started"
	CodeAlreadyCancelled    = "already_cancelled"
	CodeInvalidCode         = "invalid_code"
	CodeIdempotencyConflict = "idempotency_conflict"
	CodePhoneConflict       = "phone_conflict"
	CodeTooManyRequests     = "too_many_requests"
	CodeInternalError       = "internal_error"
)

var ErrUnauthorized = errors.New("unauthorized")

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, code, message string, details any) {
	WriteJSON(w, status, ErrorResponse{Code: code, Message: message, Details: details})
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	WriteJSON(w, status, payload)
}
