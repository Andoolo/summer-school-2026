package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/http/handlers"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
)

const marshalToken = "test-marshal-token"

// Внесение времён кругов маршалом (F6). Проверки идут вокруг одного риска:
// испорченная таблица рекордов. Опечатка или запись не в ту бронь остаются
// в системе навсегда, поэтому правила фиксируются тестами.
func TestMarshalLapResults(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	repo := postgres.NewSlotRepository(db)
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{
		Slots:             handlers.NewSlotHandler(repo),
		RouteLeaderboard:  handlers.NewSlotHandler(repo).Leaderboard,
		MarshalLapResults: handlers.NewMarshalHandler(repo, marshalToken).SubmitLapResults,
		MarshalRaceRoster: handlers.NewMarshalHandler(repo, marshalToken).RaceRoster,
	})

	pastBooking := seedRace(t, ctx, db, raceFixture{
		clientID:  "e1000000-0000-4000-8000-000000000001",
		phone:     "+79995000001",
		name:      "Пилот Прошлый",
		slotID:    "e2000000-0000-4000-8000-000000000001",
		bookingID: "e3000000-0000-4000-8000-000000000001",
		startAt:   time.Now().Add(-2 * time.Hour),
		status:    "active",
	})

	t.Run("без токена доступ закрыт", func(t *testing.T) {
		r := submitLaps(router, "", pastBooking, `{"lap_times_ms":[42000]}`)
		if r.Code != http.StatusForbidden {
			t.Fatalf("статус = %d, ожидали 403; тело = %s", r.Code, r.Body.String())
		}
	})

	t.Run("с чужим токеном доступ закрыт", func(t *testing.T) {
		r := submitLaps(router, "не-тот-токен", pastBooking, `{"lap_times_ms":[42000]}`)
		if r.Code != http.StatusForbidden {
			t.Fatalf("статус = %d, ожидали 403", r.Code)
		}
	})

	t.Run("результаты сохраняются и попадают в рекорды", func(t *testing.T) {
		r := submitLaps(router, marshalToken, pastBooking, `{"lap_times_ms":[44120,42318,42990]}`)
		if r.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело = %s", r.Code, r.Body.String())
		}

		var resp struct {
			Laps          int    `json:"laps"`
			BestLapMs     int    `json:"best_lap_ms"`
			RacerName     string `json:"racer_name"`
			IsTrackRecord bool   `json:"is_track_record"`
		}
		decode(t, r, &resp)

		if resp.Laps != 3 {
			t.Errorf("кругов = %d, ожидали 3", resp.Laps)
		}
		if resp.BestLapMs != 42318 {
			t.Errorf("лучший круг = %d, ожидали 42318", resp.BestLapMs)
		}
		if resp.RacerName != "Пилот Прошлый" {
			t.Errorf("гонщик = %q", resp.RacerName)
		}
		if !resp.IsTrackRecord {
			t.Error("первый результат на трассе обязан считаться рекордом")
		}
	})

	t.Run("повторная отправка заменяет, а не добавляет круги", func(t *testing.T) {
		// Маршал исправляет опечатку. Если бы круги добавлялись, таблица рекордов
		// показала бы удвоенное число кругов и старое ошибочное время.
		r := submitLaps(router, marshalToken, pastBooking, `{"lap_times_ms":[41500,41890]}`)
		if r.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело = %s", r.Code, r.Body.String())
		}

		var laps, best int
		if err := db.QueryRow(ctx,
			`SELECT count(*), MIN(lap_time_ms) FROM lap_results WHERE booking_id = $1`,
			pastBooking).Scan(&laps, &best); err != nil {
			t.Fatalf("прочитать круги: %v", err)
		}
		if laps != 2 {
			t.Errorf("кругов в базе = %d, ожидали 2 — прежние должны были замениться", laps)
		}
		if best != 41500 {
			t.Errorf("лучший круг в базе = %d, ожидали 41500", best)
		}
	})

	t.Run("неправдоподобное время отклоняется", func(t *testing.T) {
		for name, body := range map[string]string{
			"секунды вместо миллисекунд": `{"lap_times_ms":[42]}`,
			"лишний ноль":                `{"lap_times_ms":[4231800]}`,
			"пустой список":              `{"lap_times_ms":[]}`,
		} {
			t.Run(name, func(t *testing.T) {
				r := submitLaps(router, marshalToken, pastBooking, body)
				if r.Code != http.StatusBadRequest {
					t.Errorf("статус = %d, ожидали 400; тело = %s", r.Code, r.Body.String())
				}
			})
		}
	})

	t.Run("для будущего заезда результаты не принимаются", func(t *testing.T) {
		future := seedRace(t, ctx, db, raceFixture{
			clientID:  "e1000000-0000-4000-8000-000000000002",
			phone:     "+79995000002",
			name:      "Пилот Будущий",
			slotID:    "e2000000-0000-4000-8000-000000000002",
			bookingID: "e3000000-0000-4000-8000-000000000002",
			startAt:   time.Now().Add(48 * time.Hour),
			status:    "active",
		})

		r := submitLaps(router, marshalToken, future, `{"lap_times_ms":[42000]}`)
		if r.Code != http.StatusConflict {
			t.Fatalf("статус = %d, ожидали 409: заезд ещё не состоялся", r.Code)
		}
	})

	t.Run("для отменённой записи результаты не принимаются", func(t *testing.T) {
		cancelled := seedRace(t, ctx, db, raceFixture{
			clientID:  "e1000000-0000-4000-8000-000000000003",
			phone:     "+79995000003",
			name:      "Пилот Отменённый",
			slotID:    "e2000000-0000-4000-8000-000000000003",
			bookingID: "e3000000-0000-4000-8000-000000000003",
			startAt:   time.Now().Add(-3 * time.Hour),
			status:    "cancelled",
		})

		r := submitLaps(router, marshalToken, cancelled, `{"lap_times_ms":[42000]}`)
		if r.Code != http.StatusConflict {
			t.Fatalf("статус = %d, ожидали 409: запись отменена", r.Code)
		}
	})

	t.Run("несуществующая бронь", func(t *testing.T) {
		r := submitLaps(router, marshalToken, "e3000000-0000-4000-8000-00000000dead", `{"lap_times_ms":[42000]}`)
		if r.Code != http.StatusNotFound {
			t.Fatalf("статус = %d, ожидали 404", r.Code)
		}
	})
}

