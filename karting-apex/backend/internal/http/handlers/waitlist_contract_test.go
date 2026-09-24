package handlers_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/http/handlers"
	"summer-school-2026/backend/internal/service/waitlist"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	legacyrouter "github.com/getkin/kin-openapi/routers/legacy"
)

// Контракт листа ожидания — 00-analysis-karting/api/waitlist/api.yaml (через общий
// openapi.yaml). Ручки написаны вручную, без генерации, поэтому их сверяет этот тест:
// каждый запрос и ответ реальных хэндлеров проверяется по спецификации.
const kartingSpecPath = "../../../../../00-analysis-karting/api/openapi.yaml"

func loadKartingSpec(t *testing.T) routers.Router {
	t.Helper()
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromFile(kartingSpecPath)
	if err != nil {
		t.Fatalf("load OpenAPI spec: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("OpenAPI spec is invalid: %v", err)
	}
	// Сервер в спецификации — учебный адрес с префиксом /v1; API отвечает без префикса.
	doc.Servers = nil
	router, err := legacyrouter.NewRouter(doc)
	if err != nil {
		t.Fatalf("build spec router: %v", err)
	}
	return router
}

type contractCall struct {
	method, path, token, body string
	// requestInvalid — запрос нарочно нарушает контракт (проверяем ответ на него).
	requestInvalid bool
}

func TestWaitlistMatchesOpenAPIContract(t *testing.T) {
	spec := loadKartingSpec(t)
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	const slotID = "66666666-6666-6666-6666-666666666666"
	if _, err := db.Exec(ctx, `UPDATE slots SET start_at = $1, free_seats = 2 WHERE id = $2`, time.Now().Add(5*time.Hour), slotID); err != nil {
		t.Fatal(err)
	}
	clientID := "cdcdcdcd-cdcd-cdcd-cdcd-cdcdcdcdcdcd"
	insertClientSession(t, ctx, db, clientID, "+79990021001", "contract-token")

	handler := handlers.NewWaitlistHandler(waitlist.NewService(postgres.NewWaitlistRepository(db), nil), nil)
	app := httpapi.NewRouter(nil, httpapi.RouterOptions{
		WaitlistStatus: handler.Status, WaitlistJoin: handler.Join, WaitlistLeave: handler.Leave, WaitlistMine: handler.Mine,
	})

	exchange := func(c contractCall) int {
		t.Helper()
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		route, pathParams, err := spec.FindRoute(req)
		if err != nil {
			t.Fatalf("%s %s is not in the contract: %v", c.method, c.path, err)
		}
		input := &openapi3filter.RequestValidationInput{
			Request: req, PathParams: pathParams, Route: route,
			Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
		}
		if err := openapi3filter.ValidateRequest(ctx, input); err != nil && !c.requestInvalid {
			t.Fatalf("%s %s request violates the contract: %v", c.method, c.path, err)
		} else if err == nil && c.requestInvalid {
			t.Fatalf("%s %s %s must violate the contract", c.method, c.path, c.body)
		}
		// Валидация прочитала тело запроса — отдаём его приложению заново.
		req.Body = io.NopCloser(strings.NewReader(c.body))

		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if err := openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
			RequestValidationInput: input,
			Status:                 rec.Code,
			Header:                 rec.Header(),
			Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
			Options:                &openapi3filter.Options{IncludeResponseStatus: true},
		}); err != nil {
			t.Fatalf("%s %s → %d %s violates the contract: %v", c.method, c.path, rec.Code, rec.Body.String(), err)
		}
		return rec.Code
	}
	expect := func(c contractCall, status int) {
		t.Helper()
		if got := exchange(c); got != status {
			t.Fatalf("%s %s status = %d, want %d", c.method, c.path, got, status)
		}
	}
	slotPath := "/slots/" + slotID + "/waitlist"
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}

	expect(contractCall{method: http.MethodGet, path: slotPath}, http.StatusUnauthorized)
	expect(contractCall{method: http.MethodGet, path: slotPath, token: "contract-token"}, http.StatusOK)
	expect(contractCall{method: http.MethodPost, path: slotPath, token: "contract-token", body: `{"seats_count":1}`}, http.StatusForbidden)
	exec(`UPDATE clients SET telegram_chat_id = 9101, telegram_notifications = false WHERE id = $1`, clientID)
	expect(contractCall{method: http.MethodPost, path: slotPath, token: "contract-token", body: `{"seats_count":1}`}, http.StatusForbidden)
	exec(`UPDATE clients SET telegram_notifications = true WHERE id = $1`, clientID)
	expect(contractCall{method: http.MethodPost, path: slotPath, token: "contract-token", body: `{"seats_count":1}`}, http.StatusConflict)
	expect(contractCall{method: http.MethodPost, path: slotPath, token: "contract-token", body: `{"seats_count":7}`, requestInvalid: true}, http.StatusBadRequest)

	exec(`UPDATE slots SET free_seats = 0 WHERE id = $1`, slotID)
	expect(contractCall{method: http.MethodPost, path: slotPath, token: "contract-token", body: `{"seats_count":2}`}, http.StatusCreated)
	expect(contractCall{method: http.MethodPost, path: slotPath, token: "contract-token", body: `{"seats_count":2}`}, http.StatusOK)
	expect(contractCall{method: http.MethodGet, path: slotPath, token: "contract-token"}, http.StatusOK)
	expect(contractCall{method: http.MethodGet, path: "/waitlist", token: "contract-token"}, http.StatusOK)

	// Предложение места: в ответах появляется offer_expires_at.
	exec(`UPDATE waitlist_entries SET status = 'notified', notified_at = now() WHERE client_id = $1`, clientID)
	expect(contractCall{method: http.MethodGet, path: slotPath, token: "contract-token"}, http.StatusOK)
	expect(contractCall{method: http.MethodGet, path: "/waitlist", token: "contract-token"}, http.StatusOK)

	expect(contractCall{method: http.MethodDelete, path: slotPath, token: "contract-token"}, http.StatusNoContent)
	expect(contractCall{method: http.MethodGet, path: slotPath, token: "contract-token"}, http.StatusOK)
	expect(contractCall{method: http.MethodGet, path: "/waitlist", token: "contract-token"}, http.StatusOK)
	expect(contractCall{method: http.MethodGet, path: "/waitlist"}, http.StatusUnauthorized)
	// Спецификация — OpenAPI 3.1: format в ней только пометка, и kin-openapi не отвергает
	// такой путь. Контракт здесь — ответ: 404 задокументирован и сверяется.
	expect(contractCall{method: http.MethodGet, path: "/slots/not-a-uuid/waitlist", token: "contract-token"}, http.StatusNotFound)

	exec(`UPDATE slots SET start_at = $1 WHERE id = $2`, time.Now().Add(-time.Minute), slotID)
	expect(contractCall{method: http.MethodPost, path: slotPath, token: "contract-token", body: `{"seats_count":1}`}, http.StatusUnprocessableEntity)
}
