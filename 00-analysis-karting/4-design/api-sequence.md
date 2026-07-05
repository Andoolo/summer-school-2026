# Sequence-диаграмма API-взаимодействия

> **Этап 3. Проектирование.** Как клиент и сервер обмениваются вызовами в критичных сценариях
> бронирования. Контракты API — в многофайловой спецификации
> [`../api/`](../api/) (домены `bookings`, `slots`, `auth`).
> Операции: `createBooking`, `cancelBooking`.

> **Сквозные правила взаимодействия.**
> - Все вызовы — с `Authorization: Bearer <token>` (`bearerAuth`); при истёкшем/неверном токене
>   сервер отвечает `401`, клиент уходит на вход [SCR-001](../3-design-brief/SCR-001-registration.md).
> - Сервер — **источник истины** по времени и доступности: `slot.start_at` в UTC, тип отмены и
>   наличие мест-картов/прокатной экипировки проверяет сервер, клиент их не пересчитывает
>   (R-005, R-021).
> - Запись/отмена **атомарны**: овербукинг и двойная бронь исключены (NFR-8, NFR-9).
> - Таймаут запроса ~10 с; мутации офлайн запрещены — см. единый паттерн Error/Retry (R-020).

## Сценарий 1: Создание брони (`createBooking`, UC-1)

Поток: [SCR-004 «Оформление записи»](../3-design-brief/SCR-004-booking.md) → `POST /bookings`
→ [BS-002 «Подтверждение»](../3-design-brief/BS-002-booking-success.md). Клиент отправляет
`slot_id`, `seats_count` (1..3) и `rental_count` (0..seats_count). Итоговую цену `price_total`
(RUB, read-only) считает сервер — клиент её не вычисляет, а показывает (R-005, R-010).

```mermaid
sequenceDiagram
    actor User as Клиент
    participant App as Приложение
    participant API as API (bookings)

    Note over App: SCR-004: выбраны места-карты/экипировка,<br/>цена показана из price_total слота
    User->>App: Тап «Записаться»
    App->>App: Генерирует Idempotency-Key (UUID)

    App->>API: POST /bookings<br/>{slot_id, seats_count, rental_count}<br/>Authorization: Bearer, Idempotency-Key
    Note over API: Атомарно: проверка свободных мест-картов/<br/>экипировки, фиксация цены, списание (NFR-8/9)

    alt Успех
        API-->>App: 201 Booking {id, status: active,<br/>price_total, created_at, slot}
        App-->>User: BS-002 «Вы записаны» + сводка<br/>(после первой записи — запрос push)
    else Нет мест-картов/экипировки (409 Conflict)
        API-->>App: 409 {code: slot_full,<br/>available_seats, available_rental_kits}
        App-->>User: Сообщение о нехватке мест/экипировки,<br/>обновление доступности слота
    else Двойная бронь (409 Conflict)
        API-->>App: 409 {code: double_booking, booking_id}
        App-->>User: «У вас уже есть бронь» → переход в SCR-005
    else Слот отменён центром (410 Gone)
        API-->>App: 410 {code: slot_cancelled}
        App-->>User: «Заезд отменён центром», запись недоступна
    else Невалидные данные (400 / 422)
        API-->>App: 400 BadRequest / 422 Unprocessable
        App-->>User: Подсказка по полям / правилу
    else Токен истёк (401)
        API-->>App: 401 Unauthorized
        App->>App: refresh; при неуспехе — очистка токенов
        App-->>User: Переход на вход (SCR-001)
    else Сеть/сервер/таймаут (~10 c, 5xx)
        API-->>App: Ошибка / нет ответа
        App-->>User: Error + «Повторить» (повтор с тем же<br/>Idempotency-Key — без двойной брони)
    end
```

