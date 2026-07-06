# План реализации CMP-клиента для «Апекса»

## Контекст анализа

Документ проектирует клиентское приложение «Апекс» для Android, iOS и Web на Kotlin Compose Multiplatform. Основа: чистая архитектура, MVI, контрактная интеграция с существующим Go BE и OpenAPI из `01-analysis/api`.

Источники:

- `01-analysis/5-mobile-app-spec/README.md` и экранные ТЗ `SCR-*`, `BS-*`.
- Переиспользуемые логики `01-analysis/5-mobile-app-spec/09_Логики/LOGIC-*`.
- API-контракты `01-analysis/api/{auth,slots,bookings,profile,instructors}/`.
- Backend implementation under `backend/`.
- `02-development/BE_IMPLEMENTATION_PLAN.md`.
- Figma file: `https://www.figma.com/design/ySEt0cjmRqmhdWyDlTpDM5/...`. В текущей сессии содержимое Figma недоступно через инструмент, поэтому точные токены темы нужно извлечь отдельным шагом после доступа к файлу.

## Выводы по требованиям

MVP-клиент покрывает только роль клиента:

- `SCR-001` вход/регистрация по телефону и OTP.
- `SCR-002` список слотов с фильтрами.
- `BS-001` фильтры.
- `SCR-003` карточка слота.
- `SCR-004` оформление брони.
- `BS-002` успех бронирования и запрос push-разрешения.
- `SCR-005` мои бронирования.
- `SCR-006` детали брони и отмена.
- `BS-003` подтверждение отмены.
- `BS-004` карта трассы.
- `SCR-007` профиль, выход, операции профиля.

Клиент не должен реализовывать instructor/admin UI, schedule CRUD, создание/редактирование слотов, онлайн-оплату, отзывы, loyalty, no-show и auto-weather cancellation.

Сквозные логики, которые должны стать отдельными domain/application сервисами или pure-функциями:

- `LOGIC-001` OTP-авторизация и сессия.
- `LOGIC-002` доступность: `min(free_seats, route.capacity_cap, 3)` плюс проверка `free_rental_boards`.
- `LOGIC-003` цена: `price * seats_count + rental_price * rental_count`.
- `LOGIC-004` отмена: `>= 2h` ранняя, `< 2h` поздняя, после старта отмена недоступна.
- `LOGIC-005` фильтры слотов: OR внутри группы, AND между группами.
- `LOGIC-006` карта: Yandex Static/JS/API или платформенный адаптер, обязательный текстовый фолбэк.
- `LOGIC-007` push-разрешение после первой успешной брони.
- `LOGIC-008` Loading / Content / Empty / Error, Refreshing поверх контента, action submitting.

## Выводы по текущему BE

Реализованные клиентские endpoints:

| Домен | operationId | Метод |
|---|---|---|
| Auth | `requestAuthCode` | `POST /auth/request-code` |
| Auth | `verifyAuthCode` | `POST /auth/verify-code` |
| Auth | `logout` | `POST /auth/logout` |
| Profile | `getProfile` | `GET /profile` |
| Profile | `updateProfile` | `PATCH /profile` |
| Profile | `deleteAccount` | `DELETE /profile` |
| Profile | `requestPhoneChangeCode` | `POST /profile/phone/request-code` |
| Profile | `confirmPhoneChange` | `POST /profile/phone/confirm` |
| Slots | `listSlots` | `GET /slots` |
| Slots | `getSlot` | `GET /slots/{slotId}` |
| Instructors | `listInstructors` | `GET /instructors` |
| Bookings | `createBooking` | `POST /bookings` |
| Bookings | `listBookings` | `GET /bookings` |
| Bookings | `getBooking` | `GET /bookings/{bookingId}` |
| Bookings | `cancelBooking` | `POST /bookings/{bookingId}/cancel` |

BE важные особенности для клиента:

