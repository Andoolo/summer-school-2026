package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/http/handlers"
	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/service/booking"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCreateBookingFlowAndIdempotency(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "booking-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	insertClientSession(t, ctx, db, clientID, "+79990001001", token)

	router := bookingRouter(db)
	body := `{"slot_id":"55555555-5555-5555-5555-555555555555","seats_count":2,"rental_count":1}`
	idempotencyKey := "77777777-7777-7777-7777-777777777777"

	first := performCreateBooking(router, token, idempotencyKey, body)
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d, body = %s", first.Code, first.Body.String())
	}
	var firstResponse struct {
		ID         string `json:"id"`
		PriceTotal int    `json:"price_total"`
		Slot       struct {
			FreeSeats        int `json:"free_seats"`
			FreeRentalBoards int `json:"free_rental_boards"`
		} `json:"slot"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstResponse); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if firstResponse.PriceTotal != 5800 || firstResponse.Slot.FreeSeats != 6 || firstResponse.Slot.FreeRentalBoards != 11 {
		t.Fatalf("unexpected first response: %+v", firstResponse)
	}

	retry := performCreateBooking(router, token, idempotencyKey, body)
	if retry.Code != http.StatusCreated {
		t.Fatalf("retry status = %d, body = %s", retry.Code, retry.Body.String())
	}
	var retryResponse struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(retry.Body.Bytes(), &retryResponse); err != nil {
		t.Fatalf("decode retry response: %v", err)
	}
	if retryResponse.ID != firstResponse.ID {
		t.Fatalf("retry booking id = %q, want %q", retryResponse.ID, firstResponse.ID)
	}

	var bookingCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM bookings WHERE client_id = $1`, clientID).Scan(&bookingCount); err != nil {
		t.Fatalf("count bookings: %v", err)
	}
	if bookingCount != 1 {
		t.Fatalf("booking count = %d, want 1", bookingCount)
	}

	doubleBooking := performCreateBooking(router, token, "88888888-8888-8888-8888-888888888888", body)
	if doubleBooking.Code != http.StatusConflict {
		t.Fatalf("double booking status = %d, want %d", doubleBooking.Code, http.StatusConflict)
	}
}

