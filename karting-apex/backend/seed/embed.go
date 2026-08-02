// Package seed встраивает демо-данные лидерборда в бинарник (go:embed), чтобы их можно было
// применить программно при старте — тем же надёжным путём, что и миграции (AUTO_MIGRATE),
// см. cmd/api/main.go и AUTO_SEED. Отдельно от миграций намеренно: демо-сид — не часть схемы
// и не должен попадать в тестовую БД (go test применяет только migrations/*.sql).
package seed

import _ "embed"

//go:embed karting_lap_results.sql
var KartingLapResults string
