# Промпты · Этап API. OpenAPI-контракт

## Промпт 9 — OpenAPI-спецификация

> **Промпт (отправлен ИИ):**
>
> Спроектируй многофайловую OpenAPI 3.1 спецификацию клиентского контракта «Апекс»
> (Redocly-совместимую). Домены: auth (SMS OTP requestCode/verifyCode, refresh, logout), profile
> (getMe/updateMe/deleteAccount, смена телефона с OTP), slots (listSlots с фильтрами + getSlot),
> bookings (createBooking с Idempotency-Key, listBookings с пагинацией, cancelBooking), marshals
> (справочник). Общие модели в common: Error (code+details), Pagination, Money (RUB копейки),
> статусы.
>
> Сквозные правила: bearerAuth (кроме code/verify); единая модель Error; коды синхронизированы с
> матрицей ошибок (slot_full → available_seats/available_rental_kits, double_booking → booking_id,
> slot_cancelled 410, slot_started 422, already_cancelled 409); время ISO-8601 UTC.
>
> Секреты (ключи Яндекс.Карт, APNs/FCM, SMS-провайдер) НЕ описывай в контракте — это параметры
> окружения.
>
> createBooking: тело {slot_id, seats_count 1..3, rental_count 0..seats_count}; ответ 201 Booking
> с price_total (read-only). cancelBooking: без тела; тип отмены определяет сервер.
>
> **Результат:** 8 файлов, все FR/UC покрыты эндпоинтами, коды ошибок синхронны с error-matrix.md.