func TestCreateBookingIdempotencyConflict(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "booking-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	insertClientSession(t, ctx, db, clientID, "+79990001003", token)
	router := bookingRouter(db)
	idempotencyKey := "77777777-7777-7777-7777-777777777777"

	first := performCreateBooking(router, token, idempotencyKey, `{"slot_id":"55555555-5555-5555-5555-555555555555","seats_count":1,"rental_count":0}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d, body = %s", first.Code, first.Body.String())
	}
	second := performCreateBooking(router, token, idempotencyKey, `{"slot_id":"55555555-5555-5555-5555-555555555555","seats_count":2,"rental_count":0}`)
	if second.Code != http.StatusConflict {
		t.Fatalf("second status = %d, body = %s", second.Code, second.Body.String())
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	if body.Code != httpapi.CodeIdempotencyConflict {
		t.Fatalf("code = %q, want %q", body.Code, httpapi.CodeIdempotencyConflict)
	}
}

func TestCreateBookingConcurrencyDoesNotOverbook(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	router := bookingRouter(db)
	body := `{"slot_id":"55555555-5555-5555-5555-555555555555","seats_count":1,"rental_count":1}`

	const requests = 12
	var wg sync.WaitGroup
	statuses := make(chan int, requests)
	for i := 0; i < requests; i++ {
		i := i
		clientID := fmt.Sprintf("aaaaaaaa-aaaa-aaaa-aaaa-%012d", i+1)
		token := fmt.Sprintf("booking-token-%d", i+1)
		insertClientSession(t, ctx, db, clientID, fmt.Sprintf("+79990002%03d", i+1), token)
		wg.Add(1)
		go func() {
			defer wg.Done()
			idempotencyKey := fmt.Sprintf("99999999-9999-9999-9999-%012d", i+1)
			statuses <- performCreateBooking(router, token, idempotencyKey, body).Code
		}()
	}
	wg.Wait()
	close(statuses)

	successes := 0
	conflicts := 0
	for status := range statuses {
		switch status {
		case http.StatusCreated:
			successes++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected status = %d", status)
		}
	}
	if successes != 8 || conflicts != 4 {
		t.Fatalf("successes=%d conflicts=%d, want 8/4", successes, conflicts)
	}

	var freeSeats, freeBoards int
	if err := db.QueryRow(ctx, `SELECT free_seats, free_rental_boards FROM slots WHERE id = '55555555-5555-5555-5555-555555555555'`).Scan(&freeSeats, &freeBoards); err != nil {
		t.Fatalf("read availability: %v", err)
	}
	if freeSeats != 0 || freeBoards != 4 {
		t.Fatalf("availability seats=%d boards=%d, want 0/4", freeSeats, freeBoards)
	}
}

func TestListAndGetBookings(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "booking-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	otherToken := "other-booking-token"
	otherClientID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	insertClientSession(t, ctx, db, clientID, "+79990003001", token)
	insertClientSession(t, ctx, db, otherClientID, "+79990003002", otherToken)

	activeID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	cancelledID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	otherID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	if _, err := db.Exec(ctx, `
INSERT INTO bookings (id, slot_id, client_id, seats_count, rental_count, status, created_at, cancelled_at)
VALUES
    ($1, '55555555-5555-5555-5555-555555555555', $2, 1, 0, 'active', now(), NULL),
    ($3, '66666666-6666-6666-6666-666666666666', $2, 2, 1, 'cancelled', now(), now()),
    ($4, '55555555-5555-5555-5555-555555555555', $5, 1, 0, 'active', now(), NULL)`, activeID, clientID, cancelledID, otherID, otherClientID); err != nil {
		t.Fatalf("insert bookings: %v", err)
	}

	router := bookingRouter(db)

	listRecorder := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/bookings?status=active&limit=1&offset=0", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(listRecorder, listReq)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var listResponse struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
		Meta struct {
			Limit int `json:"limit"`
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResponse.Items) != 1 || listResponse.Items[0].ID != activeID || listResponse.Items[0].Status != "active" || listResponse.Meta.Total != 1 || listResponse.Meta.Limit != 1 {
		t.Fatalf("unexpected list response: %+v", listResponse)
	}

	getRecorder := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/bookings/"+cancelledID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(getRecorder, getReq)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRecorder.Code, getRecorder.Body.String())
	}
	var getResponse struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Slot   struct {
			MeetingPoint string `json:"meeting_point"`
		} `json:"slot"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &getResponse); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if getResponse.ID != cancelledID || getResponse.Status != "cancelled" || getResponse.Slot.MeetingPoint == "" {
		t.Fatalf("unexpected get response: %+v", getResponse)
	}

	forbiddenRecorder := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/bookings/"+otherID, nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(forbiddenRecorder, forbiddenReq)
	if forbiddenRecorder.Code != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, want %d", forbiddenRecorder.Code, http.StatusForbidden)
	}

	notFoundRecorder := httptest.NewRecorder()
	notFoundReq := httptest.NewRequest(http.MethodGet, "/bookings/00000000-0000-0000-0000-000000000000", nil)
	notFoundReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(notFoundRecorder, notFoundReq)
	if notFoundRecorder.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d, want %d", notFoundRecorder.Code, http.StatusNotFound)
	}
}

func TestCancelBookingEarlyLateAndAfterStart(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "cancel-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	insertClientSession(t, ctx, db, clientID, "+79990004001", token)
	router := bookingRouter(db)

	earlyID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	if _, err := db.Exec(ctx, `UPDATE slots SET free_seats = 6, free_rental_boards = 11 WHERE id = '55555555-5555-5555-5555-555555555555'`); err != nil {
		t.Fatalf("update early slot: %v", err)
	}
	insertBooking(t, ctx, db, earlyID, "55555555-5555-5555-5555-555555555555", clientID, 2, 1)
	earlyRecorder := performCancelBooking(router, token, earlyID)
	if earlyRecorder.Code != http.StatusOK {
		t.Fatalf("early cancel status = %d, body = %s", earlyRecorder.Code, earlyRecorder.Body.String())
	}
	var earlyResponse struct {
		Status string `json:"status"`
		Slot   struct {
			FreeSeats        int `json:"free_seats"`
			FreeRentalBoards int `json:"free_rental_boards"`
		} `json:"slot"`
	}
	if err := json.Unmarshal(earlyRecorder.Body.Bytes(), &earlyResponse); err != nil {
		t.Fatalf("decode early response: %v", err)
	}
	if earlyResponse.Status != "cancelled" || earlyResponse.Slot.FreeSeats != 8 || earlyResponse.Slot.FreeRentalBoards != 12 {
		t.Fatalf("unexpected early response: %+v", earlyResponse)
	}
	repeatRecorder := performCancelBooking(router, token, earlyID)
	if repeatRecorder.Code != http.StatusConflict {
		t.Fatalf("repeat cancel status = %d, want %d", repeatRecorder.Code, http.StatusConflict)
	}

	lateID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	if _, err := db.Exec(ctx, `UPDATE slots SET start_at = $1, free_seats = 10, free_rental_boards = 9 WHERE id = '66666666-6666-6666-6666-666666666666'`, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("update late slot: %v", err)
	}
	insertBooking(t, ctx, db, lateID, "66666666-6666-6666-6666-666666666666", clientID, 2, 1)
	lateRecorder := performCancelBooking(router, token, lateID)
	if lateRecorder.Code != http.StatusOK {
		t.Fatalf("late cancel status = %d, body = %s", lateRecorder.Code, lateRecorder.Body.String())
	}
	var lateResponse struct {
		Status string `json:"status"`
		Slot   struct {
			FreeSeats        int `json:"free_seats"`
			FreeRentalBoards int `json:"free_rental_boards"`
		} `json:"slot"`
	}
	if err := json.Unmarshal(lateRecorder.Body.Bytes(), &lateResponse); err != nil {
		t.Fatalf("decode late response: %v", err)
	}
	if lateResponse.Status != "late_cancel" || lateResponse.Slot.FreeSeats != 10 || lateResponse.Slot.FreeRentalBoards != 9 {
		t.Fatalf("unexpected late response: %+v", lateResponse)
	}

	afterStartID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	if _, err := db.Exec(ctx, `UPDATE slots SET start_at = $1 WHERE id = '55555555-5555-5555-5555-555555555555'`, time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("update past slot: %v", err)
	}
	insertBooking(t, ctx, db, afterStartID, "55555555-5555-5555-5555-555555555555", clientID, 1, 0)
	afterStartRecorder := performCancelBooking(router, token, afterStartID)
	if afterStartRecorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("after start cancel status = %d, want %d", afterStartRecorder.Code, http.StatusUnprocessableEntity)
	}
}