- Bearer token хранится на клиенте и стирается при любом `401`.
- `requestAuthCode` в dev-режиме возвращает `code` в ответе, хотя это не должно быть production UX-зависимостью.
- `verifyAuthCode` возвращает `token`, `client`, `is_new`; при `is_new=true` нужно выполнить `updateProfile`.
- `createBooking` принимает опциональный `Idempotency-Key` UUID. Клиент должен генерировать и удерживать ключ на время retry одного и того же payload.
- Ошибки приходят в формате `{ code, message, details? }`; для action errors UI показывает snackbar с `message`.
- `slot_full` может содержать `details.available_seats` и `details.available_rental_boards`; UI должен обновлять форму и подсказывать уменьшить места/прокат.
- В текущем OpenAPI и BE `Booking`/`BookingSummary` возвращают `price_total`. Это расходится с `LOGIC-003`, где сказано, что API его не возвращает. Клиентское решение: в форме `SCR-004` считать цену локально до отправки, для уже созданных броней предпочитать серверный `price_total`, а при его отсутствии считать fallback-формулой.
- OpenAPI `Route.geometry` есть, миграции содержат `routes.geometry`, но текущие BE-мапперы `catalog.go` и `bookings.go` не возвращают `geometry`. Для `LOGIC-006` клиент должен иметь фолбэк "пин + текст", а BE gap нужно закрыть отдельно, если линия трассы обязательна для MVP.

## Целевая структура проекта

Создать клиент под `client/`:

```text
client/
  settings.gradle.kts
  build.gradle.kts
  gradle/libs.versions.toml
  shared/
    build.gradle.kts
    src/commonMain/kotlin/com/volna/app/
    src/commonTest/kotlin/com/volna/app/
    src/androidMain/kotlin/com/volna/app/
    src/iosMain/kotlin/com/volna/app/
    src/wasmJsMain/kotlin/com/volna/app/
  androidApp/
  iosApp/
  webApp/
```

Рекомендуемый стек:

- Kotlin Multiplatform + Compose Multiplatform.
- Coroutines, Flow, kotlinx.serialization, kotlinx.datetime.
- Ktor Client for REST.
- Koin или простой manual DI. Для MVP предпочтителен Koin, если не требуется строгая compile-time DI.
- Decompose для навигации и lifecycle или Voyager только если нужен более лёгкий Compose-first подход. Для Android/iOS/Web и MVI лучше Decompose.
- Multiplatform Settings для preferences.
- Secure storage expect/actual: Android Keystore/EncryptedSharedPreferences, iOS Keychain, Web local/session storage с отдельной security note.
- Kermit/Napier для логов.
- moko-permissions или собственные expect/actual для push permission.
- Карты через expect/actual `MapRenderer`: Android/iOS native/web JS адаптеры или web-first Yandex adapter + fallback.

## Чистая архитектура

Зависимости направлены внутрь:

```text
presentation -> domain <- data
presentation -> application/usecase -> domain
data -> domain interfaces
platform adapters -> shared interfaces
```

Слои:

- `domain`: сущности, value objects, pure rules, repository interfaces.
- `application`: use cases, session orchestration, MVI reducers/effects where appropriate.
- `data`: Ktor API clients, DTO, mappers, repositories, pagination, idempotency persistence.
- `presentation`: screens, components, MVI stores, navigation, theme.
- `platform`: expect/actual storage, links, permissions, maps, clock.

Пакеты:

```text
com.volna.app.core
  config
  error
  network
  time
  storage
  mvi
  ui
  theme
  navigation
com.volna.app.auth
com.volna.app.profile
com.volna.app.catalog
com.volna.app.booking
com.volna.app.map
com.volna.app.notifications
```

## Domain model

Основные модели:

```kotlin
data class Client(
    val id: ClientId,
    val name: String?,
    val phone: Phone,
    val createdAt: Instant,
)

data class Slot(
    val id: SlotId,
    val startAt: Instant,
    val route: Route,
    val instructor: Instructor,
    val totalSeats: Int,
    val freeSeats: Int,
    val freeRentalBoards: Int,
    val price: MoneyRub,
    val rentalPrice: MoneyRub,
    val meetingPoint: MeetingPoint,
    val status: SlotStatus,
)

data class Booking(
    val id: BookingId,
    val slotId: SlotId,
    val clientId: ClientId?,
    val seatsCount: Int,
    val rentalCount: Int,
    val status: BookingStatus,
    val priceTotal: MoneyRub?,
    val createdAt: Instant,
    val cancelledAt: Instant?,
    val slot: Slot?,
)
```

Pure services:

- `AvailabilityPolicy`: `maxSeatsForBooking(slot)`, `canRentBoards(seats, rental, slot)`.
- `BookingPriceCalculator`: `price * seats + rentalPrice * rental`.
- `CancellationPolicy`: `early`, `late`, `unavailableAfterStart`.
- `SlotFilterPolicy`: query builder and applied filter count.
- `BookingGroupingPolicy`: upcoming/past derived from `slot.startAt`, not from stored status.

