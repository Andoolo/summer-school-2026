# Реестр проектных решений (R-XXX, RR-Dxx)

> Единая точка разрешения ссылок на решения `R-NNN` и дизайн-ревью `RR-Dxx`, разбросанные по всем
> артефактам аналитики (~260 ссылок). Каждое решение зафиксировано инлайн в источнике; здесь —
> краткий указатель для быстрой навигации.
>
> Нумерация стабильна; пропуски — следствие сужения скоупа (часть решений относится к
> существующей инфраструктуре и в клиентскую поставку не входит).

## Ключевые проектные решения (R-NNN)

| ID | Решение | Где зафиксировано |
| :-- | :-- | :-- |
| R-001 | Оценки маршалов вынесены в Phase 2 (согласовано с заказчиком 2026-06-30) | [бриф](../0-customer-brief/customer-brief.md), [BR](business-requirements.md), [FR](functional-requirements.md) |
| R-002 | Нативное/гибридное приложение iOS/Android через сторы | [NFR-4](non-functional-requirements.md), [foundations §1](../3-design-brief/00-foundations.md) |
| R-003 | Раздельная модель доступности: места-карты и прокатная экипировка — два независимых лимита | [домен](../1-elicitation/domain-description.md), [FR-13/14](functional-requirements.md) |
| R-004 | Граница серверной интеграции: бэкенд — black-box источник истины, гарантия 0 двойных броней на его стороне | [бриф](../0-customer-brief/customer-brief.md), [домен](../1-elicitation/domain-description.md) |
| R-005 | `price_total` рассчитывает и возвращает сервер (read-only); клиент не пересчитывает | [FR-45](functional-requirements.md), [data-model](../4-design/data-model.md) |
| R-006 | Канал уведомлений MVP — системный push (APNs/FCM) | [FR-33](functional-requirements.md), [NFR-17](non-functional-requirements.md), [foundations §10](../3-design-brief/00-foundations.md) |
| R-007 | Вход/подтверждение по SMS OTP | [FR-43](functional-requirements.md), [UC-4](use-cases.md) |
| R-008 | Отмена слота центром → бронь в статус `club_cancelled` + push; повторная запись запрещена | [FR-46](functional-requirements.md), [домен](../1-elicitation/domain-description.md) |
| R-009 | Управление профилем/аккаунтом; удаление → анонимизация ПДн, освобождение ресурсов | [FR-47–49](functional-requirements.md), [UC-5](use-cases.md) |
| R-010 | Модель цены: `price × seats_count + rental_price × rental_count`, RUB; цена фиксируется на момент брони | [FR](functional-requirements.md), [data-model](../4-design/data-model.md) |
| R-011 | Место сбора и геометрия трассы обязательны; Яндекс.Карты + fallback | [FR-44](functional-requirements.md), [NFR-26](non-functional-requirements.md), [foundations §8](../3-design-brief/00-foundations.md) |
| R-012 | Заполненные/отменённые слоты не скрываются, CTA disabled | [FR-9](functional-requirements.md), [SCR-002](../3-design-brief/SCR-002-slot-list.md) |
| R-013 | По гостям — только количество мест и выбор экипировки (без имён/контактов/возраста) | [FR-12](functional-requirements.md), [UC-1](use-cases.md) |
| R-014 | Отмена только целиком (нет частичной отмены гостя) | [FR-16](functional-requirements.md), [UC-2](use-cases.md) |
| R-015 | Каноническая схема — контракт API; миграция/backfill вне скоупа (учебный проект) | [домен](../1-elicitation/domain-description.md), [data-model](../4-design/data-model.md) |
| R-016 | Сессия JWT access/refresh; 401-flow с refresh; TLS | [NFR-18](non-functional-requirements.md) |
| R-017 | Базовые требования к ПДн (согласие, маскирование, retention 12 мес); полный 152-ФЗ — Phase 2 | [NFR-20](non-functional-requirements.md) |
| R-018 | Измеримые метрики производительности (p95) и нагрузочный сценарий на гонку бронирований | [NFR-21/22](non-functional-requirements.md) |
| R-019 | Доступность 99% в рабочие часы центра (зона центра); исключения и деградация | [NFR-23](non-functional-requirements.md) |
| R-020 | Offline-просмотр кэша; мутации офлайн запрещены; единый Error/Retry; таймаут ~10 с | [NFR-24](non-functional-requirements.md), [foundations §5](../3-design-brief/00-foundations.md) |
| R-021 | Правило таймзоны: UTC хранение; бизнес-правила — зона центра; отображение — зона устройства; сервер — источник истины | [домен](../1-elicitation/domain-description.md), [FR-17](functional-requirements.md), [use-cases](use-cases.md) |
| R-022 | Идемпотентность `createBooking` через `Idempotency-Key` | [UC-1](use-cases.md), [bookings API](../api/bookings/api.yaml) |
| R-023 | Матрица ошибок API (код → HTTP → details → UX) | [error-matrix.md](error-matrix.md) |
| R-025 | Клиенту доступна вся история броней (пагинация) | [FR-35a](functional-requirements.md), [data-model](../4-design/data-model.md) |
| R-026 | Семантика фильтров: внутри группы ИЛИ, между группами И; границы дат включительные | [FR-38](functional-requirements.md), [BS-001](../3-design-brief/BS-001-filters.md) |
| R-027 | Дефолт списка слотов — ближайшие 7 дней; больший период фильтром | [FR-9](functional-requirements.md), [SCR-002](../3-design-brief/SCR-002-slot-list.md) |
| R-028 | Текущая поставка — только роль Клиент | [бриф](../0-customer-brief/customer-brief.md), [домен](../1-elicitation/domain-description.md) |
| R-029 | Критерии доступности (a11y): WCAG 2.1 AA, ≥44pt, Dynamic Type | [NFR-25](non-functional-requirements.md) |
| R-030 | Разрывы в нумерации NFR — намеренны (сужение скоупа) | [NFR](non-functional-requirements.md) |

## Находки дизайн-ревью (RR-Dxx)

| ID | Решение | Где учтено |
| :-- | :-- | :-- |
| RR-D03 | BS-002 — полноэкранный экран успеха, не Bottom Sheet (ID сохранён) | [BS-002](../3-design-brief/BS-002-booking-success.md) |
| RR-D06 | «Поделиться заездом» — Won't/Phase 2; иконка декоративна | [FR](functional-requirements.md), [SCR-003](../3-design-brief/SCR-003-slot-card.md) |
| RR-D07 | Геометрия трассы и место сбора обязательны | [FR-44](functional-requirements.md), [SCR-003](../3-design-brief/SCR-003-slot-card.md) |
| RR-D08 | Фильтр дат: пресеты + кастомный диапазон | [FR-38](functional-requirements.md), [BS-001](../3-design-brief/BS-001-filters.md) |
