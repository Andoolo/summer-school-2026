package postgres

import (
	"database/sql"
	"fmt"
	"strings"
)

// SeedSQL выполняет произвольный многостейтментный SQL-скрипт (демо-данные) через отдельное
// подключение database/sql. simple_protocol нужен, чтобы pgx выполнил несколько
// разделённых точкой с запятой стейтментов за один Exec (как это делает psql по умолчанию) —
// стандартный extended-протокол pgx такое не поддерживает.
func SeedSQL(databaseURL, sqlText string) error {
	connString := databaseURL
	if strings.Contains(connString, "?") {
		connString += "&default_query_exec_mode=simple_protocol"
	} else {
		connString += "?default_query_exec_mode=simple_protocol"
	}

	db, err := sql.Open("pgx", connString)
	if err != nil {
		return fmt.Errorf("open db for seed: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(sqlText); err != nil {
		return fmt.Errorf("apply seed: %w", err)
	}
	return nil
}