// Без MARSHAL_TOKEN маршрут не должен существовать вовсе — это защита от того,
// чтобы забыть выключить внесение результатов в production.
func TestMarshalRouteAbsentWithoutToken(t *testing.T) {
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/bookings/e3000000-0000-4000-8000-000000000001/lap-results",
		bytes.NewBufferString(`{"lap_times_ms":[42000]}`))
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound && recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("статус = %d, ожидали отсутствие маршрута", recorder.Code)
	}
}

// Список участников заезда: маршал выбирает гонщика из него, а не вводит
// идентификатор брони руками.
func TestMarshalRaceRoster(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	repo := postgres.NewSlotRepository(db)
	handler := handlers.NewMarshalHandler(repo, marshalToken)
	router := httpapi.NewRouter(slog.Default(), httpapi.RouterOptions{
		MarshalRaceRoster: handler.RaceRoster,
		MarshalLapResults: handler.SubmitLapResults,
	})

	slotID := "e2000000-0000-4000-8000-0000000000a1"
	booking := seedRace(t, ctx, db, raceFixture{
		clientID:  "e1000000-0000-4000-8000-0000000000a1",
		phone:     "+79995001001",
		name:      "Гонщик Первый",
		slotID:    slotID,
		bookingID: "e3000000-0000-4000-8000-0000000000a1",
		startAt:   time.Now().Add(-4 * time.Hour),
		status:    "active",
	})
	// Отменённая запись на том же заезде: гонщик не выходил на трассу.
	seedExtraBooking(t, ctx, db, slotID, "e1000000-0000-4000-8000-0000000000a2",
		"+79995001002", "Гонщик Отменённый", "e3000000-0000-4000-8000-0000000000a2", "cancelled")

	t.Run("без токена доступ закрыт", func(t *testing.T) {
		r := getRoster(router, "", slotID)
		if r.Code != http.StatusForbidden {
			t.Fatalf("статус = %d, ожидали 403", r.Code)
		}
	})

	t.Run("отменённые записи в список не попадают", func(t *testing.T) {
		r := getRoster(router, marshalToken, slotID)
		if r.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело = %s", r.Code, r.Body.String())
		}
		var resp struct {
			AlreadyRaced bool `json:"already_raced"`
			Participants []struct {
				RacerName string `json:"racer_name"`
				Laps      int    `json:"laps"`
				BestLapMs *int   `json:"best_lap_ms"`
			} `json:"participants"`
		}
		decode(t, r, &resp)

		if len(resp.Participants) != 1 {
			t.Fatalf("участников = %d, ожидали 1: отменённая запись не должна попадать", len(resp.Participants))
		}
		if resp.Participants[0].RacerName != "Гонщик Первый" {
			t.Errorf("гонщик = %q", resp.Participants[0].RacerName)
		}
		if resp.Participants[0].Laps != 0 || resp.Participants[0].BestLapMs != nil {
			t.Errorf("до внесения результатов кругов быть не должно: %+v", resp.Participants[0])
		}
		if !resp.AlreadyRaced {
			t.Error("заезд в прошлом — already_raced должен быть true")
		}
	})

	t.Run("после внесения результаты видны в списке", func(t *testing.T) {
		if r := submitLaps(router, marshalToken, booking, `{"lap_times_ms":[43000,42100]}`); r.Code != http.StatusOK {
			t.Fatalf("внести круги: статус %d, тело %s", r.Code, r.Body.String())
		}

		r := getRoster(router, marshalToken, slotID)
		var resp struct {
			Participants []struct {
				Laps      int  `json:"laps"`
				BestLapMs *int `json:"best_lap_ms"`
			} `json:"participants"`
		}
		decode(t, r, &resp)

		if resp.Participants[0].Laps != 2 {
			t.Errorf("кругов = %d, ожидали 2", resp.Participants[0].Laps)
		}
		if resp.Participants[0].BestLapMs == nil || *resp.Participants[0].BestLapMs != 42100 {
			t.Errorf("лучший круг = %v, ожидали 42100", resp.Participants[0].BestLapMs)
		}
	})

	t.Run("несуществующий заезд", func(t *testing.T) {
		r := getRoster(router, marshalToken, "e2000000-0000-4000-8000-00000000dead")
		if r.Code != http.StatusNotFound {
			t.Fatalf("статус = %d, ожидали 404", r.Code)
		}
	})
}

