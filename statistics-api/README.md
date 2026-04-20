I've added new collumn into db.
So, in order to avoid any errors you should migrate or rebuild your database:

1. Migration:
```
docker compose exec postgres psql -U postgres -d core -c "ALTER TABLE tracks ADD COLUMN IF NOT EXISTS total_plays BIGINT NOT NULL DEFAULT 0 CHECK (total_plays >= 0);"
```
Checking (total_plays collumn should occur):
```
docker compose exec postgres psql -U postgres -d core -c "\d tracks"
```
2. Dropping and rebuild (in case if migration will not work properly, but all data will be deleted):
```
docker compose down -v
docker compose up -d --build
```
