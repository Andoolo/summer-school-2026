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
