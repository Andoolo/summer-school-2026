package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	ShutdownTimeout time.Duration
	// Dev true в окружениях, отличных от production.
	// В dev разрешено возвращать OTP-код в ответе /auth/request-code (для ручной проверки
	// без реальной отправки SMS). В production код никогда не возвращается клиенту.
	Dev bool
	// AllowedOrigin — точный Origin фронтенда, которому разрешён CORS в production
	// (например, https://apex.onrender.com). Пусто — CORS в production не включается.
	// В dev это не используется: там разрешён любой Origin (devCORSMiddleware).
	AllowedOrigin string
	// AutoMigrate включает применение миграций программно при старте сервиса
	// (через то же DATABASE_URL, что и всё приложение). По умолчанию выключено —
	// локальный workflow (make migrate) не меняется. Включается явно (AUTO_MIGRATE=true)
	// для окружений вроде Render, где внешний доступ к БД для ручных миграций недоступен/нестабилен.
	AutoMigrate bool
	// AutoSeed включает применение демо-данных лидерборда (seed.KartingLapResults) программно
	// при старте, тем же путём, что и AutoMigrate. Отдельный флаг: сид — не часть схемы, не
	// нужен на каждом окружении (например, в тестах — только миграции).
	AutoSeed bool
	// MarshalToken — общий секрет для внесения времён кругов маршалом (F6).
	// ПУСТО ОЗНАЧАЕТ, ЧТО ФУНКЦИЯ ВЫКЛЮЧЕНА ЦЕЛИКОМ: эндпоинт не регистрируется.
	// Так открытую запись в чужие результаты нельзя оставить по недосмотру —
	// её нужно включить осознанно, задав переменную окружения.
	MarshalToken string
	// RateLimit — лимиты запросов с одного IP. По умолчанию включены в production и
	// выключены в dev (там k6 и ручные проверки шлют всё с одного адреса). RATE_LIMIT
	// = on/off переопределяет умолчание.
	RateLimit bool
	// TrustProxy — брать IP клиента из CF-Connecting-IP (Render стоит за Cloudflare).
	// Включается сам на Render (платформа всегда выставляет RENDER=true) или явно
	// TRUST_PROXY=true. Без Cloudflare включать нельзя: заголовок подделывается.
	TrustProxy bool
	// DemoLogin — гостевой вход без регистрации. Включён по умолчанию, DEMO_LOGIN=off
	// выключает.
	DemoLogin bool
	// TelegramBotToken — токен бота для входа через Telegram. Секрет: задаётся только в
	// окружении, в репозиторий не попадает. Пусто — вход через Telegram выключен.
	TelegramBotToken string
	// PublicURL — внешний адрес сервиса для вебхука Telegram. На Render подставляется
	// сам (RENDER_EXTERNAL_URL), PUBLIC_URL переопределяет.
	PublicURL string
	// TelegramAPIBase — адрес Bot API вместо официального, для локальной проверки с
	// имитатором Telegram. Действует только вне production: иначе ошибочная переменная
	// отправила бы токен бота на чужой сервер.
	TelegramAPIBase string
	// AdminTelegramChatID — чат администратора: алерты и команда /stats. Свой chat id
	// бот присылает по команде /whoami. 0 — алерты только в журнал.
	AdminTelegramChatID int64
	// Version — версия (коммит) для сводки и сообщения о выкладке. На Render —
	// RENDER_GIT_COMMIT, APP_VERSION переопределяет.
	Version string
}

func Load() (Config, error) {
	shutdownTimeout, err := durationFromEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	dev := boolFromEnvIsNot("APP_ENV", "production")
	rateLimit := !dev
	switch os.Getenv("RATE_LIMIT") {
	case "on":
		rateLimit = true
	case "off":
		rateLimit = false
	}

	adminChat, err := int64FromEnv("ADMIN_TELEGRAM_CHAT_ID")
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:         stringFromEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:      stringFromEnv("DATABASE_URL", "postgres://volna:volna@localhost:5432/volna?sslmode=disable"),
		ShutdownTimeout:  shutdownTimeout,
		Dev:              dev,
		AllowedOrigin:    stringFromEnv("ALLOWED_ORIGIN", ""),
		AutoMigrate:      os.Getenv("AUTO_MIGRATE") == "true",
		AutoSeed:         os.Getenv("AUTO_SEED") == "true",
		MarshalToken:     stringFromEnv("MARSHAL_TOKEN", ""),
		RateLimit:        rateLimit,
		TrustProxy:       os.Getenv("TRUST_PROXY") == "true" || os.Getenv("RENDER") == "true",
		DemoLogin:        os.Getenv("DEMO_LOGIN") != "off",
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		PublicURL:        stringFromEnv("PUBLIC_URL", os.Getenv("RENDER_EXTERNAL_URL")),
		TelegramAPIBase:  os.Getenv("TELEGRAM_API_BASE"),

		AdminTelegramChatID: adminChat,
		Version:             stringFromEnv("APP_VERSION", os.Getenv("RENDER_GIT_COMMIT")),
	}, nil
}

func stringFromEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// boolFromEnvIsNot возвращает true, если значение env-переменной key не равно forbidden.
// Используется для флага Dev: всё, что не "production" — считаем dev (безопасный дефолт).
func boolFromEnvIsNot(key, forbidden string) bool {
	return os.Getenv(key) != forbidden
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer number of seconds", key)
	}

	return time.Duration(seconds) * time.Second, nil
}

func int64FromEnv(key string) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
