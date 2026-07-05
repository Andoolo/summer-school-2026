# F3 · Фича: обновление сессии через `/auth/refresh` (access + refresh, ротация)

> **Тип:** фича · **Слой:** backend (миграция + storage + service + handler + router) · **Сложность:** средняя · **Статус:** ✅ реализовано

## Потребность

Контракт (`01-analysis/api/auth/`) описывает сессии как **пару токенов** `TokenPair`
(`access_token` + `refresh_token`) и эндпоинт `POST /auth/refresh` (R-016): короткоживущий
access в `Authorization`, долгоживущий refresh — только для обновления пары. Клиент при 401
обновляет сессию через refresh, не заставляя пользователя логиниться заново.

Бэкенд же выдавал **одиночный** сессионный токен, `/auth/refresh` не было — то есть контракт не
реализован, и клиент не может продлевать сессию.

## Решение и его границы (scope)

Реализация сделана **аддитивно поверх существующей сессионной модели**, без слома работающей auth
и текущего клиента:

- verify-code теперь возвращает **супермножество**: прежний `token` (обратная совместимость с
  клиентом, который читает одиночный токен) **плюс** `tokens: { access_token, refresh_token }`
  (форма `TokenPair` из контракта);
- добавлен `POST /auth/refresh` — обмен refresh на новую пару с **атомарной ротацией** (старый
  refresh гасится в одной транзакции через `UPDATE ... RETURNING` — защита от reuse при гонке);
- `logout` инвалидирует access-сессию и refresh-токены **только текущей сессии** (по `session_id`,
  не трогая другие устройства клиента), R-016.

**Осознанные отклонения (задокументированы намеренно):**
- Токены — **непрозрачные** (случайные, хранятся как SHA-256 хеш), а не JWT, как в тексте
  контракта. Причина: вся текущая auth построена на opaque-сессиях (`auth_sessions.token_hash`),
  и переход на JWT затронул бы middleware и все защищённые эндпоинты — большой риск перед сдачей.
  Формат токена не меняет семантику access/refresh/ротации; переход на JWT — отдельная задача.
- **push-tokens НЕ реализованы намеренно.** Борд называл F3 «auth/refresh + push-tokens», но по
  зафиксированному ранее решению scope (`todo_after_review.md`, п. 15) **push из MVP убран**.
  Реализация push противоречила бы собственным докам проекта. `/auth/push-tokens` оставлен за
  скоупом; при возврате push — отдельная фича.
- `/auth/refresh` зарегистрирован **вручную** (`RouterOptions.AuthRefresh`), т.к. сгенерированный
  транспорт auth ещё не содержит этого маршрута (спека ушла вперёд кода). Регенерация OpenAPI —
  отдельная задача (заодно затронула бы verify-code/push).

## Требования

- Контракт: `TokenPair`, `POST /auth/refresh`, `POST /auth/logout` — `01-analysis/api/auth/api.yaml`, `models.yaml`.
- R-016 (жизненный цикл сессии: refresh, ротация, инвалидция при logout).

## Реализация

**1. Миграция `migrations/00003_refresh_tokens.sql`** — таблица `refresh_tokens`
(`client_id` FK, `token_hash` unique, `expires_at`, `revoked_at`), индексы по client и hash.

**2. `internal/storage/postgres/auth.go`** — 5 методов:
`SessionClientID`, `CreateRefreshToken`, `RefreshTokenClientID` (валидность: не отозван и не истёк),
`RevokeRefreshToken`, `RevokeClientRefreshTokens`.

**3. `internal/service/auth/service.go`**
- `refreshTTL = 30 дней` (access-сессия — прежние 24ч);
- `VerifyCode` дополнительно выдаёт refresh (`VerifyCodeResult.RefreshToken`);
- `Refresh(refreshToken)` → валидация → **ротация** (гасим предъявленный refresh) → новая
  access-сессия + новый refresh; недействительный refresh → `ErrInvalidSession`;
- `Logout` гасит access-сессию **и** все refresh-токены клиента.

**4. `internal/http/handlers/auth.go`**
- `VerifyAuthCode` пишет `verifyCodeResponseDTO` (token + tokens{access,refresh} + client + is_new);
- новый `Refresh` handler: `{refresh_token}` → `TokenPair`; `ErrInvalidSession` → 401.

