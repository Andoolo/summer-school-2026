package postgres_test

import (
	"context"
	"testing"

	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
	"summer-school-2026/backend/migrations"
)

// Размер пула задаётся в DATABASE_URL (pool_max_conns). Миграции и сид ходят в базу мимо
// пула, и параметры пула не должны уходить серверу: иначе Postgres отказывает в
// подключении, и сервис с AUTO_MIGRATE не стартует.
func TestPoolParamsInDatabaseURL(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t) + "&pool_max_conns=7"

	if err := postgres.Migrate(databaseURL, migrations.FS); err != nil {
		t.Fatalf("Migrate() with pool_max_conns: %v", err)
	}
	if err := postgres.SeedSQL(databaseURL, "SELECT 1; SELECT 2"); err != nil {
		t.Fatalf("SeedSQL() with pool_max_conns: %v", err)
	}
	db, err := postgres.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("Connect() with pool_max_conns: %v", err)
	}
	defer db.Close()
	if got := db.Config().MaxConns; got != 7 {
		t.Fatalf("MaxConns = %d, want 7 from DATABASE_URL", got)
	}
}
