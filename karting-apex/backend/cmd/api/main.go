package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"summer-school-2026/backend/internal/config"
	httpapi "summer-school-2026/backend/internal/http"
	"summer-school-2026/backend/internal/http/handlers"
	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/service/booking"
	"summer-school-2026/backend/internal/service/profile"
	"summer-school-2026/backend/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	authRepo := postgres.NewAuthRepository(db)
	authService := auth.NewService(authRepo, logger)
	authHandler := handlers.NewAuthHandler(authService, cfg.Dev)
	profileRepo := postgres.NewProfileRepository(db)
	profileService := profile.NewService(profileRepo, logger)
	profileHandler := handlers.NewProfileHandler(profileService)
	bookingService := booking.NewService(postgres.NewBookingRepository(db))
	bookingHandler := handlers.NewBookingHandler(bookingService)
	slotHandler := handlers.NewSlotHandler(postgres.NewSlotRepository(db))
	instructorHandler := handlers.NewInstructorHandler(postgres.NewInstructorRepository(db))

	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewRouter(logger, httpapi.RouterOptions{
			Auth:             authHandler,
			AuthRefresh:      authHandler.Refresh,
			Profile:          profileHandler,
			Bookings:         bookingHandler,
			Slots:            slotHandler,
			Instructors:      instructorHandler,
			RouteLeaderboard: slotHandler.Leaderboard,
			Dev:              cfg.Dev,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api server started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("api server stopped")
}
