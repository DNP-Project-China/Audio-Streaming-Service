SHELL := /bin/bash

.PHONY: migrate dev test sys-up sys-down

migrate:
	docker run --rm -v "$(PWD)/core-api:/src" -w /src/sqlc sqlc/sqlc generate

dev:
	trap 'exit 0' INT TERM; set -a && source .env && set +a && POSTGRES_HOST=localhost go run ./core-api/cmd/

test:
	set -a && source .env && set +a && POSTGRES_HOST=localhost go test ./...

sys-up:
	docker compose up -d postgres kafka

sys-down:
	docker compose down -v

