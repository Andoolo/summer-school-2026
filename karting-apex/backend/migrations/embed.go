// Package migrations встраивает SQL-файлы миграций в бинарник (go:embed), чтобы их можно
// было применить программно при старте сервиса — без внешнего инструмента goose CLI и без
// сетевого доступа к go-module-proxy в среде исполнения. См. cmd/api/main.go и AUTO_MIGRATE.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
