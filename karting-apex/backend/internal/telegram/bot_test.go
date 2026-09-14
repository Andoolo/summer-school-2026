package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testToken = "123456:SECRET-token_value"

func TestClientCallsMethodsAndDecodesResults(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"username":"apex_login_bot"}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
	t.Cleanup(server.Close)
	client := NewClient(testToken).WithAPIBase(server.URL)

	username, err := client.GetMe(context.Background())
	if err != nil || username != "apex_login_bot" {
		t.Fatalf("GetMe() = %q, %v", username, err)
	}
	if gotPath != "/bot"+testToken+"/getMe" {
		t.Fatalf("path = %q", gotPath)
	}

	if err := client.SetWebhook(context.Background(), "https://example.onrender.com/telegram/webhook", "sec"); err != nil {
		t.Fatalf("SetWebhook() error = %v", err)
	}
	if gotBody["secret_token"] != "sec" || gotBody["drop_pending_updates"] != true {
		t.Fatalf("setWebhook body = %v", gotBody)
	}

	keyboard := ReplyKeyboard{Keyboard: [][]KeyboardButton{{{Text: "Поделиться номером", RequestContact: true}}}, OneTimeKeyboard: true}
	if err := client.SendMessage(context.Background(), 42, "Привет", keyboard); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if gotBody["chat_id"] != float64(42) || gotBody["text"] != "Привет" || gotBody["reply_markup"] == nil {
		t.Fatalf("sendMessage body = %v", gotBody)
	}
	if _, hasParseMode := gotBody["parse_mode"]; hasParseMode {
		t.Fatal("messages must be sent without parse_mode")
	}
}

func TestClientErrorsNeverContainToken(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":401,"description":"Unauthorized"}`))
	}))
	t.Cleanup(failing.Close)

	_, err := NewClient(testToken).WithAPIBase(failing.URL).GetMe(context.Background())
	if !errors.Is(err, ErrAPI) {
		t.Fatalf("GetMe() error = %v, want ErrAPI", err)
	}
	if strings.Contains(err.Error(), testToken) {
		t.Fatalf("API error leaks token: %v", err)
	}

	// Сетевая ошибка: адрес, где никто не слушает. Текст ошибки net/http содержит URL.
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closed.Close()
	err = NewClient(testToken).WithAPIBase(closed.URL).SendMessage(context.Background(), 1, "x", nil)
	if err == nil || strings.Contains(err.Error(), testToken) || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("network error = %v, must exist and not leak token", err)
	}
}

func TestClientAPIErrorCarriesCode(t *testing.T) {
	cases := []struct {
		body      string
		status    int
		permanent bool
		blocked   bool
	}{
		{`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`, http.StatusForbidden, true, true},
		{`{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`, http.StatusBadRequest, true, false},
		{`{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 3"}`, http.StatusTooManyRequests, false, false},
		// Без error_code в теле код берётся из HTTP-ответа.
		{`{"ok":false,"description":"Bad Gateway"}`, http.StatusBadGateway, false, false},
	}
	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		err := NewClient(testToken).WithAPIBase(server.URL).SendMessage(context.Background(), 1, "x", nil)
		server.Close()

		var apiErr *APIError
		if !errors.As(err, &apiErr) || !errors.Is(err, ErrAPI) {
			t.Fatalf("%s: error = %v, want *APIError matching ErrAPI", tc.body, err)
		}
		if apiErr.Code != tc.status || apiErr.Permanent() != tc.permanent || apiErr.Blocked() != tc.blocked {
			t.Fatalf("%s: code=%d permanent=%v blocked=%v", tc.body, apiErr.Code, apiErr.Permanent(), apiErr.Blocked())
		}
		if strings.Contains(err.Error(), testToken) {
			t.Fatalf("API error leaks token: %v", err)
		}
	}
}

func TestWebhookSecretIsStableAndAllowed(t *testing.T) {
	secret := WebhookSecret(testToken)
	if secret != WebhookSecret(testToken) || secret == WebhookSecret("other") {
		t.Fatal("secret must be deterministic per token")
	}
	if len(secret) < 32 || len(secret) > 256 || strings.Contains(secret, testToken) {
		t.Fatalf("secret %q has invalid length or contains token", secret)
	}
	for _, r := range secret {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("secret has disallowed char %q", r)
		}
	}
}