func seedExtraBooking(t *testing.T, ctx context.Context, db *pgxpool.Pool, slotID, clientID, phone, name, bookingID, status string) {
	t.Helper()
	if _, err := db.Exec(ctx, `INSERT INTO clients (id, phone, name) VALUES ($1, $2, $3)`, clientID, phone, name); err != nil {
		t.Fatalf("вставить клиента: %v", err)
	}
	var cancelledAt *time.Time
	if status != "active" {
		now := time.Now()
		cancelledAt = &now
	}
	if _, err := db.Exec(ctx, `
INSERT INTO bookings (id, slot_id, client_id, seats_count, rental_count, status, created_at, cancelled_at)
VALUES ($1, $2, $3, 1, 0, $4, now(), $5)`, bookingID, slotID, clientID, status, cancelledAt); err != nil {
		t.Fatalf("вставить бронь: %v", err)
	}
}

func getRoster(router http.Handler, token, slotID string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/slots/%s/participants", slotID), nil)
	if token != "" {
		req.Header.Set("X-Marshal-Token", token)
	}
	router.ServeHTTP(recorder, req)
	return recorder
}

type raceFixture struct {
	clientID  string
	phone     string
	name      string
	slotID    string
	bookingID string
	startAt   time.Time
	status    string
}

// seedRace создаёт клиента, слот и бронь под один сценарий и возвращает id брони.
func seedRace(t *testing.T, ctx context.Context, db *pgxpool.Pool, f raceFixture) string {
	t.Helper()

	if _, err := db.Exec(ctx,
		`INSERT INTO clients (id, phone, name) VALUES ($1, $2, $3)`,
		f.clientID, f.phone, f.name); err != nil {
		t.Fatalf("вставить клиента: %v", err)
	}
	if _, err := db.Exec(ctx, `
INSERT INTO slots (id, route_id, instructor_id, start_at, total_seats, free_seats,
                   rental_boards_total, free_rental_boards, price, rental_price,
                   meeting_point, meeting_point_lat, meeting_point_lng, status)
VALUES ($1, '11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333',
        $2, 8, 7, 12, 12, 2500, 800, 'Главный бокс', 55.75, 37.61, 'scheduled')`,
		f.slotID, f.startAt); err != nil {
		t.Fatalf("вставить слот: %v", err)
	}
	// Схема требует, чтобы у отменённой брони было проставлено время отмены.
	var cancelledAt *time.Time
	if f.status != "active" {
		now := time.Now()
		cancelledAt = &now
	}
	if _, err := db.Exec(ctx, `
INSERT INTO bookings (id, slot_id, client_id, seats_count, rental_count, status, created_at, cancelled_at)
VALUES ($1, $2, $3, 1, 0, $4, now(), $5)`,
		f.bookingID, f.slotID, f.clientID, f.status, cancelledAt); err != nil {
		t.Fatalf("вставить бронь: %v", err)
	}
	return f.bookingID
}

func submitLaps(router http.Handler, token, bookingID, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/bookings/%s/lap-results", bookingID),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Marshal-Token", token)
	}
	router.ServeHTTP(recorder, req)
	return recorder
}

func decode(t *testing.T, r *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(r.Body.Bytes(), target); err != nil {
		t.Fatalf("разобрать ответ: %v; тело = %s", err, r.Body.String())
	}
}
