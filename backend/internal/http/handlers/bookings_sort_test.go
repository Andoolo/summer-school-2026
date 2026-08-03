package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

// TestListBookingsSortOrder — регресс на B2 (двунаправленная сортировка списка броней):
// предстоящие идут по возрастанию start_at (ближайшая первой, AC-002), затем прошедшие
// по убыванию start_at (свежая первой, AC-003).
// Написан в рамках Блока 3 (Столп 3, автотесты) — покрывает многоэлементный порядок,
// который прежние тесты не проверяли (list использовал limit=1).
func TestListBookingsSortOrder(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "sort-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	insertClientSession(t, ctx, db, clientID, "+79990005001", token)

	routeID := "11111111-1111-1111-1111-111111111111"      // из dev-seed
	instructorID := "33333333-3333-3333-3333-333333333333" // из dev-seed

	// slotID -> смещение start_at от now
	slots := []struct {
		slotID    string
		bookingID string
		offset    time.Duration
	}{
		{"f1000000-0000-0000-0000-000000000001", "b1000000-0000-0000-0000-000000000001", 1 * 24 * time.Hour},   // +1д  (ближайшая предстоящая)
		{"f1000000-0000-0000-0000-000000000010", "b1000000-0000-0000-0000-000000000010", 10 * 24 * time.Hour},  // +10д
		{"f2000000-0000-0000-0000-000000000001", "b2000000-0000-0000-0000-000000000001", -1 * 24 * time.Hour},  // -1д  (свежая прошедшая)
		{"f2000000-0000-0000-0000-000000000010", "b2000000-0000-0000-0000-000000000010", -10 * 24 * time.Hour}, // -10д
	}

	now := time.Now().UTC()
	for _, s := range slots {
		if _, err := db.Exec(ctx, `
INSERT INTO slots (id, route_id, instructor_id, start_at, total_seats, free_seats,
                   rental_boards_total, free_rental_boards, price, rental_price,
                   meeting_point, meeting_point_lat, meeting_point_lng, status)
VALUES ($1, $2, $3, $4, 8, 8, 12, 12, 2500, 800, 'МП', 59.9, 30.2, 'scheduled')`,
			s.slotID, routeID, instructorID, now.Add(s.offset)); err != nil {
			t.Fatalf("insert slot %s: %v", s.slotID, err)
		}
		insertBooking(t, ctx, db, s.bookingID, s.slotID, clientID, 1, 0)
	}

	router := bookingRouter(db)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bookings?limit=10&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	want := []string{
		"b1000000-0000-0000-0000-000000000001", // +1д  предстоящая, ближайшая
		"b1000000-0000-0000-0000-000000000010", // +10д предстоящая
		"b2000000-0000-0000-0000-000000000001", // -1д  прошедшая, свежая
		"b2000000-0000-0000-0000-000000000010", // -10д прошедшая
	}
	if len(response.Items) != len(want) {
		t.Fatalf("got %d items, want %d: %+v", len(response.Items), len(want), response.Items)
	}
	for i, id := range want {
		if response.Items[i].ID != id {
			gotIDs := make([]string, len(response.Items))
			for j, it := range response.Items {
				gotIDs[j] = it.ID
			}
			t.Fatalf("порядок неверный на позиции %d: got %v, want %v", i, gotIDs, want)
		}
	}
}
