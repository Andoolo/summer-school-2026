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
}

func Load() (Config, error) {
	shutdownTimeout, err := durationFromEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:        stringFromEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:     stringFromEnv("DATABASE_URL", "postgres://volna:volna@localhost:5432/volna?sslmode=disable"),
		ShutdownTimeout: shutdownTimeout,
		Dev:             boolFromEnvIsNot("APP_ENV", "production"),
		AllowedOrigin:   stringFromEnv("ALLOWED_ORIGIN", ""),
		AutoMigrate:     os.Getenv("AUTO_MIGRATE") == "true",
		AutoSeed:        os.Getenv("AUTO_SEED") == "true",
		MarshalToken:    stringFromEnv("MARSHAL_TOKEN", ""),
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
