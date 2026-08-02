package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestUnknownPathReturnsContractError(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)

	NewRouter(slog.Default()).ServeHTTP(recorder, req)

	assertErrorResponse(t, recorder, http.StatusNotFound, CodeNotFound)
}

func TestInvalidJSONReturnsContractError(t *testing.T) {
	router := testRouter()
	router.Post("/decode", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if err := DecodeJSON(r, &body); err != nil {
			WriteError(w, http.StatusBadRequest, CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
			return
		}
		writeJSON(w, http.StatusOK, body)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/decode", bytes.NewBufferString(`{"name":`))
	router.ServeHTTP(recorder, req)

	assertErrorResponse(t, recorder, http.StatusBadRequest, CodeBadRequest)
}

func TestMissingBearerTokenReturnsContractError(t *testing.T) {
	router := testRouter()
	router.With(RequireAuth).Get("/private", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	router.ServeHTTP(recorder, req)

	assertErrorResponse(t, recorder, http.StatusUnauthorized, CodeUnauthorized)
}

func TestBearerTokenAcceptsCaseInsensitiveScheme(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "bearer token-123")

	token, err := BearerToken(req)
	if err != nil {
		t.Fatalf("BearerToken() error = %v", err)
	}
	if token != "token-123" {
		t.Fatalf("token = %q, want %q", token, "token-123")
	}
}

func testRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Use(requestIDMiddleware)
	router.Use(recoverMiddleware(slog.Default()))
	router.Use(accessLogMiddleware(slog.Default()))
	router.Use(jsonContentTypeMiddleware)
	return router
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, wantStatus, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}

	var body ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != wantCode {
		t.Fatalf("error code = %q, want %q", body.Code, wantCode)
	}
	if body.Message == "" {
		t.Fatal("error message must not be empty")
	}
}
