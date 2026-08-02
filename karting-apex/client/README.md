# Volna CMP Client

Compose Multiplatform client for Android, iOS and Web.

## Modules

- `shared` — common UI, domain models, MVI primitives, feature contracts and platform adapters.
- `androidApp` — Android host application.
- `webApp` — Web/Wasm host application.
- `iosApp` — placeholder for the native iOS host; `shared` produces the `VolnaShared` framework.

## Commands

Run from `client/` after adding a Gradle wrapper or installing Gradle:

```bash
gradle :shared:allTests
gradle :androidApp:assembleDebug
gradle :webApp:wasmJsBrowserDevelopmentRun
```

The current repository does not include `gradlew`; CI/local setup should add it before relying on these commands.

## Architecture

The shared module follows clean architecture:

- `domain` contains entities and pure policies for availability, price and cancellation.
- `core` contains MVI, error, time, theme and reusable UI primitives.
- feature packages define repository contracts and screen-facing models.
- platform source sets provide `expect/actual` adapters such as route map rendering.

Theme values are intentionally centralized under `core/theme`; replace placeholder tokens with exported Figma variables before visual polish.