## MVI стандарт

Каждый экран получает:

- `State`: immutable UI state.
- `Intent`: пользовательские действия.
- `Effect`: одноразовые события навигации/snackbar/system dialog.
- `Store`: coroutine scope, reducer, use case calls.

Базовые типы:

```kotlin
sealed interface Loadable<out T> {
    data object Initial : Loadable<Nothing>
    data object Loading : Loadable<Nothing>
    data class Content<T>(val value: T, val refreshing: Boolean = false) : Loadable<T>
    data class Empty(val reason: EmptyReason) : Loadable<Nothing>
    data class Error(val error: UiError) : Loadable<Nothing>
}

enum class ActionStatus { Idle, Submitting }
```

Правила:

- Первичная загрузка: `Loading -> Content|Empty|Error`.
- Pull-to-refresh: контент сохраняется, `refreshing=true`; ошибка только snackbar.
- Action submit: отдельный `ActionStatus.Submitting`, повторный tap блокируется.
- 4xx с `message`: snackbar, а не full-screen error, если это action.
- `401`: централизованно очистить сессию и отправить `RootEffect.OpenAuth`.

## Навигация

Root graph:

```text
Root
  SplashSessionCheck
  AuthFlow
    PhoneStep
    OtpStep
    NameStep
  MainTabs
    SlotsStack
      SlotList
      SlotDetails
      BookingForm
    BookingsStack
      BookingList
      BookingDetails
    ProfileStack
      Profile
```

Bottom sheets:

- `FiltersSheet` over `SlotList`.
- `BookingSuccessSheet` after successful `createBooking`.
- `CancelConfirmSheet` over `BookingDetails`.
- `RouteMapSheet` over `SlotDetails` or `BookingDetails`.

На Web bottom sheet можно рендерить как адаптивный modal/drawer при широкой ширине, но сохранить тот же route и MVI контракт.

## Data layer и API

Репозитории:

- `AuthRepository`: `requestCode`, `verifyCode`, `logout`.
- `ProfileRepository`: `getProfile`, `updateName`, `deleteAccount`, `requestPhoneChangeCode`, `confirmPhoneChange`.
- `SlotRepository`: `listSlots`, `getSlot`.
- `InstructorRepository`: `listInstructors`.
- `BookingRepository`: `createBooking`, `listBookings`, `getBooking`, `cancelBooking`.
- `SessionRepository`: token read/write/clear, auth state flow.

Ktor setup:

- `ContentNegotiation` JSON with snake_case mapping.
- `HttpTimeout`.
- `DefaultRequest` base URL from build config.
- Auth plugin or custom interceptor injecting `Authorization: Bearer`.
- Response validator mapping API `Error` into typed `ApiFailure`.

Typed failures:

```kotlin
sealed interface AppFailure
data object Unauthorized : AppFailure
data class ApiFailure(val code: ApiErrorCode, val message: String, val details: ErrorDetails?) : AppFailure
data object NetworkUnavailable : AppFailure
data object Timeout : AppFailure
data object UnknownFailure : AppFailure
```

Idempotency:

- `CreateBookingUseCase` generates UUID `Idempotency-Key` per form submission.
- Same key is reused for retry of identical `slot_id`, `seats_count`, `rental_count`.
- If payload changes, old key is discarded.
- On `idempotency_conflict`, show server `message`, regenerate key only after user changes/retries intentionally.

## Feature design

### Auth: `SCR-001`, `LOGIC-001`

State:

- step: phone / otp / name.
- phone, code, name.
- ttlSeconds, resendAfterSeconds, countdown.
- actionStatus.

Use cases:

- `RequestAuthCodeUseCase`.
- `VerifyAuthCodeUseCase`.
- `CompleteNewProfileUseCase`.
- `ObserveSessionUseCase`.

Acceptance focus:

- E.164 validation before request.
- OTP 4-6 digits.
- timer survives minor recompositions and navigation back within auth flow.
- `is_new=false` skips name step.
- `401` after profile update clears session.

### Catalog: `SCR-002`, `BS-001`, `SCR-003`

State:

- `Loadable<PagedList<SlotSummaryUi>>`.
- applied filters and draft filters.
- instructor dictionary load state.
- active filter count.

Use cases:

- `LoadSlotsUseCase`.
- `RefreshSlotsUseCase`.
- `LoadInstructorsUseCase`.
- `BuildSlotQueryUseCase`.

