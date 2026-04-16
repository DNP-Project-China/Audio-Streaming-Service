# Core API

Service for:
- uploading audio files
- listing tracks for UI
- returning temporary URL for original file download

OpenAPI contract: `core-api/openapi.yaml`

## Requirements

You need:
- Docker + Docker Compose
- S3-compatible object storage bucket

You need a populated `.env` file in repository root.

Use this flow:
- copy `.env.example` to `.env`
- change only team-specific values (listed below)

## Required Configuration

Request those variables and put them in `.env`. Otherwise, the core-api will not work:

```env
S3_ENDPOINT=replace_me
S3_BUCKET=replace_me
S3_ACCESS_KEY=replace_me
S3_SECRET_KEY=replace_me
S3_PUBLIC_BASE_URL=https://replace_me
```

## Run Locally

1. Start the full system (API + DB + Kafka):

```bash
docker compose up -d --build
```

2. Verify API is available on localhost:

```bash
curl -s http://localhost:8000/health
```

3. Run all tests with current `.env`:

```bash
make test
```

4. Stop all services:

```bash
docker compose down -v
```

## Troubleshooting

Use different connection values depending on how you run the API.

When running locally with `make sys-up` and `make dev`:

```env
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
KAFKA_BROKERS=localhost:9094
```

When running with Docker Compose (`docker compose up -d --build`):

```env
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
KAFKA_BROKERS=kafka:9092
```

`.env.example` is preset for Docker Compose values.

## API Usage

### Health

```bash
curl -s http://localhost:8000/health
```

### Upload Track

```bash
curl -X POST http://localhost:8000/upload \
  -F "artist=Eminem" \
  -F "title=Mock Song" \
  -F "file=@core-api/server/handlers/testdata/testfile.mp3"
```

Response includes `track_id` in `pending` status.

### List Tracks

Default returns only `ready` tracks:

```bash
curl -s http://localhost:8000/tracks
```

With status filter:

```bash
curl -s "http://localhost:8000/tracks?status=pending"
```

### Download Original File URL

```bash
curl -s http://localhost:8000/download/<track_id>
```

Returns a temporary `download_url` to the original object in S3.
