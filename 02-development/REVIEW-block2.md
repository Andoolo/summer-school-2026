# Ревью Блока 2 (сабагенты + верификация)

> Два параллельных сабагента (бэкенд Go, клиент Kotlin) прошлись по диффу `04d8974..HEAD`.
> Ниже — сведённые находки с моим вердиктом (каждую сверил с кодом/контрактом). Отсортировано по severity.
> Дата: 2026-07-04. Ветка: `feature/client`.

## Итог одной строкой

Критичных «падает прод» багов нет. Есть **2 HIGH** по F3 (безопасность refresh + объём logout) и
несколько MEDIUM/LOW по соответствию контракту. Всё — устранимо точечно.

---

## HIGH

### H1 · F3 Refresh: refresh-токен можно переиспользовать при гонке (reuse не детектится) — ✅ подтверждено
**`service/auth/service.go` `Refresh()` + `storage/postgres/auth.go` `RevokeRefreshToken`**
`Refresh()` делает SELECT (`RefreshTokenClientID`) → UPDATE (`RevokeRefreshToken`) отдельными вызовами,
а UPDATE **игнорирует rows-affected**. Два параллельных запроса с одним refresh-токеном оба пройдут
SELECT и оба выдадут по новой паре → украденный/утёкший refresh размножается в две сессии. Контракт
`api/auth/api.yaml` (R-016): «уже использованный refresh-токен → 401» — не соблюдается под гонкой.
**Фикс:** атомарный «consume»: `UPDATE refresh_tokens SET revoked_at=$2 WHERE token_hash=$1 AND
revoked_at IS NULL AND expires_at > $2 RETURNING client_id`. Нет строки → `ErrInvalidSession`.
Один запрос вместо SELECT+UPDATE — закрывает гонку без явной транзакции.

### H2 · F3 Logout: гасит refresh ВСЕХ устройств клиента (шире контракта) — ✅ подтверждено
**`service/auth/service.go` `Logout()` → `RevokeClientRefreshTokens`**
Контракт `api/auth/api.yaml` (`/auth/logout`): «Завершает **текущую сессию**: инвалидирует
refresh-токен». Реализация отзывает `WHERE client_id=$1` — все refresh-токены клиента. Выход на одном
устройстве разлогинит остальные. Причина — refresh не связан с access-сессией в БД.
**Фикс (варианты):**
- (а, контракт-корректно) связать refresh с сессией (`session_id`/`family_id` в `refresh_tokens`),
  в logout гасить только refresh текущей сессии;
- (б, прагматично) принять «logout со всех устройств» как MVP-поведение и поправить формулировку в
  `api.yaml`/доке F3. **← требуется решение владельца.**

---

## MEDIUM

### M1 · TokenPair неполный: нет `token_type` и `expires_in` — ✅ подтверждено по контракту
**`http/handlers/auth.go` `tokenPairDTO`**
`models.yaml` TokenPair `required: [access_token, refresh_token, token_type, expires_in]`. DTO отдаёт
только два первых на `/auth/verify-code` и `/auth/refresh`. Клиенту по контракту нужны `token_type:
"Bearer"` и `expires_in` (планировать refresh до истечения access).
**Фикс:** добавить `token_type="Bearer"` и `expires_in=int(sessionTTL.Seconds())` (пробросить TTL из сервиса).

### M2 · F1 geometry: encoded-polyline (строка) в БД → 500 на всём /slots — ✅ подтверждено
**`http/handlers/catalog.go` `slotBase`**
`Geometry` в контракте — `oneOf`: массив координат **или** строка (polyline). Код безусловно
`json.Unmarshal(..., &[][]float32)`. Строка → ошибка → 500 на всю выдачу `/slots`. Сейчас смягчено
тем, что сид кладёт только массивы (маршруты read-only), но контракт строку разрешает.
**Фикс:** различать форму jsonb: первый значащий байт `"` → `FromGeometry1(string)`, иначе `FromGeometry0(coords)`.

### M3 · B2: `NOW()` пересчитывается на каждой странице → дубли/пропуски при пагинации — ✅ подтверждено
**`storage/postgres/bookings.go` `List` (ORDER BY)**
`NOW()` в `CASE` вычисляется в момент каждого запроса. При ленивой догрузке (R-025) бронь, пересекающая
границу `now` между запросами страниц, может продублироваться/пропасть. Окно узкое, но реально.
**Фикс:** передавать `now` параметром (`$N`, из `internal/clock`), единым для всей сессии пагинации.
Бонус — тестируемость через инъектируемое время (инвариант AGENTS.md).

---

## LOW

- **L1 · B2 расходится с задокументированным `start_at DESC`** (`api/bookings/api.yaml`, SCR-005 стр.49).
  Клиент всё равно ресортит, экран не ломается. Уже отмечено в `B2-*.md`. Решение: обновить спеку под
  двунаправленную сортировку **или** вернуть DESC. (Осознанное отклонение, задокументировано.)