| Шаг | Что происходит | Источник |
| :-- | :-- | :-- |
| Запрос | `POST /bookings` с `Idempotency-Key`; тело — `CreateBookingRequest` | bookings/api.yaml |
| Проверка | Сервер атомарно проверяет места-карты/прокатную экипировку и фиксирует цену слота | NFR-8/9, R-010 |
| `201` | Возвращается `Booking` со `status=active` и `price_total` (read-only) | R-005 |
| `409` (`slot_full`) | Нет мест-картов/экипировки; тело несёт `available_seats`/`available_rental_kits` | common/models.yaml |
| `409` (`double_booking`) | Бронь на слот уже есть; тело несёт `booking_id` | error-matrix |
| `410` | Слот отменён центром (`slot_cancelled`) | R-008 |
| Повтор | Сетевой сбой → повтор с тем же `Idempotency-Key` исключает дубль | NFR-9, R-020 |

## Сценарий 2: Отмена брони (`cancelBooking`, UC-2)

Поток: [SCR-006 «Детали брони»](../3-design-brief/SCR-006-booking-details.md) →
[BS-003 «Подтверждение отмены»](../3-design-brief/BS-003-cancel-confirm.md) →
`POST /bookings/{bookingId}/cancel`. Отмена — **только целиком** (R-014). **Тип отмены определяет
сервер** по времени до старта (источник истины — `slot.start_at` в UTC, граница 2 ч — в зоне
центра): `≥ 2 ч` → `cancelled` (места-карты и прокат возвращаются в слот), `< 2 ч` → `late_cancel`
(не возвращаются, штрафов нет). Граница «ровно 2 часа» трактуется как ранняя отмена (R-021).

```mermaid
sequenceDiagram
    actor User as Клиент
    participant App as Приложение
    participant API as API (bookings)

    Note over App: SCR-006: бронь active, старт в будущем
    User->>App: Тап «Отменить запись»
    App-->>User: BS-003 «Подтверждение отмены»<br/>(канон правила 2 часов)
    User->>App: Подтверждает отмену (целиком)

    App->>API: POST /bookings/{bookingId}/cancel<br/>Authorization: Bearer
    Note over API: Сервер по slot.start_at выбирает<br/>тип отмены; граница ровно 2ч = ранняя

    alt Ранняя отмена (≥ 2 ч)
        API-->>App: 200 Booking {status: cancelled, cancelled_at}
        App-->>User: SCR-006 + снек «Бронь отменена»<br/>(места/экипировка вернулись в слот)
    else Поздняя отмена (< 2 ч)
        API-->>App: 200 Booking {status: late_cancel, cancelled_at}
        App-->>User: SCR-006 + «Поздняя отмена: ваше место<br/>в заезде сохранено. Штраф не взимается.»
    else Слот уже стартовал (422 Unprocessable)
        API-->>App: 422 {code: slot_started}
        App-->>User: Отмена недоступна после старта
    else Уже отменена (409 Conflict)
        API-->>App: 409 {code: already_cancelled}
        App-->>User: Бронь уже отменена, статус актуализируется
    else Чужая/несуществующая бронь (403 / 404)
        API-->>App: 403 Forbidden / 404 NotFound
        App-->>User: Бронь недоступна
    else Токен истёк (401)
        API-->>App: 401 Unauthorized
        App-->>User: Переход на вход (SCR-001)
    else Сеть/сервер/таймаут (~10 c, 5xx)
        API-->>App: Ошибка / нет ответа
        App-->>User: Снек ошибки на BS-003, шторка остаётся<br/>открытой — можно повторить
    end
```

| Шаг | Что происходит | Источник |
| :-- | :-- | :-- |
| Запрос | `POST /bookings/{bookingId}/cancel` (без тела; отмена целиком) | R-014, bookings/api.yaml |
| Решение | Сервер выбирает `cancelled` / `late_cancel` по `start_at` (граница — зона центра) | R-021 |
| `200` | `Booking` с новым `status` и `cancelled_at`; экран обновляется | модель данных |
| `422` | Слот уже стартовал (`slot_started`) — отмена недоступна | UC-2 E1 |
| `409` | Повторная отмена (`already_cancelled`) — терминальный статус | UC-2 E2 |

> Полная модель состояний брони и инварианты освобождения мест-картов/прокатной экипировки —
> в [data-model.md §«Модель состояний»](data-model.md#модель-состояний-жизненный-цикл).
