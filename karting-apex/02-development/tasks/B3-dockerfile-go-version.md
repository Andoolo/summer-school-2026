# B3 · Баг: несовместимость версий Go в Dockerfile

> **Тип:** баг (build) · **Слой:** backend/infra · **Сложность:** низкая · **Статус:** ✅ исправлено

## Симптом

Сборка Docker-образа бэкенда падает на этапе `go mod download`:

```
#12 [build 4/6] RUN go mod download
#12 3.353 go: go.mod requires go >= 1.25.7 (running go 1.23.12; GOTOOLCHAIN=local)
#12 ERROR: process "/bin/sh -c go mod download" did not complete successfully: exit code: 1
failed to solve: process "/bin/sh -c go mod download" did not complete successfully: exit code: 1
```

`docker compose --profile app up --build api` не собирается → бэкенд невозможно поднять штатным способом из `LOCAL_DEV_GUIDE.md`.

## Причина / root cause

`backend/go.mod` требует Go **1.25.7**:
```go
module summer-school-2026/backend
go 1.25.7
```

А `backend/Dockerfile` использует образ **`golang:1.23-alpine`**:
```dockerfile
FROM golang:1.23-alpine AS build
```

С `GOTOOLCHAIN=local` (по умолчанию в образе) Go отказывается авто-скачивать toolchain 1.25.7 и
падает. Версии рассинхронизированы: `go.mod` обновили (видимо, при bumped зависимостей), а
`Dockerfile` забыли.

> Примечание: профиль `migrations` в `compose.yaml` использует тот же `golang:1.23-alpine`, но
> БЕЗ `GOTOOLCHAIN=local`, поэтому миграции проходят (Go сам подтягивает 1.25.7). Это и ввело в
> заблуждение — казалось, что «всё работает», хотя штатный build образа API сломан.

## Требования

Прямого функционального требования нет — это инфраструктурный инвариант: **сборка образа из
`LOCAL_DEV_GUIDE.md` должна работать** (`docker compose --profile app up --build`). Соответствует
NFR-15 (запуск к сроку) и базовому принципу «проект собирается штатно».

## Реализация

Файл: `backend/Dockerfile`

```diff
- FROM golang:1.23-alpine AS build
+ FROM golang:1.25-alpine AS build
```

Образ `golang:1.25-alpine` содержит toolchain 1.25.x ≥ 1.25.7 → `go mod download` и `go build`
проходят без авто-скачивания toolchain.

> Использован тег `1.25-alpine` (а не точный `1.25.7-alpine`): это даёт минорные патчи безопасности
> в рамках 1.25.x и следует практике репозитория (там тоже используются мажорные теги `1.23-alpine`).

## Промпты ИИ

> **Промпт (диагностика):** Сборка `docker compose --profile app up --build api` падает с
> `go: go.mod requires go >= 1.25.7 (running go 1.23.12; GOTOOLCHAIN=local)`. Как исправить?
> Сравни go.mod и Dockerfile.
>
> **Ответ/действие:** Поднять базовый образ в Dockerfile с `golang:1.23-alpine` до `golang:1.25-alpine`,
> чтобы версия toolchain удовлетворяла go.mod.

## Проверка (ручная)

```bash
cd backend
docker compose --profile app up -d --build api
# ...сборка прошла без ошибок, контейнер поднялся...
curl -s http://127.0.0.1:8090/healthz   # → {"status":"ok"}
curl -s http://127.0.0.1:8090/readyz    # → {"status":"ok"}
docker ps --filter name=backend-api    # → Up
```

✅ Образ собирается, API отвечает `{"status":"ok"}` на `/healthz` и `/readyz`.

## Инструменты

- ИИ: ZCode (встроенный ассистент)
- Среда: ZCode CLI + Docker Desktop 4.80 (engine 29.6.1) на Windows
- Проверка: `docker compose`, `curl`

## Файлы

- `backend/Dockerfile` — единственное изменение (1 строка)
