SHELL := /bin/bash

.PHONY: migrate dev db-up db-down kafka-up kafka-down

migrate:
	docker run --rm -v "$(PWD)/core-api:/src" -w /src/sqlc sqlc/sqlc generate

dev:
	trap 'exit 0' INT TERM; set -a && source .env && set +a && POSTGRES_HOST=localhost go run ./core-api/cmd/

db-up:
	docker compose up -d postgres kafka

db-down:
	docker compose down -v

kafka-up:
	docker compose up -d kafka

kafka-down:
	docker compose stop kafka

