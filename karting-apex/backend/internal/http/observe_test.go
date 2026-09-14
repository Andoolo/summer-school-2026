package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"summer-school-2026/backend/internal/ops"
)

type fakeObserver struct {
	mu     sync.Mutex
	counts map[string]int
	alerts []string
	window map[string]int
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{counts: map[string]int{}, window: map[string]int{}}
}

func (o *fakeObserver) Inc(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.counts[name]++
}

func (o *fakeObserver) Alert(key, text string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.alerts = append(o.alerts, key+"|"+text)
}

func (o *fakeObserver) CountAndAlert(key string, threshold int, _ time.Duration, text func(int) string) {
	o.mu.Lock()
	o.window[key]++
	count := o.window[key]
	o.mu.Unlock()
	if count >= threshold {
		o.Alert(key, text(count))
	}
}

func TestObserverCountsRequestsErrorsAndPanics(t *testing.T) {
	observer := newFakeObserver()
	router := NewRouter(nil, RouterOptions{
		Observer:         observer,
		RouteLeaderboard: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
		RoutePassport:    func(http.ResponseWriter, *http.Request) { panic("boom") },
	})
	serve := func(path string) int {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder.Code
	}

	serve("/healthz")
	for i := 0; i < 5; i++ {
		// id маршрута — личные данные адреса; в алерт попадает только шаблон.
		serve("/routes/secret-route-id-42/leaderboard")
	}
	if code := serve("/routes/secret-route-id-42"); code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, want 500", code)
	}

	if observer.counts[ops.HTTPRequests] != 7 || observer.counts[ops.HTTPServerErrors] != 6 || observer.counts[ops.Panics] != 1 {
		t.Fatalf("counts = %v", observer.counts)
	}
	joined := strings.Join(observer.alerts, "\n")
	if !strings.Contains(joined, "http_5xx|⚠️ API: 5 ошибок 5xx") || !strings.Contains(joined, "GET /routes/{routeID}/leaderboard") {
		t.Fatalf("5xx alert missing:\n%s", joined)
	}
	if !strings.Contains(joined, "Паника в GET /routes/{routeID}: boom") {
		t.Fatalf("panic alert missing:\n%s", joined)
	}
	if strings.Contains(joined, "secret-route-id-42") {
		t.Fatalf("alerts leak path data:\n%s", joined)
	}
}

func TestObserverCountsRateLimited(t *testing.T) {
	observer := newFakeObserver()
	router := NewRouter(nil, RouterOptions{
		Observer:  observer,
		AuthDemo:  func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		RateLimit: &RateLimitOptions{},
	})
	for i := 0; i < 8; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/demo", nil)
		req.RemoteAddr = "203.0.113.9:1234"
		router.ServeHTTP(httptest.NewRecorder(), req)
	}
	if observer.counts[ops.HTTPRateLimited] == 0 {
		t.Fatalf("429 responses not counted: %v", observer.counts)
	}
}
