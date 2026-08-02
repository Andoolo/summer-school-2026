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
	"summer-school-2026/backend/internal/service/profile"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestProfileFlow(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "profile-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	if _, err := db.Exec(ctx, `INSERT INTO clients (id, phone, name) VALUES ($1, $2, $3)`, clientID, "+79990000001", "Иван"); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO auth_sessions (client_id, token_hash, expires_at) VALUES ($1, $2, $3)`, clientID, auth.HashToken(token), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	profileRepo := postgres.NewProfileRepository(db)
	profileService := profile.NewService(profileRepo, slog.Default())
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Profile: handlers.NewProfileHandler(profileService)})

	getRecorder := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/profile", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(getRecorder, getReq)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get profile status = %d, body = %s", getRecorder.Code, getRecorder.Body.String())
	}

	patchRecorder := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/profile", bytes.NewBufferString(`{"name":"Пётр"}`))
	patchReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(patchRecorder, patchReq)
	if patchRecorder.Code != http.StatusOK {
		t.Fatalf("patch profile status = %d, body = %s", patchRecorder.Code, patchRecorder.Body.String())
	}
	var updated struct {
		Name *string `json:"name"`
	}
	if err := json.Unmarshal(patchRecorder.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated profile: %v", err)
	}
	if updated.Name == nil || *updated.Name != "Пётр" {
		t.Fatalf("updated name = %v", updated.Name)
	}

	newPhone := "+79990000002"
	code := "123456"
	if err := profileRepo.CreateOTP(ctx, newPhone, "phone_change", auth.HashOTP(newPhone, "phone_change", code), time.Now().Add(5*time.Minute)); err != nil {
		t.Fatalf("create phone change otp: %v", err)
	}
	confirmRecorder := httptest.NewRecorder()
	confirmReq := httptest.NewRequest(http.MethodPost, "/profile/phone/confirm", bytes.NewBufferString(`{"new_phone":"+79990000002","code":"123456"}`))
	confirmReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(confirmRecorder, confirmReq)
	if confirmRecorder.Code != http.StatusOK {
		t.Fatalf("confirm phone status = %d, body = %s", confirmRecorder.Code, confirmRecorder.Body.String())
	}

	deleteRecorder := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/profile", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(deleteRecorder, deleteReq)
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("delete account status = %d, body = %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}

	afterDeleteRecorder := httptest.NewRecorder()
	afterDeleteReq := httptest.NewRequest(http.MethodGet, "/profile", nil)
	afterDeleteReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(afterDeleteRecorder, afterDeleteReq)
	if afterDeleteRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("after delete get status = %d, want %d", afterDeleteRecorder.Code, http.StatusUnauthorized)
	}
}

func TestProfilePhoneConflict(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "profile-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	otherClientID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	if _, err := db.Exec(ctx, `INSERT INTO clients (id, phone) VALUES ($1, $2), ($3, $4)`, clientID, "+79990000001", otherClientID, "+79990000002"); err != nil {
		t.Fatalf("insert clients: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO auth_sessions (client_id, token_hash, expires_at) VALUES ($1, $2, $3)`, clientID, auth.HashToken(token), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	profileRepo := postgres.NewProfileRepository(db)
	profileService := profile.NewService(profileRepo, slog.Default())
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Profile: handlers.NewProfileHandler(profileService)})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/profile/phone/request-code", bytes.NewBufferString(`{"new_phone":"+79990000002"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestProfilePhoneRequestCodeReturnsDemoCode(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "profile-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	if _, err := db.Exec(ctx, `INSERT INTO clients (id, phone) VALUES ($1, $2)`, clientID, "+79990000001"); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO auth_sessions (client_id, token_hash, expires_at) VALUES ($1, $2, $3)`, clientID, auth.HashToken(token), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	profileRepo := postgres.NewProfileRepository(db)
	profileService := profile.NewService(profileRepo, slog.Default())
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Profile: handlers.NewProfileHandler(profileService)})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/profile/phone/request-code", bytes.NewBufferString(`{"new_phone":"+79990000003"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Code) != 4 {
		t.Fatalf("code = %q, want 4 digits", response.Code)
	}
}

func TestProfileRequiresToken(t *testing.T) {
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Profile: handlers.NewProfileHandler(profile.NewService(nil, slog.Default()))})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/profile", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
