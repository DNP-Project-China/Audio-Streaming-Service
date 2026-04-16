SHELL := /bin/bash

.PHONY: migrate dev db-up db-down

migrate:
	docker run --rm -v "$(PWD)/core-api:/src" -w /src/sqlc sqlc/sqlc generate

dev:
	trap 'exit 0' INT TERM; set -a && source .env && set +a && POSTGRES_HOST=localhost go run ./core-api/cmd/

db-up:
	docker compose up -d postgres

db-down:
	docker compose down -v