func TestCancelBookingConcurrencyReturnsAvailabilityOnce(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	token := "cancel-token"
	clientID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	bookingID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	insertClientSession(t, ctx, db, clientID, "+79990005001", token)
	if _, err := db.Exec(ctx, `UPDATE slots SET free_seats = 6, free_rental_boards = 11 WHERE id = '55555555-5555-5555-5555-555555555555'`); err != nil {
		t.Fatalf("update slot: %v", err)
	}
	insertBooking(t, ctx, db, bookingID, "55555555-5555-5555-5555-555555555555", clientID, 2, 1)
	router := bookingRouter(db)

	const requests = 5
	var wg sync.WaitGroup
	statuses := make(chan int, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- performCancelBooking(router, token, bookingID).Code
		}()
	}
	wg.Wait()
	close(statuses)

	successes := 0
	conflicts := 0
	for status := range statuses {
		switch status {
		case http.StatusOK:
			successes++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected cancel status = %d", status)
		}
	}
	if successes != 1 || conflicts != requests-1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}

	var freeSeats, freeBoards int
	if err := db.QueryRow(ctx, `SELECT free_seats, free_rental_boards FROM slots WHERE id = '55555555-5555-5555-5555-555555555555'`).Scan(&freeSeats, &freeBoards); err != nil {
		t.Fatalf("read availability: %v", err)
	}
	if freeSeats != 8 || freeBoards != 12 {
		t.Fatalf("availability seats=%d boards=%d, want 8/12", freeSeats, freeBoards)
	}
}

func bookingRouter(db *pgxpool.Pool) http.Handler {
	service := booking.NewService(postgres.NewBookingRepository(db))
	return httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{Bookings: handlers.NewBookingHandler(service)})
}

func insertClientSession(t *testing.T, ctx context.Context, db *pgxpool.Pool, clientID, phone, token string) {
	t.Helper()
	if _, err := db.Exec(ctx, `INSERT INTO clients (id, phone) VALUES ($1, $2)`, clientID, phone); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO auth_sessions (client_id, token_hash, expires_at) VALUES ($1, $2, $3)`, clientID, auth.HashToken(token), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("insert session: %v", err)
	}
}

func performCreateBooking(router http.Handler, token, idempotencyKey, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", idempotencyKey)
	router.ServeHTTP(recorder, req)
	return recorder
}

func performCancelBooking(router http.Handler, token, bookingID string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID+"/cancel", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)
	return recorder
}

func insertBooking(t *testing.T, ctx context.Context, db *pgxpool.Pool, bookingID, slotID, clientID string, seatsCount, rentalCount int) {
	t.Helper()
	if _, err := db.Exec(ctx, `
INSERT INTO bookings (id, slot_id, client_id, seats_count, rental_count, status, created_at)
VALUES ($1, $2, $3, $4, $5, 'active', now())`, bookingID, slotID, clientID, seatsCount, rentalCount); err != nil {
		t.Fatalf("insert booking: %v", err)
	}
}
