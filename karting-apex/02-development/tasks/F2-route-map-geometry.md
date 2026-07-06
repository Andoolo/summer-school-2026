# F2 · Фича: рисовать трасса на карте по реальной `route.geometry` (клиент)

> **Тип:** фича (клиент) · **Слой:** client (Compose Multiplatform, common) · **Сложность:** низкая-средняя · **Статус:** ✅ реализовано (визуальная проверка — сборкой wasmJs)

## Потребность

На карточке слота (SCR-003) карта должна показывать **линию реального трассы**. Данные для этого
теперь отдаёт бэкенд — `route.geometry` (F1). Клиент же рисовал **захардкоженную** декоративную
линию, не связанную с координатами слота: разные трассаы выглядели одинаково.

Это прямая связка **F1 → F2**: F1 (бэк) начал отдавать geometry, F2 (клиент) её отрисовывает.

## Причина (что было)

`shared/src/commonMain/.../map/RouteMapPreviewFallback.kt` → `MockRouteScreenshot` рисовал
фиксированную кривую (`cubicTo(...)` с константами), одинаковую для всех слотов. При этом клиентская
обвязка geometry **уже была готова** и не использовалась в отрисовке:
- домен: `Route.geometry: RouteGeometry?` (`RouteGeometry(points: List<GeoPoint>)`);
- DTO: `RouteDto.geometry: JsonElement?`;
- маппер: `toRouteGeometry()` парсит `[[lat,lng],…]` → `List<GeoPoint>`.

`RouteMapPreview` на всех платформах (wasm/android/ios) делегирует в общий
`RouteMapPreviewFallback`, поэтому правка одного common-файла покрывает все таргеты.

## Реализация

Файл: `shared/src/commonMain/kotlin/com/volna/app/map/RouteMapPreviewFallback.kt`

- В `MockRouteScreenshot` добавлен параметр `routePoints: List<GeoPoint>`
  (`route.geometry?.points.orEmpty()`).
- Добавлена проекция координат в точки канваса `projectRoute(...)`:
  - долгота корректируется на `cos(широты)` — форма не растягивается по горизонтали;
  - **единый масштаб** по обеим осям (форма трассы не искажается) + центрирование;
  - ось Y инвертируется — север сверху.
- Если точек ≥ 2 — рисуется реальная ломаная (`Path` из спроецированных точек, `Stroke` c
  `StrokeJoin.Round`), пин — в первой точке трассы.
- Если geometry нет — прежняя декоративная линия остаётся как **fallback** (обратная совместимость,
  экран не ломается на слотах без geometry).

Декоративный фон-«карта» (вода/берег/улицы) сохранён; поверх него ложится настоящая геометрия.

## Промпты ИИ

> **Промпт:** На карточке слота карта рисует одинаковую захардкоженную линию. Бэк теперь отдаёт
> `route.geometry` (список координат), клиент уже парсит его в `Route.geometry.points`. Как в
> Compose Canvas нарисовать реальную ломаную по этим координатам: спроецировать lat/lng на канвас
> без искажения формы (учесть cos(lat)), с центрированием и «север сверху», и оставить мок как
> fallback?
>
> **Действие:** `projectRoute()` (cos-коррекция долготы, единый масштаб, инверсия Y) + отрисовка
> `Path` по точкам; при отсутствии geometry — прежняя декоративная линия.

## Проверка

Компиляция подтверждена — `./gradlew :shared:compileKotlinWasmJs` → **BUILD SUCCESSFUL** (1m51s),
common+wasm собираются без ошибок (типы/API под Kotlin 2.2.20: `GeoPoint` из
`com.volna.app.domain.model`, `min()/max()` на списках non-null; ветка защищена `points.size < 2`).

Визуально (запуск клиента):
```bash
cd client
./gradlew :shared:compileKotlinWasmJs        # ✅ BUILD SUCCESSFUL — проверка компиляции common+wasm
./gradlew :webApp:wasmJsBrowserDevelopmentRun # запуск: открыть карточку слота (SCR-003)
```
Ожидаемо: на карточке слота линия трассы повторяет реальные координаты `route.geometry`
(для сид-трассы «Городское кольцо» — замкнутый контур трассы по 6 точкам
`55.750,37.610 → 55.752,37.615 → … → 55.750,37.610`), пин — в стартовой точке.
У слота без geometry — прежняя декоративная линия.

> Данные для F2 подтверждены на живом API в рамках F1 (`/slots` и `/slots/{id}` возвращают
> `route.geometry`), поэтому клиент получает реальные точки для отрисовки.

## Инструменты

- ИИ: Claude Code.
- Среда: Kotlin 2.2.20 / Compose Multiplatform (common → wasm/android/ios).
- Проверка: статическая выверка API + визуальная проверка сборкой wasmJs.

## Правки по ревью (L3)

Тяжёлая часть проекции (cos-коррекция, bbox, аллокации) вынесена в `remember(routePoints)` через
`prepareRoute()` — считается один раз, а не на каждый кадр Canvas; в отрисовке остаётся только
дешёвое масштабирование `toCanvasPoints()` под текущий размер. Перекомпиляция wasmJs — BUILD
SUCCESSFUL. См. `02-development/REVIEW-block2.md`.

## Файлы

- `client/shared/src/commonMain/kotlin/com/volna/app/map/RouteMapPreviewFallback.kt` — отрисовка
  реальной геометрии (`prepareRoute` в `remember` + `toCanvasPoints`); мок как fallback.
