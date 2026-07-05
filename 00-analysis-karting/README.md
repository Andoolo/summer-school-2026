# Блок 1 · Аналитика · «Апекс» (картинг-центр)

Артефакты аналитика по сквозному проекту летней школы. Структура повторяет классический процесс
работы аналитика: от входа заказчика до ТЗ и API-контракта, которые передаются в **Блок 2**
(разработка).

## Маршрут по этапам

| Этап | Папка | Что внутри |
| :-- | :-- | :-- |
| **Вход** | [0-customer-brief/](0-customer-brief/) | [customer-brief.md](0-customer-brief/customer-brief.md) — сырой бриф заказчика + уточнения скоупа |
| **1. Выявление требований** | [1-elicitation/](1-elicitation/) | [customer-questions.md](1-elicitation/customer-questions.md) (24 вопроса + ответы), [domain-description.md](1-elicitation/domain-description.md) (домен, ограничения, глоссарий) |
| **2. Описание требований** | [2-requirements/](2-requirements/) | [business](2-requirements/business-requirements.md) · [functional](2-requirements/functional-requirements.md) · [non-functional](2-requirements/non-functional-requirements.md) · [user-stories](2-requirements/user-stories.md) · [use-cases](2-requirements/use-cases.md) · [error-matrix](2-requirements/error-matrix.md) · [decisions-registry](2-requirements/decisions-registry.md) |
| **Бриф для дизайна** | [3-design-brief/](3-design-brief/) | [00-foundations.md](3-design-brief/00-foundations.md) (сквозные правила) + [design-brief.md](3-design-brief/design-brief.md) + 11 экранов (SCR-001…007, BS-001…004) |
| **3. Проектирование** | [4-design/](4-design/) | [data-model.md](4-design/data-model.md) (сущности, ERD, модель состояний), [api-sequence.md](4-design/api-sequence.md) (sequence-диаграммы) |
| **4. ТЗ** | [5-mobile-app-spec/](5-mobile-app-spec/) | [README.md](5-mobile-app-spec/README.md) (индекс + логики), [feature-list.md](5-mobile-app-spec/feature-list.md) (карта навигации, инвентарь экранов, трассировка) |
| **API (OpenAPI)** | [api/](api/) | [openapi.yaml](api/openapi.yaml) — многофайловый OpenAPI (домены: auth, profile, slots, bookings, marshals) |
| **Дополнительно** | [prompts/](prompts/), [checklists/](checklists/) | Все промпты ИИ (требование задания); чек-лист цифровой гигиены |

## Статус

✅ **Блок аналитики завершён.** Прошёл 3 раунда ревью сабагентами:
1. Требования (BR/FR/NFR/US/UC) — 3 критичных + 5 средних замечаний закрыты.
2. Дизайн-брифы 11 экранов — 8 средних замечаний закрыты (критичных не найдено).
3. Финальная проверка целостности — 3 кросс-расхождения (битые `$ref`, дубликаты путей,
   остаточное противоречие таймзоны в UC) + реестр решений — закрыты.

**Объём:** 42 файла. **Ключевое доменное решение:** карт всегда центра → «своя/прокатная»
относится к экипировке (шлем + подшлемник); места-карты (`free_seats`) и прокатная экипировка
(`free_rental_kits`) — два независимых лимита.

> **Передача в Блок 2:** итоговые требования + модель данных + API-спецификация + дизайн-брифы
> экранов + ТЗ.
