package handlers_test

import (
	"context"
	"encoding/json"
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
)

func TestWaitlistEndpoints(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	const slotID = "66666666-6666-6666-6666-666666666666"
	if _, err := db.Exec(ctx, `UPDATE slots SET start_at = $1, free_seats = 2 WHERE id = $2`, time.Now().Add(5*time.Hour), slotID); err != nil {
		t.Fatalf("update slot: %v", err)
	}
	clientID := "abababab-abab-abab-abab-abababababab"
	insertClientSession(t, ctx, db, clientID, "+79990016001", "waitlist-token")

	joins := 0
	handler := handlers.NewWaitlistHandler(waitlist.NewService(postgres.NewWaitlistRepository(db)), nil, func() { joins++ })
	router := httpapi.NewRouter(nil, httpapi.RouterOptions{
		WaitlistStatus: handler.Status, WaitlistJoin: handler.Join, WaitlistLeave: handler.Leave,
	})
	call := func(method, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/slots/"+slotID+"/waitlist", strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	expectCode := func(recorder *httptest.ResponseRecorder, status int, code string) {
		t.Helper()
		if recorder.Code != status || (code != "" && !strings.Contains(recorder.Body.String(), `"code":"`+code+`"`)) {
			t.Fatalf("status = %d body = %s; want %d %s", recorder.Code, recorder.Body.String(), status, code)
		}
	}

	expectCode(call(http.MethodGet, "", ""), http.StatusUnauthorized, "unauthorized")

	// Без Telegram уведомить не выйдет.
	expectCode(call(http.MethodPost, "waitlist-token", `{"seats_count":1}`), http.StatusForbidden, handlers.CodeTelegramRequired)

	if _, err := db.Exec(ctx, `UPDATE clients SET telegram_chat_id = 9001, telegram_notifications = false WHERE id = $1`, clientID); err != nil {
		t.Fatal(err)
	}
	expectCode(call(http.MethodPost, "waitlist-token", `{"seats_count":1}`), http.StatusForbidden, handlers.CodeNotificationsDisabled)
	if _, err := db.Exec(ctx, `UPDATE clients SET telegram_notifications = true WHERE id = $1`, clientID); err != nil {
		t.Fatal(err)
	}

	expectCode(call(http.MethodPost, "waitlist-token", `{"seats_count":1}`), http.StatusConflict, handlers.CodeSeatsAvailable)
	expectCode(call(http.MethodPost, "waitlist-token", `{"seats_count":7}`), http.StatusBadRequest, "bad_request")

	if _, err := db.Exec(ctx, `UPDATE slots SET free_seats = 0 WHERE id = $1`, slotID); err != nil {
		t.Fatal(err)
	}
	expectCode(call(http.MethodPost, "waitlist-token", `{"seats_count":2}`), http.StatusCreated, "")
	expectCode(call(http.MethodPost, "waitlist-token", `{"seats_count":2}`), http.StatusOK, "")
	if joins != 1 {
		t.Fatalf("dispatcher woken %d times, want once for the new entry", joins)
	}

	var status struct {
		Entry *struct {
			Status     string `json:"status"`
			SeatsCount int    `json:"seats_count"`
			Position   int    `json:"position"`
		} `json:"entry"`
		TelegramLinked       bool `json:"telegram_linked"`
		NotificationsEnabled bool `json:"notifications_enabled"`
	}
	recorder := call(http.MethodGet, "waitlist-token", "")
	expectCode(recorder, http.StatusOK, "")
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Entry == nil || status.Entry.Status != "waiting" || status.Entry.Position != 1 || status.Entry.SeatsCount != 2 || !status.TelegramLinked || !status.NotificationsEnabled {
		t.Fatalf("status = %s", recorder.Body.String())
	}

	expectCode(call(http.MethodDelete, "waitlist-token", ""), http.StatusNoContent, "")
	recorder = call(http.MethodGet, "waitlist-token", "")
	if !strings.Contains(recorder.Body.String(), `"entry":null`) {
		t.Fatalf("status after leave = %s", recorder.Body.String())
	}

	bad := httptest.NewRequest(http.MethodGet, "/slots/not-a-uuid/waitlist", nil)
	bad.Header.Set("Authorization", "Bearer waitlist-token")
	badRecorder := httptest.NewRecorder()
	router.ServeHTTP(badRecorder, bad)
	expectCode(badRecorder, http.StatusNotFound, "not_found")
}
