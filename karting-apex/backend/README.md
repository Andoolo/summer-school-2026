# Backend API

Go REST API for the Volna SUP club client app. The implementation follows the OpenAPI contracts from `../01-analysis/api` and keeps client-visible resources in MVP scope only.

## Requirements

- Go 1.23+
- PostgreSQL 16
- Docker Compose, optional but recommended for local DB/API
- k6 2.x for performance scenarios
- npm dependencies for OpenAPI lint/bundle: `npm --prefix ../01-analysis/api install`

## Environment

Copy `.env.example` to `.env` if you want shell-local defaults. Do not commit `.env`.

Important variables:

- `HTTP_ADDR`: API listen address, default `:8080`.
- `DATABASE_URL`: PostgreSQL connection string for app, migrations and k6 seed.
- `TEST_DATABASE_URL`: PostgreSQL connection string for integration tests.
- `BASE_URL`: API URL for k6, default `http://127.0.0.1:8080`.

## Local Run

Start PostgreSQL:

```bash
docker compose --profile db up -d db
```

Apply migrations and seed read-only catalog data:

```bash
make migrate
```

Run API locally:

```bash
make run
```

The API exposes operational endpoints outside the public client contract:

- `GET /healthz`
- `GET /readyz`

## Docker Compose

Run API and PostgreSQL:

```bash
docker compose --profile app up --build
```

Run migrations through Compose:

```bash
docker compose --profile migrations run --rm migrate
```

Run k6 smoke through Compose:

```bash
docker compose --profile k6 run --rm k6
```

## OpenAPI

Generate transport DTOs/server interfaces:

```bash
make generate
```

Check OpenAPI contracts:

```bash
make lint-api
```

Generated files under `internal/http/openapi/*/*.gen.go` are not edited manually.

## Tests

Unit and package tests:

```bash
make test
```

Static checks:

```bash
make lint
```

Race detector:

```bash
go test -race ./...
```

PostgreSQL integration tests:

```bash
TEST_DATABASE_URL=postgres://volna:volna@localhost:5432/volna?sslmode=disable go test ./internal/http/handlers ./internal/storage/postgres -count=1
```

Integration tests create isolated schemas and skip automatically when `TEST_DATABASE_URL` is not set.

## k6

Prepare deterministic test users/sessions and reset test slots:

```bash
make k6-seed
```

Smoke test:

```bash
BASE_URL=http://127.0.0.1:8080 make k6-smoke
```

300 VU booking scenario:

```bash
BASE_URL=http://127.0.0.1:8080 VUS=300 DURATION=1m make k6-booking-300
```

300 VU cancel scenario:

```bash
make k6-seed
BASE_URL=http://127.0.0.1:8080 VUS=300 DURATION=1m TOKEN=vu-token-1 BOOKING_IDS=99999999-9999-9999-9999-999999999999 make k6-cancel-300
```

`routes`, `instructors`, and `slots` are read-only for client API. Development data is inserted by migrations and reset for load testing by `cmd/k6seed`.
