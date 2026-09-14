package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"summer-school-2026/backend/internal/ops"

	"github.com/go-chi/chi/v5"
)

// Observer — счётчики и алерты (ops.Recorder). nil — наблюдение выключено.
type Observer interface {
	Inc(name string)
	Alert(key, text string)
	CountAndAlert(key string, threshold int, window time.Duration, text func(count int) string)
}

type contextKey string

const requestIDKey contextKey = "request_id"

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}

		w.Header().Set("X-Request-Id", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return recoverObservedMiddleware(logger, nil)
}

func recoverObservedMiddleware(logger *slog.Logger, observer Observer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "panic", recovered, "request_id", RequestID(r.Context()))
					if observer != nil {
						observer.Inc(ops.Panics)
						// Само значение паники — только в журнал: в нём может оказаться что угодно.
						observer.Alert("panic:"+routePattern(r), fmt.Sprintf("🔥 Паника в %s %s: %s", r.Method, routePattern(r), ops.DescribePanic(recovered)))
					}
					WriteError(w, http.StatusInternalServerError, CodeInternalError, "Что-то пошло не так. Попробуйте ещё раз позже.", nil)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func accessLogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(recorder, r)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"request_id", RequestID(r.Context()),
			)
		})
	}
}

// observeMiddleware считает запросы, ответы 429 и 5xx; всплеск 5xx — алерт администратору.
func observeMiddleware(observer Observer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)

			observer.Inc(ops.HTTPRequests)
			switch {
			case recorder.status == http.StatusTooManyRequests:
				observer.Inc(ops.HTTPRateLimited)
			case recorder.status >= http.StatusInternalServerError:
				observer.Inc(ops.HTTPServerErrors)
				route := r.Method + " " + routePattern(r)
				observer.CountAndAlert("http_5xx", 5, 5*time.Minute, func(count int) string {
					return fmt.Sprintf("⚠️ API: %d ошибок 5xx за 5 минут. Последняя: %s → %d.", count, route, recorder.status)
				})
			}
		})
	}
}

// routePattern — шаблон маршрута (/bookings/{bookingId}), а не сам путь: в алерт не
// попадают id и прочие данные из адреса.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return "(маршрут не найден)"
}

func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
