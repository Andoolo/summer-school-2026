package handlers_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/http/handlers"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestCatalogSlotsFiltersAndDetails(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{
		Slots:       handlers.NewSlotHandler(postgres.NewSlotRepository(db)),
		Instructors: handlers.NewInstructorHandler(postgres.NewInstructorRepository(db)),
	})

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/slots?route_type=novice&only_available=true&limit=10&offset=0", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list slots status = %d, body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var listResponse struct {
		Items []struct {
			ID    string `json:"id"`
			Route struct {
				Type string `json:"type"`
			} `json:"route"`
		} `json:"items"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResponse.Items) != 1 || listResponse.Meta.Total != 1 || listResponse.Items[0].Route.Type != "novice" {
		t.Fatalf("unexpected slot list response: %+v", listResponse)
	}

	detailRecorder := httptest.NewRecorder()
	router.ServeHTTP(detailRecorder, httptest.NewRequest(http.MethodGet, "/slots/55555555-5555-5555-5555-555555555555", nil))
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("get slot status = %d, body = %s", detailRecorder.Code, detailRecorder.Body.String())
	}
	var detailResponse struct {
		MeetingPoint string `json:"meeting_point"`
		Instructor   struct {
			Name string `json:"name"`
		} `json:"instructor"`
	}
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &detailResponse); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	if detailResponse.MeetingPoint == "" || detailResponse.Instructor.Name == "" {
		t.Fatalf("slot detail misses nested data: %+v", detailResponse)
	}

	notFoundRecorder := httptest.NewRecorder()
	router.ServeHTTP(notFoundRecorder, httptest.NewRequest(http.MethodGet, "/slots/00000000-0000-0000-0000-000000000000", nil))
	if notFoundRecorder.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d, want %d", notFoundRecorder.Code, http.StatusNotFound)
	}
}

func TestCatalogInstructorsPagination(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Instructors: handlers.NewInstructorHandler(postgres.NewInstructorRepository(db))})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/instructors?limit=1&offset=0", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("list instructors status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Meta struct {
			Limit int `json:"limit"`
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode instructors response: %v", err)
	}
	if len(response.Items) != 1 || response.Meta.Limit != 1 || response.Meta.Total != 2 {
		t.Fatalf("unexpected instructors response: %+v", response)
	}
}