- **L2 · F3 частичный сбой ротации/logout не транзакционен** (`service/auth/service.go`). Сбой между
  revoke и create оставит клиента без сессии; logout может вернуть 500 вместо 204. Закрывается той же
  транзакцией/атомарностью, что и H1.
- **L3 · F2 `projectRoute()` пересчитывается на каждый кадр Canvas** (`RouteMapPreviewFallback.kt`).
  Для длинной геометрии — лишние аллокации. Фикс: `remember(routePoints, size)`.
- **L4 · F2 гео edge-cases** (антимеридиан ±180°, полюса cos(lat)≤0) не обрабатываются. Для сёрф-региона
  нерелевантно. (Клиент-агент: математика проекции в остальном корректна — север сверху, без зеркал,
  деление на ноль защищено epsilon+guard.)

---

## Что чисто (проверено, замечаний нет)

- B1 (OTP-гейт): логика верна, безопасный дефолт `APP_ENV != production`.
- B3 (Go 1.25): тривиально, `go.mod` совместим.
- Миграция 00003: goose-транзакция по умолчанию; схема/индексы/FK/CHECK консистентны с `auth_sessions`.
- SQL-инъекций нет (везде плейсхолдеры `$N`); слои AGENTS.md соблюдены (тонкие handler-ы, явный DTO-маппинг).
- `go test ./...` — зелёный; `go vet` — чист; wasmJs — BUILD SUCCESSFUL.

---

## Рекомендуемый порядок исправления

1. **H1** (reuse refresh) — атомарный consume. Безопасность, чисто, низкий риск. → фикс + re-prove.
2. **M1** (TokenPair token_type/expires_in) — быстрый контракт-фикс.
3. **M2** (geometry string-guard) — быстрый, защитный.
4. **H2** (logout scope) — **нужно решение**: связать refresh↔сессию (а) или принять «logout всех» (б).
5. **M3** (now-параметр в B2) — среднее, убирает пагинационную гонку + тестируемость.
6. **L2/L3** — транзакция ротации / `remember` в F2 — по желанию.

---

## Итог исправлений (все приняты — HIGH+MEDIUM+LOW)

Решение по H2: **вариант (а)** — контракт-корректно (logout только текущей сессии).

| # | Что сделано | Проверка |
|---|-------------|----------|
| **H1** | Ротация атомарна: `RotateSession` — `UPDATE refresh_tokens ... WHERE revoked_at IS NULL RETURNING client_id` в одной транзакции. Reuse не проходит. | live: reuse refresh → **401** |
| **H2** | Связал refresh↔сессию (миграция 00004 `session_id`). `Logout` = `RevokeSessionByAccessToken`: гасит только refresh текущей сессии. | live: logout A → refreshA **401**, refreshB (устройство B) **200** |
| **L2** | Ротация и logout — в транзакциях (`IssueSession`/`RotateSession`/`RevokeSessionByAccessToken`), частичный сбой откатывается. | `go test` зелёный |
| **M1** | `tokenPairDTO` дополнен `token_type:"Bearer"` + `expires_in` (из `AccessTTLSeconds`). | live: `token_type=Bearer, expires_in=86400` |
| **M2** | `catalog.go` различает форму geometry: строка (polyline)→`FromGeometry1`, массив→`FromGeometry0` (helper `isJSONString`). | сборка + `go test` |
| **M3** | Граница now в сортировке B2 — параметр `$now` из инъектируемых часов (`BookingRepository.now`). Полная кросс-страничная стабильность требует курсорной пагинации — отмечено как follow-up. | `go test` зелёный |
| **L3** | F2: тяжёлая часть проекции вынесена в `remember(routePoints)` (`prepareRoute`/`toCanvasPoints`), в Canvas — только масштабирование. | wasmJs BUILD SUCCESSFUL |
| L1 | B2-отклонение от DESC — осознанное, задокументировано в `B2-*.md`. | — |
| L4 | Гео edge-cases (антимеридиан/полюса) — нерелевантны сёрф-региону, оставлено. | — |

**Известное ограничение (honest note):** ротация создаёт новую сессию, не связанную «семьёй» со
старой. Поэтому logout по **устаревшему** (до-ротационному) access-токену не гасит ротированного
потомка. В нормальном потоке клиент после refresh держит новый access и logout делает им — тогда
всё корректно. Полноценные token-family (revoke всей линии при reuse) — отдельная задача.

Повторная проверка после правок: `go test ./...` — зелёный; wasmJs — BUILD SUCCESSFUL; live-флоу
verify/refresh/rotation/logout(мульти-девайс) — как ожидается.
