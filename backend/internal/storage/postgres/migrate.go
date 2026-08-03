package postgres

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует database/sql-драйвер "pgx" для goose
)

func Migrate(databaseURL string, migrationsFS embed.FS) error {
	// Простой протокол — см. withSimpleProtocol: с кэшем подготовленных запросов
	// миграции не проходят на управляемом Postgres за прокси.
	db, err := sql.Open("pgx", withSimpleProtocol(databaseURL))
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
