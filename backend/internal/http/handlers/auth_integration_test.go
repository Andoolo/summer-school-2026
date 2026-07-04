package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/http/handlers"
	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestAuthVerifyAndLogoutFlow(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	repo := postgres.NewAuthRepository(db)
	phone := "+79991234567"
	code := "123456"
	if err := repo.CreateOTP(ctx, phone, "login", auth.HashOTP(phone, "login", code), time.Now().Add(5*time.Minute)); err != nil {
		t.Fatalf("create otp: %v", err)
	}

	service := auth.NewService(repo, slog.Default())
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Auth: handlers.NewAuthHandler(service, true)})

	verifyBody := bytes.NewBufferString(`{"phone":"+79991234567","code":"123456"}`)
	verifyRecorder := httptest.NewRecorder()
	router.ServeHTTP(verifyRecorder, httptest.NewRequest(http.MethodPost, "/auth/verify-code", verifyBody))

	if verifyRecorder.Code != http.StatusOK {
		t.Fatalf("verify status = %d, body = %s", verifyRecorder.Code, verifyRecorder.Body.String())
	}
	var verifyResponse struct {
		Token  string `json:"token"`
		IsNew  bool   `json:"is_new"`
		Client struct {
			Phone string `json:"phone"`
		} `json:"client"`
	}
	if err := json.Unmarshal(verifyRecorder.Body.Bytes(), &verifyResponse); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if verifyResponse.Token == "" || !verifyResponse.IsNew || verifyResponse.Client.Phone != phone {
		t.Fatalf("unexpected verify response: %+v", verifyResponse)
	}

	logoutRecorder := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+verifyResponse.Token)
	router.ServeHTTP(logoutRecorder, logoutReq)

	if logoutRecorder.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %s", logoutRecorder.Code, logoutRecorder.Body.String())
	}
}

func TestAuthRequestCodeReturnsDemoCode(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	service := auth.NewService(postgres.NewAuthRepository(db), slog.Default())
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Auth: handlers.NewAuthHandler(service, true)})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewBufferString(`{"phone":"+79991234567"}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("request code status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Code               string `json:"code"`
		TTLSeconds         int    `json:"ttl_seconds"`
		ResendAfterSeconds int    `json:"resend_after_seconds"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode request code response: %v", err)
	}
	if len(response.Code) != 4 || response.TTLSeconds != 300 || response.ResendAfterSeconds != 60 {
		t.Fatalf("unexpected response: %+v", response)
	}
}
