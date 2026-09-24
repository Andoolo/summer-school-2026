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
	"summer-school-2026/backend/internal/ops"
	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/service/booking"
	"summer-school-2026/backend/internal/service/botactions"
	"summer-school-2026/backend/internal/service/botrouter"
	"summer-school-2026/backend/internal/service/notify"
	"summer-school-2026/backend/internal/service/profile"
	"summer-school-2026/backend/internal/service/telegramlogin"
	"summer-school-2026/backend/internal/service/waitlist"
	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/telegram"
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
		logger.Info("demo login enabled")
	}
	// Бот нужен и входу через Telegram, и алертам администратору.
	var botClient *telegram.Client
	var alertBot ops.Sender
	if cfg.TelegramBotToken != "" {
		botClient = telegram.NewClient(cfg.TelegramBotToken)
		if cfg.TelegramAPIBase != "" {
			if cfg.Dev {
				botClient.WithAPIBase(cfg.TelegramAPIBase)
				logger.Warn("telegram bot api base overridden for local testing")
			} else {
				logger.Error("TELEGRAM_API_BASE ignored in production")
			}
		}
		alertBot = botClient
	}
	if cfg.AdminTelegramChatID != 0 && alertBot == nil {
		logger.Warn("ADMIN_TELEGRAM_CHAT_ID is set, but TELEGRAM_BOT_TOKEN is empty: alerts go to logs only")
	}

	// Наблюдаемость: счётчики в базе, алерты администратору, сводка /stats.
	opsRepo := postgres.NewOpsRepository(db)
	recorder := ops.NewRecorder(opsRepo, alertBot, cfg.AdminTelegramChatID, logger)
	recorderDone := make(chan struct{})
	go func() {
		recorder.Run(ctx)
		close(recorderDone)
	}()
	// В фоне: недоступный Telegram не должен задерживать старт сервиса.
	go func() {
		if err := ops.AnnounceVersion(ctx, opsRepo, recorder, cfg.Version, time.Now()); err != nil && ctx.Err() == nil {
			logger.Error("announce version failed, will retry on next start", "error", err)
		}
	}()
	stats := ops.NewStats(opsRepo, recorder, cfg.Version)

	go runMaintenance(ctx, db, logger, cfg.DemoLogin, recorder)

	// Вход через Telegram включается токеном бота. Бот подключается в фоне; до этого
	// кнопка Telegram в приложении не показывается.
	telegramUsername := func() string { return "" }
	var telegramStart, telegramPoll, telegramWebhook http.HandlerFunc
	onBookingChange := func() {}
	var waitlistStatus, waitlistJoin, waitlistLeave, waitlistMine http.HandlerFunc
	if botClient != nil {
		webhookSecret := telegram.WebhookSecret(cfg.TelegramBotToken)
		connector := telegram.NewConnector(botClient, cfg.PublicURL, webhookSecret, logger).WithAdminChat(cfg.AdminTelegramChatID)
		go connector.Run(ctx)
		telegramUsername = connector.Username
		telegramRepo := postgres.NewTelegramLoginRepository(db)
		loginService := telegramlogin.NewService(telegramRepo, authService, botClient, connector.Username, logger)

		// Уведомления о бронях идут через того же бота. Слать можно и до подключения
		// вебхука: отправка сообщений от него не зависит.
		// Лист ожидания — там же: предложения мест рассылает тот же рассыльщик.
		waitlistRepo := postgres.NewWaitlistRepository(db)
		dispatcher := notify.NewDispatcher(postgres.NewNotificationRepository(db), botClient, logger).
			WithWaitlist(waitlistRepo, waitlist.OfferTTL, cfg.AllowedOrigin).
			WithObserver(recorder)
		go dispatcher.Run(ctx)
		onBookingChange = dispatcher.Wake
		// Кнопка «Отменить бронь» под уведомлениями: нажатия приходят в тот же вебхук.
		onBotCancel := func() {
			recorder.Inc(ops.BotCancellations)
			dispatcher.Wake()
		}
		cancelFromBot := botactions.NewService(postgres.NewBotActionsRepository(db), botClient, onBotCancel, logger)
		// Вебхук у бота один: вход, кнопки под уведомлениями и команды разбирает роутер.
		botRouter := botrouter.New(botrouter.Config{
			Login:         loginService,
			Callbacks:     cancelFromBot,
			Notifications: telegramRepo,
			Bot:           botClient,
			AdminChatID:   cfg.AdminTelegramChatID,
			Stats:         stats.Text,
			Logger:        logger,
		})
		telegramHandler := handlers.NewTelegramHandler(loginService, botRouter, webhookSecret, logger)
		telegramStart, telegramPoll, telegramWebhook = telegramHandler.Start, telegramHandler.Poll, telegramHandler.Webhook
		waitlistHandler := handlers.NewWaitlistHandler(waitlist.NewService(waitlistRepo, dispatcher.Wake), logger)
		waitlistStatus, waitlistJoin, waitlistLeave, waitlistMine = waitlistHandler.Status, waitlistHandler.Join, waitlistHandler.Leave, waitlistHandler.Mine
		logger.Info("booking notifications and waitlist enabled")
	}
	authMethods := handlers.AuthMethodsHandler(handlers.AuthMethods{SMS: cfg.Dev, Demo: cfg.DemoLogin, TelegramBotUsername: telegramUsername})
	profileRepo := postgres.NewProfileRepository(db)
	profileService := profile.NewService(profileRepo, logger)
	profileHandler := handlers.NewProfileHandler(profileService)
	bookingService := booking.NewService(postgres.NewBookingRepository(db)).WithOnChange(onBookingChange)
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
			TelegramStart:     telegramStart,
			TelegramPoll:      telegramPoll,
			TelegramWebhook:   telegramWebhook,
			Profile:           profileHandler,
			Bookings:          bookingHandler,
			Slots:             slotHandler,
			Instructors:       instructorHandler,
			RouteLeaderboard:  slotHandler.Leaderboard,
			RoutePassport:     slotHandler.TrackPassport,
			MarshalLapResults: marshalLapResults,
			MarshalRaceRoster: marshalRaceRoster,
			WaitlistStatus:    waitlistStatus,
			WaitlistJoin:      waitlistJoin,
			WaitlistLeave:     waitlistLeave,
			WaitlistMine:      waitlistMine,
			Dev:               cfg.Dev,
			AllowedOrigin:     cfg.AllowedOrigin,
			RateLimit:         rateLimit,
			Observer:          recorder,
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

	// Последний сброс счётчиков в базу — иначе события перед сном Render потерялись бы.
	select {
	case <-recorderDone:
	case <-time.After(6 * time.Second):
		logger.Warn("ops counters final flush timed out")
	}
	logger.Info("api server stopped")
}

