# Integration Tests

End-to-end integration tests for the httpSMS API. These tests validate the complete SMS lifecycle by running the full application stack in Docker alongside a phone emulator service.

## Architecture

```
┌──────────────┐     HTTP      ┌──────────────┐
│  Test Runner │─────────────▶│   API (Go)   │
│   (Go test)  │              │  Port 8000   │
└──────────────┘              └──────┬───────┘
                                     │
                          FCM Push   │   Events
                          (HTTP)     │   (HTTP)
                                     ▼
                              ┌──────────────┐
                              │   Emulator   │
                              │  (Fiber v3)  │
                              │  Port 9090   │
                              └──────────────┘
                                     │
                              ┌──────┴───────┐
                              │  PostgreSQL  │   │    Redis    │
                              │  Port 5435   │   │  Port 6379  │
                              └──────────────┘   └─────────────┘
```

### Components

| Component       | Description                                             |
| --------------- | ------------------------------------------------------- |
| **API**         | The httpSMS Go API server running in Docker             |
| **Emulator**    | A Fiber v3 Go service that simulates an Android phone   |
| **PostgreSQL**  | Database for the API                                    |
| **Redis**       | Cache and queue backend                                 |
| **Seed**        | One-shot container that seeds test data into PostgreSQL |
| **Test Runner** | Go test binary that runs on the host machine            |

### How It Works

1. **Send SMS flow**: Test sends `POST /v1/messages/send` → API pushes FCM notification to emulator → Emulator calls `GET /v1/messages/outstanding` → Emulator fires `SENT` and `DELIVERED` events → Test polls `GET /v1/messages/{id}` until status is `delivered`

2. **Receive SMS flow**: Test sends `POST /v1/messages/receive` (as the phone) → API stores message → Test verifies via `GET /v1/messages/{id}`

### FCM Redirect

The API's Firebase SDK is configured (via `FCM_ENDPOINT` env var) to redirect all FCM HTTP requests to the emulator instead of Google's servers. The emulator serves:

- `/token` — Fake OAuth2 token endpoint (Firebase SDK requests tokens before sending)
- `/v1/projects/:project/messages:send` — Fake FCM push endpoint

## Test Coverage

- [x] **Send SMS E2E** — Full send lifecycle: API → FCM push → emulator responds with SENT/DELIVERED events → message reaches `delivered` status
- [x] **Receive SMS E2E** — Phone submits received message to API → message is stored and retrievable via GET endpoint

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Docker Compose
- [Go 1.22+](https://go.dev/dl/)
- [jq](https://jqlang.github.io/jq/download/) (for Firebase credentials generation)
- [OpenSSL](https://www.openssl.org/) (for RSA key generation)

## Running Locally

### 1. Generate Firebase Credentials

The integration tests use a fake Firebase service account. Generate it with:

```bash
cd tests
bash generate-firebase-credentials.sh
```

This creates `firebase-credentials.json` with a throwaway RSA key (the emulator doesn't validate tokens).

### 2. Set Environment Variable

```bash
export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json)
```

### 3. Start the Stack

```bash
docker compose up -d --build --wait
```

This starts PostgreSQL, Redis, the API, and the emulator. The `--wait` flag blocks until all health checks pass.

### 4. Wait for Seeding

```bash
docker compose wait seed
sleep 2
```

The seed container inserts test users, phones, and API keys into PostgreSQL after the API has run its GORM migrations.

### 5. Run Tests

```bash
go test -v -timeout 120s ./...
```

### 6. Tear Down

```bash
docker compose down -v
```

The `-v` flag removes volumes (database data) for a clean slate next run.

### One-Liner

```bash
cd tests && \
  bash generate-firebase-credentials.sh && \
  export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json) && \
  docker compose up -d --build --wait && \
  docker compose wait seed && \
  sleep 2 && \
  go test -v -timeout 120s ./... ; \
  docker compose down -v
```

## CI/CD

Integration tests run automatically via GitHub Actions (`.github/workflows/integration-test.yml`):

- **Trigger**: Push to `main` or pull request targeting `main`
- **Flow**: Generates credentials → Starts Docker stack → Seeds DB → Runs tests → Collects logs on failure → Tears down
- **Gate**: Deployment should only proceed if integration tests pass

## Test Data

| Entity         | Value                                  |
| -------------- | -------------------------------------- |
| User API Key   | `test-user-api-key`                    |
| Phone API Key  | `pk_test-phone-api-key`                |
| Phone Number   | `+18005550199`                         |
| Contact Number | `+18005550100`                         |
| User ID        | `test-user-id`                         |
| Phone ID       | `a1b2c3d4-e5f6-7890-abcd-ef1234567890` |

See [`seed.sql`](./seed.sql) for the complete seed data.

## Project Structure

```
tests/
├── docker-compose.yml       # Full stack orchestration
├── seed.sql                 # Database seed data
├── .env.test                # API environment variables
├── generate-firebase-credentials.sh  # Generates fake Firebase credentials
├── go.mod                   # Test runner Go module
├── go.sum
├── helpers_test.go          # Test utilities (HTTP client, polling)
├── integration_test.go      # E2E test cases
└── emulator/                # Phone emulator service
    ├── Dockerfile
    ├── go.mod
    ├── go.sum
    ├── main.go              # Fiber v3 entry point
    ├── emulator.go          # Emulator struct and config
    ├── token_handler.go     # Fake OAuth2 token endpoint
    ├── fcm_handler.go       # Fake FCM push receiver
    └── events.go            # Event firing logic (SENT/DELIVERED)
```

## Troubleshooting

### API fails to start

Check the API logs:

```bash
docker compose logs api
```

Common issues:

- `FIREBASE_CREDENTIALS` env var not set or malformed
- PostgreSQL not ready (increase `start_period` in healthcheck)

### Tests timeout waiting for `delivered` status

Check the emulator logs:

```bash
docker compose logs emulator
```

The emulator should show:

1. `[FCM]` — Receiving the push notification
2. `[EVENTS]` — Fetching outstanding messages and firing events

If no `[FCM]` entries appear, the API isn't reaching the emulator (check `FCM_ENDPOINT` in `.env.test`).

### Seed container fails

```bash
docker compose logs seed
```

If you see "relation does not exist" errors, the API hasn't finished GORM migrations yet. Increase the API's `start_period` in `docker-compose.yml`.

## Adding New Tests

1. Add test functions to `integration_test.go` (or create new `*_test.go` files)
2. Use `doRequest()` helper for authenticated HTTP calls
3. Use `pollMessageStatus()` to wait for async state changes
4. Update the test coverage checklist in this README
