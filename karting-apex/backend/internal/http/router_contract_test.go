package httpapi_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/http/handlers"
	"summer-school-2026/backend/internal/service/booking"
)

func TestOpenAPIParameterErrorsReturnContractError(t *testing.T) {
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{
		Slots:    handlers.NewSlotHandler(nil),
		Bookings: handlers.NewBookingHandler(booking.NewService(nil)),
	})

	tests := []struct {
		name   string
		method string
		path   string
		auth   bool
	}{
		{name: "invalid slot id", method: http.MethodGet, path: "/slots/not-a-uuid"},
		{name: "invalid date", method: http.MethodGet, path: "/slots?date_from=bad-date"},
		{name: "invalid bool", method: http.MethodGet, path: "/slots?only_available=maybe"},
		{name: "invalid route type", method: http.MethodGet, path: "/slots?route_type=hard"},
		{name: "invalid booking status", method: http.MethodGet, path: "/bookings?status=past", auth: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.auth {
				req.Header.Set("Authorization", "Bearer token")
			}
			router.ServeHTTP(recorder, req)

			assertContractError(t, recorder, http.StatusBadRequest, httpapi.CodeBadRequest)
		})
	}
}

func TestOpenAPIParameterErrorsKeepJSONContentType(t *testing.T) {
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Slots: handlers.NewSlotHandler(nil)})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/slots/not-a-uuid", nil))

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func assertContractError(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, wantStatus, recorder.Body.String())
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Code != wantCode || body.Message == "" {
		t.Fatalf("error body = %+v, want code %q and non-empty message", body, wantCode)
	}
}
