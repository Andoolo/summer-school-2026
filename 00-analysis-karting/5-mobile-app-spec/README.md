# ТЗ на мобильное приложение «Апекс»

> **Этап 5.** Детальное техническое задание на клиентское мобильное приложение «Апекс»
> (самостоятельная запись на групповые заезды, роль «Клиент»).

**Статус:** Актуален · **Версия:** 1.0 · **Дата:** 2026-07-03

ТЗ опирается на [фича-лист](feature-list.md) и [дизайн-брифы по экранам](../3-design-brief/):
каждый экран/шторка детально описаны в отдельном документе, переиспользуемая логика — в
[foundations](../3-design-brief/00-foundations.md).

**Источники:**
[Фича-лист](feature-list.md) ·
[Дизайн-брифы](../3-design-brief/) ·
[Функциональные требования](../2-requirements/functional-requirements.md) ·
[Нефункциональные требования](../2-requirements/non-functional-requirements.md) ·
[Use cases](../2-requirements/use-cases.md) ·
[User stories](../2-requirements/user-stories.md) ·
[Модель данных](../4-design/data-model.md) ·
[Матрица ошибок](../2-requirements/error-matrix.md) ·
[API (OpenAPI, многофайловый)](../api/)

---

## Экраны и шторки

| ID | Экран / Шторка | Тип | Зона | Приоритет | Дизайн-бриф |
|----|----------------|-----|------|-----------|-----|
| SCR-001 | Регистрация / Вход | Экран | НЗ | Critical | [SCR-001-registration.md](../3-design-brief/SCR-001-registration.md) |
| SCR-002 | Список слотов | Экран | АЗ | Critical | [SCR-002-slot-list.md](../3-design-brief/SCR-002-slot-list.md) |
| BS-001 | Фильтры | Bottom Sheet | АЗ | High | [BS-001-filters.md](../3-design-brief/BS-001-filters.md) |
| SCR-003 | Карточка слота | Экран | АЗ | Critical | [SCR-003-slot-card.md](../3-design-brief/SCR-003-slot-card.md) |
| SCR-004 | Оформление записи | Экран | АЗ | Critical | [SCR-004-booking.md](../3-design-brief/SCR-004-booking.md) |
| BS-002 | Подтверждение записи («Вы записаны») | Экран | АЗ | High | [BS-002-booking-success.md](../3-design-brief/BS-002-booking-success.md) |
| SCR-005 | Мои бронирования | Экран | АЗ | Critical | [SCR-005-my-bookings.md](../3-design-brief/SCR-005-my-bookings.md) |
| SCR-006 | Детали брони + отмена | Экран | АЗ | Critical | [SCR-006-booking-details.md](../3-design-brief/SCR-006-booking-details.md) |
| BS-003 | Подтверждение отмены | Bottom Sheet | АЗ | High | [BS-003-cancel-confirm.md](../3-design-brief/BS-003-cancel-confirm.md) |
| BS-004 | Карта трассы | Bottom Sheet | АЗ | Medium | [BS-004-route-map.md](../3-design-brief/BS-004-route-map.md) |
| SCR-007 | Профиль клиента | Экран | АЗ | Medium | [SCR-007-profile.md](../3-design-brief/SCR-007-profile.md) |

> **Зоны:** НЗ — неавторизованная, АЗ — авторизованная.

## Переиспользуемые логики

Бизнес- и UI-логика, общая для нескольких экранов (на которые ссылаются дизайн-брифы):

| Логика | Где применяется | Суть |
| :-- | :-- | :-- |
| LOGIC-001 OTP-авторизация | SCR-001, SCR-007 | Запрос/проверка SMS-кода, антифрод (rate-limits), хранение токенов в Keychain/Keystore |
| LOGIC-002 Расчёт доступности | SCR-003, SCR-004 | `max_seats = min(free_seats, track_config.capacity_cap, 3)`; `rental_count ≤ free_rental_kits` (два независимых лимита) |
| LOGIC-003 Расчёт цены | SCR-004 | Отображение серверного `price_total` (`price × seats_count + rental_price × rental_count`), без локального пересчёта |
| LOGIC-004 Правило 2 часов отмены | SCR-006, BS-003 | Граница 2 ч (зона центра); тип отмены определяет сервер (R-021) |
| LOGIC-005 Фильтрация слотов | SCR-002, BS-001 | Семантика И/ИЛИ, пресеты дат, пустой справочник маршалов |
| LOGIC-006 Карта трассы | SCR-003, SCR-006, BS-004 | Яндекс.Карты (`geometry` + `meeting_point`), fallback на текст |
| LOGIC-007 Запрос push-разрешения | BS-002 | После первой записи; при отказе — штатная работа без push |
| LOGIC-008 Паттерн состояний экрана | SCR-002, SCR-003, SCR-005, SCR-006 | Loading → Content → Empty → Error (+ offline) |

## Навигация

Полная карта переходов между экранами — [feature-list.md §3](feature-list.md). У каждого ТЗ
секции «Навигация» (входящая/исходящая) согласованы с этой картой.

## Соглашения

- **Платформа:** **нативное мобильное приложение** (iOS + Android) — клиент устанавливается из
  сторов. Это даёт доступ к нативным возможностям, на которые опираются ТЗ: системный push
  (LOGIC-007) и хранение сессионного токена в защищённом хранилище ОС (Keychain / Keystore,
  LOGIC-001).
- **API:** все запросы REST, спецификации — [`../api/`](../api/) (домены `auth`, `profile`,
  `slots`, `bookings`, `marshals`). В ТЗ указываются точные `operationId`, метод и путь.
- **Числа не хардкодятся:** потолки конфигураций, прокатный фонд экипировки, цены, лимиты приходят
  из данных слота/конфигурации.
- **Сервер — источник истины:** цена (`price_total`), число мест-картов, статус отмены — из API;
  клиент не пересчитывает (FR-45, R-021).
- **Время:** хранение/передача — UTC; бизнес-правила (правило 2 ч, окно выдачи) — в зоне центра;
  отображение — в зоне устройства клиента (R-021).
- **Карты:** ключ Яндекс.Карт — параметр окружения (не в коде/репозитории); при недоступности —
  текстовый fallback (FR-44, NFR-26).
