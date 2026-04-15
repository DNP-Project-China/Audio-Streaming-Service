.PHONY: migrate

migrate:
	docker run --rm -v "$(PWD)/core-api:/src" -w /src/sqlc sqlc/sqlc generate

