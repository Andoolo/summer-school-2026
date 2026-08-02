package postgres

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует database/sql-драйвер "pgx" для goose
)

// Migrate применяет непринятые миграции из migrationsFS к базе по databaseURL через
// отдельное (не-pool) подключение database/sql. Используется при старте сервиса — так
// миграции идут через то же DATABASE_URL, что и всё приложение, и в средах вроде Render
// выполняются через быструю внутреннюю сеть, а не через внешнее подключение.
func Migrate(databaseURL string, migrationsFS embed.FS) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