**5. `internal/http/router.go` + `cmd/api/main.go`** — `RouterOptions.AuthRefresh` и регистрация
`POST /auth/refresh`; проброс `authHandler.Refresh` из main.

## Промпты ИИ

> **Промпт:** Контракт требует TokenPair (access+refresh) и /auth/refresh с ротацией и инвалидцией
> refresh при logout, но бэк отдаёт одиночный opaque-токен и refresh-эндпоинта нет. Как добавить
> refresh аддитивно, не ломая текущий клиент (читает `token`) и не переходя на JWT/регенерацию?
>
> **Действие:** таблица refresh_tokens; verify отдаёт супермножество (token + tokens); сервисные
> VerifyCode/Refresh/Logout с ротацией; ручной маршрут /auth/refresh. push — вне скоупа по решению.

## Проверка (ручная)

Собран образ (Go 1.25 — компиляция всего бэка прошла), применена миграция, живой прогон флоу:

```text
== 2. verify-code -> token + tokens{access,refresh} ==
{ "token": "_Zx9K…", "tokens": { "access_token": "_Zx9K…", "refresh_token": "IFqoh…" }, ... }
== 3. access-токен работает на /slots ==            GET /slots -> 200
== 4. /auth/refresh со старым refresh -> новая пара == { "access_token": "SN0T7…", "refresh_token": "KlZea…" }
== 5. РОТАЦИЯ: старый refresh больше не принимается == POST /auth/refresh (old refresh) -> 401
== 6. новый access-токен работает на /slots ==       GET /slots (new access) -> 200
== 7. logout новым access ==                          POST /auth/logout -> 204
== 8. после logout новый refresh непригоден ==        POST /auth/refresh (after logout) -> 401
```

✅ verify выдаёт пару; access авторизует запросы; refresh обменивается на новую пару; **ротация**
(старый refresh → 401); **logout** гасит refresh (→ 401). Полный жизненный цикл сессии R-016 работает.

Дополнительно: `go vet ./...` в контейнере `golang:1.25` — типизация всех пакетов и тестов
(включая обновлённый `fakeRepo`) без ошибок.

## Инструменты

- ИИ: Claude Code.
- Среда: Docker Desktop (Postgres 16 + API-образ Go 1.25), `curl`, `psql`.
- Проверка: сборка образа + миграция + живой auth-флоу (verify/refresh/rotation/logout) + `go vet`.

## Файлы

- `backend/migrations/00003_refresh_tokens.sql` — таблица refresh_tokens.
- `backend/migrations/00004_refresh_token_session.sql` — связь `session_id` (для logout по сессии).
- `backend/internal/storage/postgres/auth.go` — транзакционные `IssueSession`/`RotateSession`/`RevokeSessionByAccessToken`.
- `backend/internal/service/auth/service.go` — refreshTTL, выдача пары, `Refresh` (атомарная ротация), logout по сессии, `AccessTTLSeconds`.
- `backend/internal/http/handlers/auth.go` — DTO ответа (`token_type`/`expires_in`) + handler `Refresh`.
- `backend/internal/http/router.go`, `backend/cmd/api/main.go` — маршрут `/auth/refresh`.
- `backend/internal/service/auth/service_test.go` — стабы новых методов в `fakeRepo`.

## Правки по ревью (см. `02-development/REVIEW-block2.md`)

- **H1 (безопасность):** ротация атомарна — `RotateSession` гасит предъявленный refresh через
  `UPDATE ... WHERE revoked_at IS NULL RETURNING client_id` в транзакции; повторное использование
  (в т.ч. при гонке) → 401.
- **H2 (контракт):** refresh связан с сессией (`session_id`, миграция 00004); `logout` гасит только
  refresh текущей сессии — другие устройства клиента продолжают работать.
- **M1 (контракт):** `TokenPair` дополнен `token_type:"Bearer"` и `expires_in` (срок жизни access).
- **L2:** ротация/logout/issue — в транзакциях (частичный сбой откатывается).
- Проверено live: `token_type=Bearer, expires_in=86400`; logout A → refreshA 401, refreshB (устройство B) 200; reuse refresh → 401.

> **Ограничение:** ротация создаёт новую сессию, не связанную «семьёй» со старой. Logout по
> устаревшему (до-ротационному) access-токену не гасит ротированного потомка; в нормальном потоке
> клиент логаутится текущим токеном — тогда корректно. Token-family — отдельная задача.
