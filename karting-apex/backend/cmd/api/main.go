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
	"summer-school-2026/backend/migrations"
	"summer-school-2026/backend/seed"

	"github.com/jackc/pgx/v5/pgxpool"
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

	if cfg.AutoMigrate {
		if err := postgres.Migrate(cfg.DatabaseURL, migrations.FS); err != nil {
			logger.Error("failed to apply migrations", "error", err)
			os.Exit(1)
		}
		logger.Info("migrations applied")
	}

	if cfg.AutoSeed {
		if err := postgres.SeedSQL(cfg.DatabaseURL, seed.KartingLapResults); err != nil {
			logger.Error("failed to apply seed data", "error", err)
			os.Exit(1)
		}
		logger.Info("seed data applied")
	}

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	authRepo := postgres.NewAuthRepository(db)
	authService := auth.NewService(authRepo, logger)
	authHandler := handlers.NewAuthHandler(authService, cfg.Dev)

	var demoLogin http.HandlerFunc
	if cfg.DemoLogin {
		demoLogin = handlers.DemoLoginHandler(auth.NewDemoService(postgres.NewDemoRepository(db), auth.DefaultDemoConfig()), logger)
		go runDemoCleanup(ctx, db, logger)
		logger.Info("demo login enabled")
	}
	authMethods := handlers.AuthMethodsHandler(handlers.AuthMethods{SMS: cfg.Dev, Demo: cfg.DemoLogin})
	profileRepo := postgres.NewProfileRepository(db)
	profileService := profile.NewService(profileRepo, logger)
	profileHandler := handlers.NewProfileHandler(profileService)
	bookingService := booking.NewService(postgres.NewBookingRepository(db))
	bookingHandler := handlers.NewBookingHandler(bookingService)
	slotRepo := postgres.NewSlotRepository(db)
	slotHandler := handlers.NewSlotHandler(slotRepo)

	// Внесение результатов маршалом включается только явным MARSHAL_TOKEN.
	// Пусто — обработчик не создаётся и маршрут не регистрируется.
	var marshalLapResults, marshalRaceRoster http.HandlerFunc
	if cfg.MarshalToken != "" {
		marshalHandler := handlers.NewMarshalHandler(slotRepo, cfg.MarshalToken)
		marshalLapResults = marshalHandler.SubmitLapResults
		marshalRaceRoster = marshalHandler.RaceRoster
		logger.Info("marshal lap entry enabled")
	}
	instructorHandler := handlers.NewInstructorHandler(postgres.NewInstructorRepository(db))

	var rateLimit *httpapi.RateLimitOptions
	if cfg.RateLimit {
		rateLimit = &httpapi.RateLimitOptions{TrustProxy: cfg.TrustProxy, Logger: logger}
		logger.Info("rate limiting enabled", "trust_proxy", cfg.TrustProxy)
	}

	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewRouter(logger, httpapi.RouterOptions{
			Auth:              authHandler,
			AuthRefresh:       authHandler.Refresh,
			AuthDemo:          demoLogin,
			AuthMethods:       authMethods,
			Profile:           profileHandler,
			Bookings:          bookingHandler,
			Slots:             slotHandler,
			Instructors:       instructorHandler,
			RouteLeaderboard:  slotHandler.Leaderboard,
			RoutePassport:     slotHandler.TrackPassport,
			MarshalLapResults: marshalLapResults,
			MarshalRaceRoster: marshalRaceRoster,
			Dev:               cfg.Dev,
			AllowedOrigin:     cfg.AllowedOrigin,
			RateLimit:         rateLimit,
		}),
		// Таймауты на всё соединение, а не только на заголовки: иначе медленный клиент
		// (slowloris) держит соединение сколь угодно долго, отправляя тело по байту.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
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

// runDemoCleanup удаляет истёкших гостей при старте и затем раз в час. На бесплатном
// Render сервис засыпает, поэтому запуск при старте важнее расписания: он срабатывает
// при каждом пробуждении.
func runDemoCleanup(ctx context.Context, db *pgxpool.Pool, logger *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		for {
			removed, err := postgres.CleanupExpiredDemoClients(ctx, db, time.Now().UTC(), 200)
			if err != nil {
				if ctx.Err() == nil {
					logger.Error("demo cleanup failed", "error", err)
				}
				break
			}
			if removed > 0 {
				logger.Info("expired demo guests removed", "count", removed)
			}
			if removed < 200 {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
