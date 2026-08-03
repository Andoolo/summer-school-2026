package httpapi

import (
	"log/slog"
	"net/http"

	authapi "summer-school-2026/backend/internal/http/openapi/auth"
	bookingsapi "summer-school-2026/backend/internal/http/openapi/bookings"
	instructorsapi "summer-school-2026/backend/internal/http/openapi/instructors"
	profileapi "summer-school-2026/backend/internal/http/openapi/profile"
	slotsapi "summer-school-2026/backend/internal/http/openapi/slots"

	"github.com/go-chi/chi/v5"
)

type healthResponse struct {
	Status string `json:"status"`
}

type RouterOptions struct {
	Auth        authapi.ServerInterface
	Profile     profileapi.ServerInterface
	Bookings    bookingsapi.ServerInterface
	Slots       slotsapi.ServerInterface
	Instructors instructorsapi.ServerInterface
	// AuthRefresh — ручной маршрут POST /auth/refresh (обмен refresh-токена).
	// В сгенерированном транспорте auth его пока нет, поэтому регистрируется отдельно.
	AuthRefresh http.HandlerFunc
	// RouteLeaderboard — ручной маршрут GET /routes/{routeID}/leaderboard (рекорды трассы, картинг).
	RouteLeaderboard http.HandlerFunc
	// RoutePassport — ручной маршрут GET /routes/{routeID} (карточка трассы, картинг F5).
	RoutePassport http.HandlerFunc
	// MarshalLapResults — ручной маршрут POST /bookings/{bookingID}/lap-results
	// (внесение времён кругов маршалом, картинг F6). nil — маршрут не регистрируется
	// вовсе: без заданного MARSHAL_TOKEN функция должна быть недоступна, а не просто
	// отвечать отказом.
	MarshalLapResults http.HandlerFunc
	// Dev включает dev-удобства (например, разрешающий CORS для локального веб-клиента).
	// В production должен быть false (берётся из APP_ENV).
	Dev bool
	// AllowedOrigin — точный Origin, которому разрешён CORS, когда Dev=false (production).
	// В отличие от devCORSMiddleware (разрешает любой Origin), здесь — строгое сравнение
	// с одним заранее известным доменом фронтенда. Пусто — CORS в production выключен.
	AllowedOrigin string
}

func NewRouter(logger *slog.Logger, options ...RouterOptions) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	var opts RouterOptions
	if len(options) > 0 {
		opts = options[0]
	}

	router := chi.NewRouter()
	router.Use(requestIDMiddleware)
	router.Use(recoverMiddleware(logger))
	router.Use(accessLogMiddleware(logger))
	switch {
	case opts.Dev:
		// Только в dev: разрешающий CORS для локального веб-клиента (wasmJs на другом порту).
		router.Use(devCORSMiddleware)
	case opts.AllowedOrigin != "":
		// В production: CORS разрешён только заранее заданному домену фронтенда — без
		// отражения произвольного Origin, в отличие от dev-режима.
		router.Use(productionCORSMiddleware(opts.AllowedOrigin))
	}
	router.Use(jsonContentTypeMiddleware)
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotFound, CodeNotFound, "Запрашиваемый ресурс не найден.", nil)
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotFound, CodeNotFound, "Запрашиваемый ресурс не найден.", nil)
	})
	router.Get("/healthz", healthHandler)
	router.Get("/readyz", healthHandler)
	if opts.Auth != nil {
		authapi.HandlerWithOptions(opts.Auth, authapi.ChiServerOptions{BaseRouter: router, ErrorHandlerFunc: OpenAPIErrorHandler})
	}
	if opts.AuthRefresh != nil {
		router.Post("/auth/refresh", opts.AuthRefresh)
	}
	if opts.RouteLeaderboard != nil {
		router.Get("/routes/{routeID}/leaderboard", opts.RouteLeaderboard)
	}
	if opts.RoutePassport != nil {
		router.Get("/routes/{routeID}", opts.RoutePassport)
	}
	if opts.MarshalLapResults != nil {
		router.Post("/bookings/{bookingID}/lap-results", opts.MarshalLapResults)
	}
	if opts.Profile != nil {
		profileapi.HandlerWithOptions(opts.Profile, profileapi.ChiServerOptions{BaseRouter: router, ErrorHandlerFunc: OpenAPIErrorHandler})
	}
	if opts.Bookings != nil {
		bookingsapi.HandlerWithOptions(opts.Bookings, bookingsapi.ChiServerOptions{BaseRouter: router, ErrorHandlerFunc: OpenAPIErrorHandler})
	}
	if opts.Slots != nil {
		slotsapi.HandlerWithOptions(opts.Slots, slotsapi.ChiServerOptions{BaseRouter: router, ErrorHandlerFunc: OpenAPIErrorHandler})
	}
	if opts.Instructors != nil {
		instructorsapi.HandlerWithOptions(opts.Instructors, instructorsapi.ChiServerOptions{BaseRouter: router, ErrorHandlerFunc: OpenAPIErrorHandler})
	}

	return router
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// devCORSMiddleware разрешает cross-origin запросы локального веб-клиента (dev).
// Отражает Origin запроса и обрабатывает preflight OPTIONS. Подключается только при Dev=true.
func devCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// productionCORSMiddleware разрешает cross-origin запросы только с точно заданного Origin
// (домен задеплоенного фронтенда). В отличие от devCORSMiddleware, НЕ отражает произвольный
// Origin — запросы с любого другого источника не получат CORS-заголовков и будут заблокированы
// браузером.
func productionCORSMiddleware(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
