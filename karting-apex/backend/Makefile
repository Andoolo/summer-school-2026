.PHONY: fmt generate check-generated lint lint-api test run migrate k6-seed k6-smoke k6-booking-300 k6-cancel-300

fmt:
	gofmt -w ./cmd ./internal

generate:
	go generate ./internal/http/openapi

check-generated: generate
	git diff --exit-code -- internal/http/openapi

lint:
	go vet ./...

lint-api:
	npm --prefix ../01-analysis/api run lint

test:
	go test ./...

run:
	go run ./cmd/api

migrate:
	go run github.com/pressly/goose/v3/cmd/goose@v3.24.1 -dir migrations postgres "$${DATABASE_URL:-postgres://volna:volna@localhost:5432/volna?sslmode=disable}" up

k6-seed:
	go run ./cmd/k6seed

k6-smoke:
	k6 run k6/smoke.js

k6-booking-300:
	k6 run k6/booking_300_vu.js

k6-cancel-300:
	k6 run k6/cancel_300_vu.js
