# F1 · Фича: отдавать `route.geometry` в ответах `/slots`

> **Тип:** фича (закрывает нарушение контракта) · **Слой:** backend (storage + handler) · **Сложность:** низкая · **Статус:** ✅ реализовано

## Потребность

Клиенту нужна геометрия маршрута (ломаная координат), чтобы нарисовать линию прогулки на карте
в карточке слота (SCR-003) — это вход для фичи **F2 (карта по geometry)**. По контракту поле
`geometry` у схемы `Route` — **обязательное**, но бэкенд его не возвращал.

## Причина (почему не было)

`Route` (в `01-analysis/api/instructors/models.yaml`) требует `geometry` (в списке `required`),
и обе схемы `Slot`/`SlotSummary` ссылаются на этот `Route`. Однако:

- `backend/internal/storage/postgres/slots.go` — структура `Slot` не имела поля geometry, а запросы
  `List`/`GetByID` не выбирали колонку `r.geometry` (хотя в БД `routes.geometry jsonb` заполнена сидом).
- `backend/internal/http/handlers/catalog.go` (`slotBase`) собирал `Route` без `Geometry`.

Итог: и `listSlots`, и `getSlot` отдавали `route` без обязательного `geometry` — нарушение контракта
и блокер для карты на клиенте.

## Требования

- Контракт: `Route.geometry` (required) — `01-analysis/api/instructors/models.yaml`.
- `getSlot` (`01-analysis/api/slots/api.yaml`): «Полные данные слота для карточки (SCR-003):
  маршрут с geometry…».
- SCR-003 (карточка слота) — линия маршрута на карте; вход для F2.

## Реализация

Поле geometry хранится в БД как `jsonb`-массив координат `[[lat,lng],…]`. Прокинул его сквозь слои,
не трогая доменную модель:

**1. `internal/storage/postgres/slots.go`**
- В `Slot` добавлено поле `RouteGeometry []byte` (сырой jsonb).
- В SELECT обоих запросов (`List`, `GetByID`) добавлена колонка `r.geometry`; в `Scan` —
  `&slot.RouteGeometry`.

**2. `internal/http/handlers/catalog.go` (`slotBase`)**
- Если geometry не пустой — распаковываем `[][]float32` и кладём в union-тип
  `slotsapi.Geometry` через `FromGeometry0` (вариант «массив координат»):
```go
if len(slot.RouteGeometry) > 0 {
    var coords [][]float32
    json.Unmarshal(slot.RouteGeometry, &coords)
    var geometry slotsapi.Geometry
    geometry.FromGeometry0(coords)
    route.Geometry = &geometry
}
```
- Так как `slotBase` строит общий `Route` и для списка, и для детали — одно изменение покрывает
  оба эндпоинта.

> Union `Geometry` по контракту допускает и закодированную полилинию (строку). Сейчас в БД и сиде —
> массивы координат, поэтому реализован вариант `Geometry0` (`[][]float32`); строковый вариант можно
> добавить, если появится encoded-polyline источник.

## Промпты ИИ

> **Промпт:** В контракте `Route.geometry` — обязательное поле для карты (SCR-003), но `/slots` и
> `/slots/{id}` его не отдают. В БД `routes.geometry` (jsonb, `[[lat,lng],…]`) заполнена.
> Как прокинуть geometry из БД в ответ, не таща jsonb в доменную модель, и корректно положить в
> сгенерированный union-тип `Geometry`?
>
> **Действие по подсказке:** сырой `[]byte` из `r.geometry` в storage → в handler распаковать в
> `[][]float32` и `FromGeometry0`. Одно место маппинга (`slotBase`) на оба эндпоинта.

## Проверка (ручная)

Собран и запущен API из образа (`docker compose --profile app up -d --build api`, Go 1.25 —
сборка всего бэка прошла). Auth-флоу в dev → токен → запросы:

```bash
# list — route.geometry присутствует:
curl -s localhost:8080/slots -H "Authorization: Bearer $TOKEN"
# → route: { ..., "geometry": [[59.978,30.262],[59.981,30.271],[59.976,30.285]], ... }

# detail (getSlot, SCR-003):
curl -s localhost:8080/slots/55555555-5555-5555-5555-555555555555 -H "Authorization: Bearer $TOKEN"
# → route.geometry = [[59.978,30.262],[59.981,30.271],[59.976,30.285]]
```

Фактический вывод detail-запроса:
```text
route.id       = 11111111-1111-1111-1111-111111111111
geometry       = [[59.978, 30.262], [59.981, 30.271], [59.976, 30.285]]
```

✅ И `listSlots`, и `getSlot` возвращают `route.geometry`; координаты совпадают с сидом маршрута
«Острова и каналы». Контракт (required geometry) выполнен; F2 разблокирована.

## Инструменты

- ИИ: Claude Code.
- Среда: Docker Desktop (Postgres 16 + API-образ Go 1.25), `curl`, dev-seed.
- Проверка: сборка образа + живой auth-флоу + `/slots` и `/slots/{id}`.

## Файлы

- `backend/internal/storage/postgres/slots.go` — поле `RouteGeometry`, `r.geometry` в 2 запросах и scan.
- `backend/internal/http/handlers/catalog.go` — маппинг geometry в `Route.Geometry` (union `Geometry0`).
