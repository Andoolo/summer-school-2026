# B1 · Баг: OTP-код возвращается в ответе /auth/request-code всегда

> **Тип:** баг (безопасность) · **Слой:** backend · **Сложность:** низкая · **Статус:** ✅ исправлено

## Симптом

Эндпоинт `POST /auth/request-code` безусловно возвращает сгенерированный OTP-код в теле ответа,
независимо от окружения:

```json
POST /auth/request-code {"phone":"+79991234567"}
→ 200 {"code":"9469","resend_after_seconds":60,"ttl_seconds":300}
```

В production это **утечка секрета**: любой, кто знает номер, получает код авторизации без SMS и
может войти в чужой аккаунт.

## Причина / root cause

`backend/internal/http/handlers/auth.go`, `RequestAuthCode`:

```go
httpapi.WriteJSON(w, http.StatusOK, authapi.RequestCodeResponse{
    TtlSeconds:         result.TTLSeconds,
    ResendAfterSeconds: result.ResendAfterSeconds,
    Code:               &result.Code,   // ← безусловно кладётся в ответ
})
```

Контракт (`01-analysis/api/auth/api.yaml`, поле `RequestCodeResponse.code`) и план реализации
(CMP-план, open-question №5) явно требуют: **в production код не должен возвращаться; dev-поведение
должно гейтиться окружением**. Гейтинга не было — код лился всегда.

## Требования

- **FR-43** (авторизация по SMS OTP) + **NFR-19** (антифрод OTP) + **NFR-20** (ПДн/безопасность).
- Контракт: `code` возвращается только в dev; в production клиент получает лишь TTL/resend_after,
  код доставляется отдельным каналом (SMS).

## Реализация

Подход: флаг dev из окружения → проброс в handler → гейт возврата кода.

**1. `internal/config/config.go`** — новое поле `Dev` (true, если `APP_ENV != "production"`):
```go
type Config struct {
    HTTPAddr        string
    DatabaseURL     string
    ShutdownTimeout time.Duration
    Dev bool   // ← new
}
// Load(): Dev: boolFromEnvIsNot("APP_ENV", "production")
```

**2. `internal/http/handlers/auth.go`** — handler хранит `dev` и гейтит возврат:
```go
type AuthHandler struct {
    service *auth.Service
    dev     bool   // ← new
}
func NewAuthHandler(service *auth.Service, dev bool) *AuthHandler { ... }

// RequestAuthCode:
response := authapi.RequestCodeResponse{
    TtlSeconds:         result.TTLSeconds,
    ResendAfterSeconds: result.ResendAfterSeconds,
}
if h.dev {
    response.Code = &result.Code   // ← только в dev
}
```

**3. `cmd/api/main.go`** — проброс `cfg.Dev` в конструктор handler.
**4. `compose.yaml`** — `APP_ENV: "development"` для dev-окружения.
**5. `auth_integration_test.go`** — вызовы `NewAuthHandler(service, true)` (тесты = dev).

> Дизайн-решение: флаг хранится в handler, а не в service. Service по-прежнему генерирует код и
> логирует его (`dev otp generated`) — это корректно для dev-лога. Ответственность handler —
> решать, что отдавать клиенту. Безопасный дефолт: всё, что не "production" — dev (лучше лишний
> раз показать код в dev, чем утечь в prod из-за опечатки в env).

## Промпты ИИ

> **Промпт (диагностика):** `/auth/request-code` возвращает `"code":"9469"` всегда, даже в
> production. Контракт требует гейтить dev-окружением. Как аккуратно добавить флаг dev, не ломая
> service и тесты? Предложи минимальный diff.
>
> **Действие по подсказке:** поле `Dev` в Config из `APP_ENV`, проброс в `NewAuthHandler`,
> `if h.dev { response.Code = ... }`, обновить main.go и тесты.

## Проверка (ручная)

Два сценария через два контейнера (dev и production):

```bash
# DEV (APP_ENV=development) — код возвращается для ручной проверки:
curl -s -X POST localhost:8080/auth/request-code -d '{"phone":"+79990000001"}'
→ {"code":"7529","resend_after_seconds":60,"ttl_seconds":300}   ✅ код есть

# PROD (APP_ENV=production) — код НЕ возвращается:
docker run --rm -d --network backend_default -e APP_ENV=production -e DATABASE_URL=... -p 8090:8090 backend-api
curl -s -X POST localhost:8090/auth/request-code -d '{"phone":"+79990000002"}'
→ {"resend_after_seconds":60,"ttl_seconds":300}   ✅ код ОТСУТСТВУЕТ
```

✅ Баг устранён: production больше не отдаёт OTP-код; dev сохраняет удобство ручной проверки.

## Инструменты

- ИИ: ZCode (встроенный ассистент)
- Среда: ZCode CLI + Docker Desktop 4.80
- Проверка: `curl` + два контейнера (dev/prod) для сравнения поведения

## Файлы

- `internal/config/config.go` — поле `Dev` + helper `boolFromEnvIsNot`
- `internal/http/handlers/auth.go` — поле `dev` + гейт возврата кода
- `cmd/api/main.go` — проброс `cfg.Dev`
- `compose.yaml` — `APP_ENV: "development"`
- `internal/http/handlers/auth_integration_test.go` — адаптация вызовов
