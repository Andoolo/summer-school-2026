package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func PrepareDatabase(t *testing.T) string {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	schemaName := fmt.Sprintf("test_%d", time.Now().UnixNano())
	ctx := context.Background()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(ctx, `CREATE SCHEMA `+schemaName); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schemaName+` CASCADE`)
	})

	if _, err := db.ExecContext(ctx, `SET search_path TO `+schemaName); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, db, migrationsDir(t)); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return databaseURLWithSearchPath(t, databaseURL, schemaName)
}

func migrationsDir(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if parent := filepath.Dir(dir); parent == dir {
			break
		}
	}

	t.Fatal("migrations directory not found")
	return ""
}

func databaseURLWithSearchPath(t *testing.T, databaseURL, schemaName string) string {
	t.Helper()

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}

	query := parsed.Query()
	if _, ok := query["sslmode"]; !ok && !strings.Contains(databaseURL, "sslmode=") {
		query.Set("sslmode", "disable")
	}
	query.Set("search_path", schemaName)
	parsed.RawQuery = query.Encode()

	return parsed.String()
}
