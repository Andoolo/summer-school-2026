# Столп 3 · Автотесты и покрытие

> Анализ существующего покрытия + дописанный ИИ регресс-тест на ключевую фичу Блока 2 (сортировка B2).
> **Метод:** прогон `go test ./... -cover` в контейнере `golang:1.25` против живой БД (интеграционные
> тесты сами накатывают миграции в свежую схему). Клиент — `commonTest` (KMP), запускается gradle.

## Прогон бэкенда — все пакеты зелёные

```text
ok  internal/config              coverage: 87.5%
ok  internal/http                coverage: 75.3%
ok  internal/http/handlers       coverage: 65.2%   (интеграционные: auth/booking/catalog/profile)
ok  internal/service/auth        coverage: 29.9%
ok  internal/service/booking     coverage: 54.8%
ok  internal/storage/postgres    coverage:  8.2%   (см. примечание)
    internal/http/openapi/*      coverage:  0.0%   (сгенерированный транспорт — не тестируется)
```

> **Примечание о покрытии:** Go считает покрытие по **своему** пакету теста. Слой `storage/postgres`
> реально прогоняется **кросс-пакетно** интеграционными тестами из `handlers` (booking/auth/catalog),
> поэтому 8.2% занижены — эффективное покрытие SQL-слоя существенно выше. Для точной цифры нужен
> `go test -coverpkg=./...` (не гоняли, чтобы не раздувать прогон). Генерённый `openapi/*` не покрываем
> намеренно.

## Существующие автотесты (13 файлов) ↔ тест-кейсы Столпа 2

| Автотест | Покрывает кейсы |
|----------|-----------------|
| `service/auth/service_test.go` | TC-AUTH-01/03/04/05 (валидация, throttle, verify) |
| `handlers/auth_integration_test.go` | TC-AUTH-06 (verify→токены), logout-флоу |
| `handlers/bookings_integration_test.go` · `TestCreateBookingFlowAndIdempotency` | TC-BOOK-01/03 (запись, идемпотентность) |
| … · `TestCreateBookingConcurrencyDoesNotOverbook` | **TC-BOOK-04** (конкурентность без овербукинга, NFR-8) |
| … · `TestCancelBookingEarlyLateAndAfterStart` | **TC-BOOK-06/07/08** (early/late/after-start отмена) |
| … · `TestCancelBookingConcurrencyReturnsAvailabilityOnce` | конкурентная отмена |
| … · `TestListAndGetBookings` | **TC-MYB-04** (только свои брони, 403/404, NFR-12) |
| `handlers/bookings_sort_test.go` · `TestListBookingsSortOrder` | **TC-MYB-02** (сортировка B2) — ⬅ **новый, Блок 3** |
| `handlers/catalog_integration_test.go` | TC-SLOT-01 (список слотов, geometry F1) |
| `service/booking/cancel_test.go`, `service_test.go` | правило 2ч, валидация пагинации |
| `storage/postgres/constraints_integration_test.go` | инварианты БД (seats/rental/уникальность) |

## Дописанный автотест (ИИ, Блок 3)

**`backend/internal/http/handlers/bookings_sort_test.go` · `TestListBookingsSortOrder`**

Заполняет пробел: прежний list-тест использовал `limit=1` и **многоэлементный порядок не проверял**.
Новый тест вставляет брони на `now±1д` и `now±10д` и проверяет двунаправленную сортировку B2:
предстоящие по возрастанию (`+1д` перед `+10д`, AC-002), затем прошедшие по убыванию (`−1д` перед
`−10д`, AC-003).

> **Промпт (ИИ):** «Напиши Go integration-тест в пакете handlers_test, который вставляет 4 брони
> клиента на слоты now+1д/now+10д/now−1д/now−10д и проверяет, что `GET /bookings` возвращает их в
> порядке [+1д, +10д, −1д, −10д] (двунаправленная сортировка B2). Используй хелперы
> insertClientSession/insertBooking/bookingRouter и seed-трасса/маршала.»

**Результат:** тест зелёный (`ok internal/http/handlers`), порядок соответствует AC-002/AC-003.

## Клиент (KMP)

`client/shared/src/commonTest/…` — 4 набора: `BookingDetailsStoreTest`, `BookingListStoreTest`,
`PhoneInputCoreTest`, `DomainPolicyTest`. Запуск: `./gradlew :shared:allTests`. Компиляция
подтверждена (wasmJs BUILD SUCCESSFUL); F2 (карта по geometry) выверена статически + визуально.

## Вывод

- Бэкенд-набор зелёный; ключевые инварианты (конкурентность, границы отмены, доступ к своим броням)
  покрыты интеграционными тестами.
- Наши фичи Блока 2 имеют регресс: **B2** (новый sort-тест), **F1** (catalog-тест на geometry),
  **F3** (auth service + integration), **B1/B3** — доказаны в Блоке 2 (curl/сборка).
- Приоритет доработки покрытия (follow-up): сервис auth (29.9% — refresh/rotate ветки), сценарии
  429 на read (пробел H8), двусторонняя гонка отмена+запись (H5).

## Инструменты

- ИИ: Claude Code (генерация теста + анализ покрытия).
- Среда: Docker (`golang:1.25`, Postgres 16), `go test -cover`; клиент — Gradle/Kotlin 2.2.20.
