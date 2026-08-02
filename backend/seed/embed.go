// Package seed встраивает демо-данные («мок» состояния клиентского приложения) в бинарник
// (go:embed), чтобы их можно было применить программно при старте — тем же надёжным путём,
// что и миграции (AUTO_MIGRATE), см. cmd/api/main.go и AUTO_SEED. Отдельно от миграций
// намеренно: демо-сид — не часть схемы и не должен попадать в тестовую БД (go test применяет
// только migrations/*.sql).
package seed

import _ "embed"

//go:embed mock_client_app_states.sql
var MockClientAppStates string