Client rules:

- Default `only_available=false`, unless UI explicitly applies "только со свободными местами".
- Slots without places are shown disabled/marked if returned by API.
- Slot details reload with `getSlot` to avoid stale availability before booking.

### Booking form: `SCR-004`, `BS-002`

State:

- slot content.
- seatsCount 1..3 and not above available max.
- board choice per seat, derived rentalCount.
- computed total price.
- actionStatus.

Use cases:

- `CreateBookingUseCase`.
- `CalculateAvailabilityUseCase`.
- `CalculateBookingPriceUseCase`.
- `RequestPushPermissionAfterBookingUseCase`.

Client rules:

- Do not hardcode route caps or board count.
- Own board consumes group seat, not rental board.
- On `slot_full`, use `details` to update max and show contextual message.
- On success, update local slot availability opportunistically or invalidate slot/list caches.

### My bookings: `SCR-005`, `SCR-006`, `BS-003`

State:

- active/cancelled/late_cancel bookings from API.
- upcoming/past tabs are derived by `slot.start_at`.
- cancel availability derived by `now < slot.startAt`.
- cancel type preview by `CancellationPolicy`.

Use cases:

- `LoadBookingsUseCase`.
- `LoadBookingDetailsUseCase`.
- `CancelBookingUseCase`.

Client rules:

- Exactly 2 hours before start is early cancellation.
- After cancel, replace details with server response and invalidate bookings list/slots.
- Repeated cancel errors (`already_cancelled`) show snackbar and refresh details.

### Profile: `SCR-007`

State:

- `Loadable<ClientUi>`.
- edit name state.
- optional phone change sub-flow.
- delete account confirmation.

Use cases:

- `LoadProfileUseCase`.
- `UpdateProfileNameUseCase`.
- `LogoutUseCase`.
- `DeleteAccountUseCase`.
- `RequestPhoneChangeCodeUseCase`.
- `ConfirmPhoneChangeUseCase`.

MVP decision needed: screen spec highlights contact data and logout, while API already supports phone change and account delete. Implement at least profile view + logout + name edit; phone change/delete can be feature flags if scope needs tightening.

### Maps: `LOGIC-006`, `BS-004`

Shared contract:

```kotlin
interface MapLauncher {
    fun openYandexMaps(point: MeetingPoint, route: Route?)
}

@Composable
expect fun RouteMapPreview(
    route: Route,
    meetingPoint: MeetingPoint,
    state: MapUiState,
    onRetry: () -> Unit,
)
```

Rules:

- Map preview and interactive sheet must not call Апекс REST API.
- Inputs come from `getSlot` or `getBooking`.
- If `route.geometry == null`, show pin + text and treat as Content without line.
- If map SDK/API/key fails, show text fallback + "Открыть в Яндекс.Картах".
- `yandex_maps_api_key` must come from remote/build config, never hardcoded in code.

## Theme from Figma

Цель: Figma становится источником visual tokens, Compose theme - source of implementation truth.

Required extraction after Figma access:

- Color styles/variables: brand, background, surface, text, border, success, warning, error, info, overlay.
- Typography: font family, sizes, line heights, weights for display/title/body/label.
- Spacing scale: 4/8-based or actual Figma variable collection.
- Radius scale: cards max 8px unless Figma system explicitly differs.
- Elevation/effects.
- Component anatomy: buttons, text fields, chips, cards, bottom sheets, tabs, snackbars, skeletons.
- Light/dark modes if defined. If Figma has only one mode, implement light first and keep dark tokens explicit but mapped conservatively.

Compose implementation:

```text
core/theme/
  VolnaTheme.kt
  VolnaColors.kt
  VolnaTypography.kt
  VolnaSpacing.kt
  VolnaShapes.kt
  VolnaElevation.kt
  VolnaComponents.kt
```

Token mapping:

| Figma | Compose |
|---|---|
| Color variables | `VolnaColorScheme` |
| Text styles | `VolnaTypography` |
| Number variables for spacing/radius | `VolnaSpacing`, `VolnaShapes` |
| Effect styles | `VolnaElevation` |
| Components/variants | reusable Compose components |

Theme acceptance:

- No hardcoded colors in feature screens.
- No hardcoded typography except inside theme.
- Components read tokens from `VolnaTheme`.
- Contrast for primary text/buttons meets WCAG AA and `NFR-1`.
- All screen states from `LOGIC-008` have tokenized skeleton, empty, error and snackbar visuals.
- Android/iOS/Web render the same token values, with platform-specific font fallback documented.