// runMaintenance при старте и затем раз в час удаляет истёкших гостей и старые запросы
// входа через Telegram (в них номера телефонов). На бесплатном Render сервис засыпает,
// поэтому запуск при старте важнее расписания: он срабатывает при каждом пробуждении.
func runMaintenance(ctx context.Context, db *pgxpool.Pool, logger *slog.Logger, demoEnabled bool, recorder *ops.Recorder) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		if removed, err := postgres.DeleteStaleTelegramLogins(ctx, db, time.Now().UTC()); err != nil {
			if ctx.Err() == nil {
				logger.Error("telegram login cleanup failed", "error", err)
				recorder.Alert("maintenance_db", "⚠️ Уборка не может обратиться к базе: "+ops.DescribeError(err))
			}
		} else if removed > 0 {
			logger.Info("stale telegram login requests removed", "count", removed)
		}
		for demoEnabled {
			removed, err := postgres.CleanupExpiredDemoClients(ctx, db, time.Now().UTC(), 200)
			if err != nil {
				if ctx.Err() == nil {
					logger.Error("demo cleanup failed", "error", err)
					recorder.Alert("maintenance_demo", "⚠️ Уборка гостевых аккаунтов падает: "+ops.DescribeError(err))
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
		if _, err := postgres.DeleteOldCounters(ctx, db, time.Now().UTC()); err != nil && ctx.Err() == nil {
			logger.Error("ops counters cleanup failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
