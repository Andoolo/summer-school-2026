package postgres

import (
	"database/sql"
	"fmt"
	"strings"
)

// withSimpleProtocol просит pgx работать простым протоколом вместо расширенного.
//
// Нужно по двум причинам сразу:
//   - несколько разделённых точкой с запятой стейтментов выполняются за один Exec
//     (как это делает psql); расширенный протокол так не умеет;
//   - расширенный протокол кэширует подготовленные запросы, а управляемый Postgres
//     за прокси (Neon) может закрыть соединение между ними. Тогда pgx пытается
//     освободить уже недоступный запрос и падает с "failed to deallocate previously
//     failed statement ... unexpected EOF" — именно на этом не проходили миграции.
func withSimpleProtocol(databaseURL string) string {
	separator := "?"
	if strings.Contains(databaseURL, "?") {
		separator = "&"
	}
	return databaseURL + separator + "default_query_exec_mode=simple_protocol"
}

// SeedSQL выполняет произвольный многостейтментный SQL-скрипт (демо-данные) через
// отдельное подключение database/sql.
func SeedSQL(databaseURL, sqlText string) error {
	db, err := sql.Open("pgx", withSimpleProtocol(databaseURL))
	if err != nil {
		return fmt.Errorf("open db for seed: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(sqlText); err != nil {
		return fmt.Errorf("apply seed: %w", err)
	}
	return nil
}
