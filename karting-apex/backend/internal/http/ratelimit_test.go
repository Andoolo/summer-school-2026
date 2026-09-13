package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWindowLimiterBlocksOverLimitAndResets(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	limiter := newWindowLimiter(3, time.Minute)
	limiter.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := limiter.allow("1.2.3.4"); !ok {
			t.Fatalf("request %d must be allowed", i+1)
		}
	}
	ok, retryAfter := limiter.allow("1.2.3.4")
	if ok {
		t.Fatal("4th request must be blocked")
	}
	if retryAfter != time.Minute {
		t.Fatalf("retryAfter = %v, want %v", retryAfter, time.Minute)
	}
	if ok, _ := limiter.allow("5.6.7.8"); !ok {
		t.Fatal("another client must have its own limit")
	}

	now = now.Add(time.Minute)
	if ok, _ := limiter.allow("1.2.3.4"); !ok {
		t.Fatal("limit must reset after the window")
	}
}

func TestWindowLimiterSweepsAndCapsKeys(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	limiter := newWindowLimiter(1, time.Minute)
	limiter.now = func() time.Time { return now }
	limiter.maxKeys = 2

	limiter.allow("a")
	limiter.allow("b")
	limiter.allow("c") // таблица полна — сбрасывается, а не блокирует
	if len(limiter.entries) != 1 {
		t.Fatalf("entries = %d, want 1 after overflow reset", len(limiter.entries))
	}

	now = now.Add(2 * time.Minute)
	limiter.allow("d")
	if _, ok := limiter.entries["c"]; ok {
		t.Fatal("expired entries must be swept")
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/slots", nil)
	req.RemoteAddr = "10.0.0.1:5555"
	// Так выглядит запрос с подделанным X-Forwarded-For на Render: подделка первой,
	// настоящий адрес — в CF-Connecting-IP.
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 198.51.100.20, 172.64.0.1, 10.1.1.1")
	req.Header.Set("CF-Connecting-IP", "198.51.100.20")

	if got := clientIP(req, false); got != "10.0.0.1" {
		t.Fatalf("without proxy trust got %q, want socket address", got)
	}
	if got := clientIP(req, true); got != "198.51.100.20" {
		t.Fatalf("with proxy trust got %q, want CF-Connecting-IP, not spoofed forwarded address", got)
	}

	req.Header.Del("CF-Connecting-IP")
	if got := clientIP(req, true); got != "10.0.0.1" {
		t.Fatalf("without CF-Connecting-IP got %q, want socket address (X-Forwarded-For is spoofable)", got)
	}

	req.Header.Set("CF-Connecting-IP", "2001:db8:1:2:aaaa:bbbb:cccc:dddd")
	if got := clientIP(req, true); got != "2001:db8:1:2::/64" {
		t.Fatalf("ipv6 got %q, want /64 prefix", got)
	}
}

func TestRateLimitMiddlewareLimitsRequestCodePerIP(t *testing.T) {
	handler := rateLimitMiddleware(RateLimitOptions{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
	)
	send := func(method, path, ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	for i := 0; i < 10; i++ {
		if rec := send(http.MethodPost, "/auth/request-code", "198.51.100.1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i+1, rec.Code)
		}
	}
	blocked := send(http.MethodPost, "/auth/request-code", "198.51.100.1")
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("11th request-code status = %d, want 429", blocked.Code)
	}
	if blocked.Header().Get("Retry-After") == "" {
		t.Fatal("429 must carry Retry-After")
	}
	if !strings.Contains(blocked.Body.String(), CodeTooManyRequests) {
		t.Fatalf("body = %s, want error code %s", blocked.Body.String(), CodeTooManyRequests)
	}

	// Лимит на запрос кода не должен задевать остальное API и других клиентов.
	if rec := send(http.MethodGet, "/slots", "198.51.100.1"); rec.Code != http.StatusOK {
		t.Fatalf("general route status = %d, want 200", rec.Code)
	}
	if rec := send(http.MethodPost, "/auth/request-code", "198.51.100.2"); rec.Code != http.StatusOK {
		t.Fatalf("other client status = %d, want 200", rec.Code)
	}
	// Проверки живости не ограничиваются.
	for i := 0; i < 400; i++ {
		if rec := send(http.MethodGet, "/healthz", "198.51.100.3"); rec.Code != http.StatusOK {
			t.Fatalf("healthz must never be limited, got %d on %d", rec.Code, i+1)
		}
	}
}

func TestBodyLimitMiddlewareRejectsLargeBody(t *testing.T) {
	var readErr error
	handler := bodyLimitMiddleware(16)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))
	req := httptest.NewRequest(http.MethodPost, "/bookings", strings.NewReader(strings.Repeat("x", 64)))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if readErr == nil {
		t.Fatal("reading body over the limit must fail")
	}
}
