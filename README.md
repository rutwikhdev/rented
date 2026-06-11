# Rented

Property management platform — backend (Go + Echo) + frontend (React + Vite).

## Prerequisites

- Go 1.26+
- Node.js 22+
- Docker & docker-compose (for Postgres + Valkey/Redis)

## Quick start

```bash
# start Postgres and Valkey
docker compose up -d

# run the server
go run ./cmd/server/

# (in another terminal) run the frontend
cd frontend && npm install && npm run dev
```

Server starts on `:8080`, frontend on `:5173`.

## Project structure

```
├── cmd/server/main.go       # entrypoint — routes, middleware, startup
├── internal/
│   ├── config/              # toml-based config
│   ├── db/                  # sqlc-generated query layer
│   ├── handler/             # HTTP handlers
│   ├── middleware/           # auth middleware
│   ├── redis/               # redis client setup
│   ├── rules/               # reservation validation rules
│   └── utils/               # logger, validators, time helpers
├── frontend/                # React + Vite SPA
├── queries.sql              # sqlc source queries
├── schema.sql               # DDL (auto-applied on server start)
├── sqlc.yaml                # sqlc configuration
└── config.toml              # server / db / redis config
```

## API

All routes except `/health` are under `/api/v1`.

### Public

| Method | Path         | Description        |
|--------|--------------|--------------------|
| GET    | /health      | Health check       |
| POST   | /api/v1/signup | Create account   |
| POST   | /api/v1/login  | Sign in          |

### Authenticated (requires `Authorization: Bearer <token>`)

| Method | Path                  | Description              |
|--------|-----------------------|--------------------------|
| POST   | /api/v1/logout        | Sign out                 |
| POST   | /api/v1/property      | List properties (paginated) |
| POST   | /api/v1/property/new  | Create a property        |
| POST   | /api/v1/reservation   | List reservations (paginated, filterable) |
| POST   | /api/v1/reservation/new | Create a reservation   |

Check `./queries.txt` for a full walkthrough with curl examples.

### Pagination

Both list endpoints accept a JSON body with a `page` field:

```json
{"page": 1}
```

Optional filters for reservations:

```json
{
  "page": 1,
  "property_name": "Beach",
  "guest_name": "Charlie",
  "check_in_from": "2026-06-01T00:00:00Z",
  "check_out_to": "2026-12-31T23:59:59Z"
}
```

### Create reservation

```json
{
  "property_id": "<uuid>",
  "guest_name": "Charlie",
  "checkin": "2026-07-01T11:00",
  "checkout": "2026-07-05T10:00",
  "timezone": "America/New_York"
}
```

All responses use JSON with an `error` field on failures (4xx / 5xx).

## Testing

```bash
# backend
go test ./...

# frontend
cd frontend && npm run build
```

## Configuration

Edit `config.toml`:

```toml
[server]
port = 8080

[database]
url = "postgres://postgres:postgres@localhost:5432/rented?sslmode=disable"

[redis]
url = "redis://localhost:6379/0"
```