Figma gap:

- Точные значения не зафиксированы в аналитике и не извлечены в этой сессии. Перед UI implementation нужно получить Figma variables/styles через Figma API/plugin export или вручную экспортировать token table.

## Platform specifics

Android:

- `androidApp` hosts Compose activity.
- Secure token in EncryptedSharedPreferences/Keystore.
- Push permission for Android 13+.
- Deep links to Yandex Maps via intent/browser fallback.

iOS:

- `iosApp` SwiftUI/Compose host.
- Secure token in Keychain.
- Push permission through UserNotifications.
- External map handoff through URL schemes/universal links.

Web:

- Compose Web/Wasm target.
- Token storage must be documented as lower-trust than native secure storage.
- Yandex Maps JS integration can be strongest on Web; native platforms may use static preview + handoff first if SDK scope is high.

## Testing strategy

Unit tests in `commonTest`:

- `LOGIC-002` availability boundaries.
- `LOGIC-003` price calculation.
- `LOGIC-004` cancellation boundary: `2h+1s`, `2h`, `1h59m59s`, after start.
- slot filter query builder.
- booking grouping upcoming/past.
- MVI reducers for every screen.
- API error mapping.

Data tests:

- Ktor MockEngine success/error for every operationId.
- `401` clears session.
- create booking idempotency key reuse/discard behavior.
- snake_case DTO serialization.

UI tests:

- Screenshot/Golden tests for theme components if project setup allows.
- StateContainer Loading/Empty/Error/Refreshing.
- Auth flow, booking form validation, cancel confirmation.

E2E/manual smoke:

- Login new user -> name -> slot list -> slot details -> booking -> success -> my bookings -> cancel -> profile logout.
- Repeat booking network retry with same idempotency key.
- Map fallback when geometry/key is missing.

## Implementation roadmap

- [ ] `CMP-00` Create `client/` Gradle KMP skeleton with Android, iOS, Web targets.
- [ ] `CMP-01` Add shared dependency catalog, CI commands, formatting and test tasks.
- [ ] `CMP-02` Implement core architecture: MVI base, DI, config, clock, error model, logging.
- [ ] `CMP-03` Implement theme skeleton and Figma token import format.
- [ ] `CMP-04` Extract Figma tokens and finalize `VolnaTheme`.
- [ ] `CMP-05` Implement network layer and generated/manual DTOs aligned with OpenAPI.
- [ ] `CMP-06` Implement secure/session storage expect/actual.
- [ ] `CMP-07` Implement Auth flow `SCR-001`.
- [ ] `CMP-08` Implement main navigation and tabs.
- [ ] `CMP-09` Implement slots list, filters and instructors dictionary `SCR-002`/`BS-001`.
- [ ] `CMP-10` Implement slot card and map preview fallback `SCR-003`.
- [ ] `CMP-11` Implement booking form and success sheet `SCR-004`/`BS-002`.
- [ ] `CMP-12` Implement bookings list/details/cancel `SCR-005`/`SCR-006`/`BS-003`.
- [ ] `CMP-13` Implement route map sheet `BS-004`.
- [ ] `CMP-14` Implement profile view, edit name, logout, optional phone change/delete `SCR-007`.
- [ ] `CMP-15` Implement push permission adapter `LOGIC-007`.
- [ ] `CMP-16` Add unit/data/UI tests for core flows.
- [ ] `CMP-17` Run integration smoke against local BE.
- [ ] `CMP-18` Polish accessibility, responsive Web layout and visual parity with Figma.

## Open questions and gaps

1. Figma tokens are not yet extracted. Need access/tooling before final UI implementation.
2. BE currently does not map `Route.geometry` into slot/booking responses, despite OpenAPI and migrations. Needed for route polyline.
3. `LOGIC-003` says `price_total` is not returned, while OpenAPI/BE returns it. Recommend updating analysis docs or explicitly accepting API as source for existing bookings.
4. Push token registration REST endpoint is not in current API. `LOGIC-007` can request OS permission, but delivery registration needs a future endpoint or external infrastructure contract.
5. Production OTP must not expose `code` in `RequestCodeResponse`; current BE dev behavior should be environment-gated and ignored by normal UI.
6. Web secure storage for Bearer token needs product/security decision: localStorage/sessionStorage vs BFF/cookie architecture.
