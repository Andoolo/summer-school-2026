# Летняя школа Surf — сводка задания (Андрей Белявцев)

Домен по брифу: **Картинг-центр**. Демонстрация всех фреймворков (Анализ → Разработка →
Тестирование) с ИИ-ассистентами.

## Про домен: Картинг (анализ) + Волна (разработка/тестирование)

Организаторы в чате разрешили использовать референс-репозиторий «Волна» как шаблон
(«можете в качестве шаблона использовать репозиторий (для упрощения)», «важнее умение промптить и
работать с ИИ, а что делать будете — не так важно»). Поэтому:

- **Блок 1 (Анализ)** выполнен по выбранному домену **«Картинг-центр»** — отдельный репозиторий
  `apex-karting` (требования, архитектура, схема данных, дизайн-брифы).
- **Блоки 2–3 (Разработка + Тестирование)** выполнены на **референсе «Волна»** (этот репозиторий) —
  чтобы работать на реальной, полной кодовой базе (Go backend + Kotlin Multiplatform клиент).

## Соответствие задания и артефактов

| Задача задания | Где |
|----------------|-----|
| 1. Требования MVP (user stories + сценарии) | `apex-karting/01-analysis` (Картинг) · `01-analysis/2-requirements` (Волна) |
| 2. Архитектурный план + схема данных | `apex-karting/01-analysis` · `01-analysis/4-design/data-model.md` |
| 3. Реализовать ≥3 фичи | `02-development/tasks/F1,F2,F3` (route.geometry, карта, auth/refresh) |
| 4. Тест-кейсы + найти/исправить 1–3 бага | `03-testing/` (кейсы) · `02-development/tasks/B1,B2,B3` (баги) |
| Для каждой — .md (симптом/цель, требования, реализация, промпты, ручная проверка, commit) | ✅ в каждом `.md` |

## Блок 2 — Разработка (3 бага + 3 фичи, доказаны)

- **B1** — OTP-код не отдаётся вне dev (безопасность). [.md](02-development/tasks/B1-otp-code-leak.md)
- **B2** — двунаправленная сортировка списка броней под пагинацию. [.md](02-development/tasks/B2-booking-list-sort-order.md)
- **B3** — Dockerfile Go 1.23→1.25 (сборка образа). [.md](02-development/tasks/B3-dockerfile-go-version.md)
- **F1** — `route.geometry` в `/slots` (бэкенд). [.md](02-development/tasks/F1-route-geometry-slots.md)
- **F2** — карта рисует маршрут по реальной geometry (клиент). [.md](02-development/tasks/F2-route-map-geometry.md)
- **F3** — `/auth/refresh` (access+refresh, ротация, logout по сессии). [.md](02-development/tasks/F3-auth-refresh.md)
- Ревью двумя сабагентами + исправление находок (H1/H2/M1/M2/M3/L2/L3). [REVIEW](02-development/REVIEW-block2.md)

Проверено: `go test ./...` зелёный, wasmJs BUILD SUCCESSFUL, приложение работает end-to-end
(веб-клиент + API + Postgres, локально).

## Блок 3 — Тестирование (3 столпа)

- **Столп 1** — ревью требований на тестопригодность (2 сабагента). [requirements-review](03-testing/requirements-review.md)
- **Столп 2** — тест-кейсы: план покрытия + детально по ядру, MD + YAML. [test-cases](03-testing/test-cases/)
- **Столп 3** — автотесты: прогон покрытия + новый регресс-тест сортировки B2. [autotests-report](03-testing/autotests-report.md)

## Технологии / инструменты

- **Backend:** Go 1.25, chi, pgx/PostgreSQL, слоёная архитектура (HTTP → usecase → domain → storage),
  goose-миграции, интеграционные тесты.
- **Клиент:** Kotlin 2.2.20, Compose Multiplatform (Android/iOS/Web-wasmJs).
- **Инфра:** Docker Desktop (Postgres + образ API), локальный веб на :8081, API на :8080 (dev-CORS).
- **ИИ:** GLM-5.2 (ZCode) + Claude Opus 4.8 (Claude Code); сабагенты для ревью/тест-дизайна.
- **Среда:** Windows + WSL2, Docker, Gradle/JDK 17.

## Как запустить локально

```bash
# БД + API
cd backend && docker compose --profile app up -d --build
# веб-клиент (порт 8081)
cd client && ./gradlew :webApp:wasmJsBrowserDevelopmentRun
# тесты бэкенда
docker run --rm --network backend_default -e TEST_DATABASE_URL=postgres://volna:volna@db:5432/volna?sslmode=disable \
  -v "$PWD/backend:/src" -w /src golang:1.25-alpine sh -c "go test ./..."
```
