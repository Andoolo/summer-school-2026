# План реализации BE для «Апекса»

## TOC / Todo реализации

- [x] [BE-00. Создать каркас backend-приложения](#be-00-создать-каркас-backend-приложения)
- [x] [BE-01. Подключить OpenAPI как контракт](#be-01-подключить-openapi-как-контракт)
- [x] [BE-02. Реализовать общую HTTP-инфраструктуру](#be-02-реализовать-общую-http-инфраструктуру)
- [x] [BE-03. Спроектировать БД и миграции](#be-03-спроектировать-бд-и-миграции)
- [x] [BE-04. Реализовать Auth: OTP и сессии](#be-04-реализовать-auth-otp-и-сессии)
- [x] [BE-05. Реализовать Profile](#be-05-реализовать-profile)
- [x] [BE-06. Реализовать read-only каталог слотов и маршалов](#be-06-реализовать-read-only-каталог-слотов-и-маршалов)
- [x] [BE-07. Реализовать атомарное создание брони](#be-07-реализовать-атомарное-создание-брони)
- [x] [BE-08. Реализовать список и детали броней](#be-08-реализовать-список-и-детали-броней)
- [x] [BE-09. Реализовать отмену брони](#be-09-реализовать-отмену-брони)
- [x] [BE-10. Довести контрактные ошибки и валидацию](#be-10-довести-контрактные-ошибки-и-валидацию)
- [x] [BE-11. Добавить полный набор Go-тестов](#be-11-добавить-полный-набор-go-тестов)
- [x] [BE-12. Добавить k6 performance tests до 300 concurrent users](#be-12-добавить-k6-performance-tests-до-300-concurrent-users)
- [x] [BE-13. Подготовить локальный запуск и документацию разработчика](#be-13-подготовить-локальный-запуск-и-документацию-разработчика)
- [x] [BE-14. Финальная проверка готовности BE](#be-14-финальная-проверка-готовности-be)

## Стек приложения

- Язык и рантайм: Go 1.23+.
- API: RESTful JSON API, OpenAPI-first, контракты из `01-analysis/api/redocly.yaml` и доменных `api.yaml`.
- HTTP: `net/http` + `chi`, middleware для auth, request id, recovery, logging.
- OpenAPI: `oapi-codegen` для генерации типов/серверных интерфейсов с сохранением `operationId`.
- БД: PostgreSQL 16, транзакции и row-level locks для бронирований, `pgx`/`pgxpool`.
- Миграции и SQL: `goose` для миграций, `sqlc` для типобезопасных запросов без ORM.
- Auth: phone/OTP flow, JWT Bearer access token, интерфейс SMS/OTP provider; dev-реализация пишет OTP в лог.
- Тесты: Go unit/integration tests, concurrency tests для booking/cancel, k6 performance tests до 300 concurrent users.
- Runtime: Docker Compose для API + PostgreSQL, конфигурация через env, structured logs через `slog`.

## Функционал и endpoints

| Домен | operationId | Endpoint | Функционал |
|---|---|---|---|
| Auth | `requestAuthCode` | `POST /auth/request-code` | Запрос OTP по телефону |
| Auth | `verifyAuthCode` | `POST /auth/verify-code` | Проверка OTP, вход/регистрация, выдача токена |
| Auth | `logout` | `POST /auth/logout` | Завершение текущей сессии |
| Profile | `getProfile` | `GET /profile` | Профиль текущего клиента |
| Profile | `updateProfile` | `PATCH /profile` | Обновление имени клиента |
| Profile | `deleteAccount` | `DELETE /profile` | Удаление аккаунта и персональных данных |
| Profile | `requestPhoneChangeCode` | `POST /profile/phone/request-code` | OTP для смены телефона |
| Profile | `confirmPhoneChange` | `POST /profile/phone/confirm` | Подтверждение нового телефона |
| Slots | `listSlots` | `GET /slots` | Список слотов с фильтрами и пагинацией |
| Slots | `getSlot` | `GET /slots/{slotId}` | Карточка слота |
| Instructors | `listInstructors` | `GET /instructors` | Read-only справочник маршалов |
| Bookings | `createBooking` | `POST /bookings` | Атомарное создание брони, `Idempotency-Key` |
| Bookings | `listBookings` | `GET /bookings` | Список броней текущего клиента |
| Bookings | `getBooking` | `GET /bookings/{bookingId}` | Детали своей брони |
| Bookings | `cancelBooking` | `POST /bookings/{bookingId}/cancel` | Отмена брони по правилу 2 часов |

Служебные endpoints для эксплуатации можно добавить вне клиентского API: `GET /healthz`, `GET /readyz`. Если они попадают в публичный контракт, сначала обновить OpenAPI.

## Правила для ralph loop

- Один пункт ниже = одна итерация: взять контекст, реализовать минимальный вертикальный срез, добавить/обновить тесты, прогнать указанную проверку.
- Любое изменение публичного API начинается с правки `01-analysis/api/*` и проверки `npm --prefix 01-analysis/api run lint`.
- Не добавлять instructor/admin UI/API, schedule CRUD, slot creation/editing, online payment, ratings, loyalty, weather cancellation и no-show.

## Декомпозиция BE

### BE-00. Создать каркас backend-приложения

Сделать:
- Создать `backend/` с Go module, `cmd/api/main.go`, `internal/config`, `internal/http`, `internal/domain`, `internal/storage`, `internal/service`.
- Добавить Makefile или `task`-команды для `fmt`, `lint`, `test`, `run`, `migrate`, `k6-smoke`.
- Поднять Docker Compose для PostgreSQL и локального API.

Готово, когда:
- `go test ./...` проходит в `backend/`.
- API стартует локально и отдаёт служебный health endpoint.

### BE-01. Подключить OpenAPI как контракт

Сделать:
- Настроить генерацию Go DTO/server interfaces из текущих доменов `auth`, `slots`, `bookings`, `profile`, `instructors`.
- Сохранить имена `operationId` в структуре handler методов.
- Добавить проверку, что generated code не редактируется вручную.

Готово, когда:
- `npm --prefix 01-analysis/api run lint` проходит после установки зависимостей.
- `go generate ./...` или выбранная команда codegen воспроизводит generated files.
- `go test ./...` проходит.

### BE-02. Реализовать общую HTTP-инфраструктуру

Сделать:
- Middleware: request id, access log, panic recovery, JSON content type, auth extractor.
- Единый error mapper под `01-analysis/api/common/models.yaml`: `code`, `message`, `details`.
- Валидация request body/query/path с возвратом `400`, `401`, `403`, `404`, `409`, `410`, `422`, `429`, `default` по контрактам.

Готово, когда:
- Handler tests проверяют формат ошибок и status codes.
- Неверный JSON, отсутствующий Bearer token и неизвестный path возвращают контрактные ответы.

### BE-03. Спроектировать БД и миграции

Сделать:
- Таблицы `clients`, `auth_sessions`, `otp_codes`, `routes`, `instructors`, `slots`, `bookings`, `idempotency_keys`.
- `routes`, `instructors`, `slots` сделать read-only для клиентского API; данные для dev/test загружать seed-миграцией или fixtures.
- Для `bookings` сохранить `status in (active,cancelled,late_cancel)`, `seats_count`, `rental_count`, `created_at`, `cancelled_at`.
- Для `slots` хранить данные, достаточные для атомарного расчёта `free_seats` и `free_rental_boards`; не полагаться на FE для лимитов.
- Добавить индексы по `phone`, `slot_id`, `client_id`, `start_at`, `status`, idempotency key.

Готово, когда:
- Миграции применяются на пустую PostgreSQL.
- Integration test поднимает схему, создаёт seed slots и читает их через repository.

### BE-04. Реализовать Auth: OTP и сессии

Сделать:
- `POST /auth/request-code`: нормализовать телефон, ограничить частоту запросов, создать OTP с TTL, вернуть `resend_after_seconds`/TTL по модели.
- `POST /auth/verify-code`: проверить OTP, создать клиента при первом входе, вернуть token, client, `is_new`.
- `POST /auth/logout`: инвалидировать текущую сессию/token id.
- Хранить OTP только в hash-виде; dev OTP provider пишет код в лог.

Готово, когда:
- Unit tests покрывают валидный код, неверный код, истёкший код, повторное использование, rate limit.
- Integration test проходит полный login flow.

### BE-05. Реализовать Profile

Сделать:
- `GET /profile`: вернуть только текущего клиента.
- `PATCH /profile`: обновить имя, телефон через этот endpoint не менять.
- `POST /profile/phone/request-code`: создать OTP для нового телефона и проверить конфликт телефона.
- `POST /profile/phone/confirm`: подтвердить OTP и атомарно сменить телефон.
- `DELETE /profile`: удалить/анонимизировать ПДн, инвалидировать сессии, аннулировать активные записи согласно `profile/api.yaml`.

Готово, когда:
- Tests проверяют доступ только к своему профилю, конфликт телефона и смену телефона по OTP.
- Удалённый аккаунт не может использовать старый token.

### BE-06. Реализовать read-only каталог слотов и маршалов

Сделать:
- `GET /slots`: фильтры `date_from`, `date_to`, `route_type[]`, `instructor_id[]`, `only_available`, `limit`, `offset`; сортировка по `start_at ASC`.
- `GET /slots/{slotId}`: вернуть слот с route geometry, meeting point, instructor, prices, availability.
- `GET /instructors`: справочник маршалов с пагинацией.
- Для `only_available=false` показывать слоты без мест с корректным `free_seats=0`; не скрывать их на сервере без параметра.

Готово, когда:
- Integration tests покрывают все фильтры, пустой результат, пагинацию и 404 для неизвестного слота.
- Клиентские поля совпадают с OpenAPI models.

### BE-07. Реализовать атомарное создание брони

Сделать:
- `POST /bookings` принимать `Idempotency-Key` и сохранять результат для безопасного retry.
- Валидировать `seats_count` в диапазоне `1..3` и `rental_count` в диапазоне `0..seats_count`.
- В транзакции заблокировать слот, проверить `status=scheduled`, `start_at` в будущем, свободные места и комплекты экипировки.
- Предотвратить double booking текущего клиента на тот же слот согласно NFR-8.
- Уменьшать доступность слота только после успешного создания брони.
- Возвращать `409 slot_full`/`double_booking`, `410 slot_cancelled`, `422 slot_started` с `details.available_*`, где применимо.

Готово, когда:
- Concurrency test с параллельными `createBooking` не допускает `free_seats < 0` и `free_rental_boards < 0`.
- Повтор с тем же `Idempotency-Key` возвращает тот же результат без второй брони.
- Сетевой retry сценарий из UC-1 E4 покрыт тестом.

### BE-08. Реализовать список и детали броней

Сделать:
- `GET /bookings`: вернуть только брони текущего клиента, поддержать фильтр `status`, `limit`, `offset`.
- `GET /bookings/{bookingId}`: вернуть только свою бронь с вложенными slot/route/instructor данными для SCR-006.
- Не хранить статус `past`; прошедшие брони определяются по `slot.start_at`.

Готово, когда:
- Tests проверяют `403` для чужой брони, `404` для неизвестной, фильтр по статусу и пагинацию.
- В ответах нет неописанных статусов вроде `past` или `no_show`.

### BE-09. Реализовать отмену брони

Сделать:
- `POST /bookings/{bookingId}/cancel`: в транзакции заблокировать booking и slot.
- Отмена доступна только до `slot.start_at`.
- Если до старта `>= 2h`, статус `cancelled`, места и комплекты экипировки возвращаются.
- Если до старта `< 2h`, статус `late_cancel`, места и комплекты экипировки не возвращаются.
- Ровно `2h` считать ранней отменой.
- Повторную отмену возвращать как контрактную ошибку `409 already_cancelled`.

Готово, когда:
- Unit tests покрывают границы времени: `2h+1s`, `2h`, `1h59m59s`, после старта.
- Concurrency test параллельных cancel не возвращает места дважды.

### BE-10. Довести контрактные ошибки и валидацию

Сделать:
- Зафиксировать machine codes: `bad_request`, `unauthorized`, `forbidden`, `not_found`, `slot_full`, `double_booking`, `slot_cancelled`, `slot_started`, `already_cancelled`, `invalid_code`, `idempotency_conflict`, `phone_conflict`, `too_many_requests`, `internal_error`.
- Проверить, что все handlers возвращают `application/json` и тело `Error` для ошибок.
- Добавить request validation для всех path/query/body параметров из OpenAPI.

Готово, когда:
- Contract tests по каждому endpoint проверяют основные success/error статусы.
- OpenAPI examples не противоречат реальным ответам.

### BE-11. Добавить полный набор Go-тестов

Сделать:
- Unit tests для domain services: availability, price calculation, cancellation rule, auth OTP.
- Repository integration tests на PostgreSQL.
- HTTP integration tests на все endpoints.
- Race/concurrency tests для create/cancel.

Готово, когда:
- `go test ./...` проходит стабильно.
- `go test -race ./...` проходит или явно выделены дорогие suites с отдельной командой.

### BE-12. Добавить k6 performance tests до 300 concurrent users

Сделать:
- `k6/smoke.js`: быстрый прогон auth/list/create/cancel на малой нагрузке.
- `k6/booking_300_vu.js`: конкурентное бронирование одного/нескольких слотов до 300 VUs.
- `k6/cancel_300_vu.js`: конкурентные отмены с проверкой, что места не освобождаются дважды.
- Подготовить seed data и test users для воспроизводимого запуска.
- Задать thresholds для latency/error rate согласно NFR-6/NFR-7.

Готово, когда:
- Smoke сценарий запускается локально одной командой.
- 300 VU сценарий не выявляет overbooking, negative availability или duplicate bookings.

### BE-13. Подготовить локальный запуск и документацию разработчика

Сделать:
- `backend/README.md`: setup, env vars, migrations, seed, run, tests, k6.
- `.env.example` без секретов.
- Docker Compose profiles для app, db, migrations, k6.
- Описать, что `routes/instructors/slots` read-only и в dev заполняются seed data.

Готово, когда:
- Новый разработчик поднимает API с нуля по README.
- Все команды из README проверены локально.

### BE-14. Финальная проверка готовности BE

Сделать:
- Прогнать OpenAPI lint/bundle.
- Прогнать Go format/lint/test/race.
- Прогнать HTTP integration tests и k6 smoke.
- Прогнать k6 300 VU на подготовленном окружении.
- Сверить endpoints с таблицей в начале файла и с `01-analysis/5-mobile-app-spec/*.md`.

Готово, когда:
- Все endpoints из OpenAPI реализованы.
- Booking/cancel выдерживают параллельные запросы без double booking и overbooking.
- Нет реализации функционала вне MVP scope без явного изменения требований.
