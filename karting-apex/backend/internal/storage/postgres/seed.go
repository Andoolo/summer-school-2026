package postgres

import (
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// openScriptDB открывает отдельное (не-pool) подключение database/sql для миграций и сида.
//
// Строка разбирается так же, как для пула: параметры пула (pool_max_conns и другие pool_*)
// понимает только pgxpool. Обычное подключение отправило бы их серверу как настройки
// сеанса, и Postgres отказал бы в подключении («unrecognized configuration parameter») —
// сервис с AUTO_MIGRATE не стартовал бы из-за размера пула в DATABASE_URL.
//
// Работает простым протоколом вместо расширенного. Нужно по двум причинам сразу:
//   - несколько разделённых точкой с запятой стейтментов выполняются за один Exec
//     (как это делает psql); расширенный протокол так не умеет;
//   - расширенный протокол кэширует подготовленные запросы, а управляемый Postgres
//     за прокси (Neon) может закрыть соединение между ними. Тогда pgx пытается
//     освободить уже недоступный запрос и падает с "failed to deallocate previously
//     failed statement ... unexpected EOF" — именно на этом не проходили миграции.
func openScriptDB(databaseURL string) (*sql.DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	connConfig := config.ConnConfig
	connConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return stdlib.OpenDB(*connConfig), nil
}

// SeedSQL выполняет произвольный многостейтментный SQL-скрипт (демо-данные) через
// отдельное подключение database/sql.
func SeedSQL(databaseURL, sqlText string) error {
	db, err := openScriptDB(databaseURL)
	if err != nil {
		return fmt.Errorf("open db for seed: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(sqlText); err != nil {
		return fmt.Errorf("apply seed: %w", err)
	}
	return nil
}
