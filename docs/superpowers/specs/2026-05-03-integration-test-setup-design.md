# Integration Test Setup for httpSMS API

## Problem

The httpSMS API has no integration tests that verify the full SMS send/receive flow end-to-end. We need a CI-gated integration test that runs the entire stack in Docker and validates the core message lifecycle before deploying the API.

## Approach

Run the full application stack (API + PostgreSQL + Redis) in Docker alongside an **emulator** service that acts as a fake Android phone. The emulator implements a fake FCM server endpoint so the API's Firebase messaging client sends push notifications to it (instead of Google). The emulator then responds with SENT/DELIVERED events, completing the SMS lifecycle. A Go test runner exercises the API externally and asserts on final message state.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Docker Compose (tests/docker-compose.yml)              │
│                                                         │
│  ┌──────────┐  ┌───────┐  ┌──────────────────────────┐ │
│  │PostgreSQL│  │ Redis │  │  API (existing Dockerfile)│ │
│  └──────────┘  └───────┘  └────────────┬─────────────┘ │
│                                         │ FCM push      │
│                                         ▼               │
│                            ┌──────────────────────────┐ │
│                            │  Emulator (fake phone)   │ │
│                            │  - Fake FCM server :9090 │ │
│                            │  - Fires SENT/DELIVERED  │ │
│                            │    events back to API    │ │
│                            └──────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
         ▲
         │ HTTP calls (send SMS, get message, etc.)
         │
┌────────┴──────────┐
│  Test Runner (Go) │  ← runs on host / in CI
│  go test ./...    │
└───────────────────┘
```

## Components

### 1. `tests/docker-compose.yml`

Brings up the full stack:

- **postgres** — Same as root `docker-compose.yml`, seeded with `tests/seed.sql`
- **redis** — Standard Redis
- **api** — Built from `api/Dockerfile`, configured with `FCM_ENDPOINT=http://emulator:9090` to redirect Firebase messaging to the emulator
- **emulator** — Built from `tests/emulator/Dockerfile`, receives FCM pushes and fires events back

### 2. `tests/emulator/` (Go project)

A lightweight Go HTTP server that:

- Exposes `POST /v1/projects/{project}/messages:send` — mimics the FCM v1 API. Receives push notification payloads from the API's Firebase messaging client.
- Exposes `POST /token` — returns a fake OAuth2 access token (the Firebase SDK calls this before sending FCM). Response format: `{"access_token": "fake-token", "token_type": "Bearer", "expires_in": 3600}`
- Exposes `GET /health` — health check endpoint
- On receiving a push with `KEY_MESSAGE_ID` in the data payload:
  1. Calls `GET http://api:8000/v1/messages/outstanding?message_id={messageID}` (using phone API key) to fetch the message like a real phone would
  2. Waits a brief delay (e.g., 200ms)
  3. Calls `POST http://api:8000/v1/messages/{messageID}/events` with event `SENT` (using phone API key)
  4. Waits another brief delay (e.g., 200ms)
  5. Calls `POST http://api:8000/v1/messages/{messageID}/events` with event `DELIVERED` (using phone API key)
- All API calls authenticated with the seeded phone API key (`x-api-key` header)
- Asserts it received the correct FCM payload structure (path, data.KEY_MESSAGE_ID present)

### 3. `tests/seed.sql`

SQL script that runs on PostgreSQL startup to create:

- A test user: `id='test-user-id'`, `email='test@httpsms.com'`, `api_key='test-user-api-key'`, `subscription_name='pro'`
- A system user (for event queue): `id='system-user-id'`, `api_key='system-user-api-key'`
- A phone: `id=<uuid>`, `user_id='test-user-id'`, `phone_number='+18005550199'`, `fcm_token='fake-fcm-token'`
- A phone API key: `id=<uuid>`, `user_id='test-user-id'`, `api_key='test-phone-api-key'`, `phone_numbers=['+18005550199']`

### 4. API Modification — FCM Transport Override

In `api/pkg/di/container.go`, modify `FirebaseMessagingClient()`:

- When `FCM_ENDPOINT` env var is set, create the Firebase App with a custom HTTP client whose `Transport` rewrites request URLs from `https://fcm.googleapis.com` to the value of `FCM_ENDPOINT`
- This requires no changes to business logic — the messaging client works normally but routes traffic to the emulator
- The Firebase credentials must be a syntactically valid fake service account JSON with `token_uri` pointing to `http://emulator:9090/token`

### 4b. `tests/.env.test` — API environment for tests

```env
ENV=production
GCP_PROJECT_ID=httpsms-test
EVENTS_QUEUE_TYPE=emulator
EVENTS_QUEUE_NAME=events-local
EVENTS_QUEUE_ENDPOINT=http://localhost:8000/v1/events
EVENTS_QUEUE_USER_API_KEY=system-user-api-key
EVENTS_QUEUE_USER_ID=system-user-id
FCM_ENDPOINT=http://emulator:9090
DATABASE_URL=postgresql://dbusername:dbpassword@postgres:5432/httpsms
DATABASE_URL_DEDICATED=postgresql://dbusername:dbpassword@postgres:5432/httpsms
REDIS_URL=redis://@redis:6379
APP_PORT=8000
ENTITLEMENT_ENABLED=false
USE_HTTP_LOGGER=true
FIREBASE_CREDENTIALS=<fake service account JSON with token_uri=http://emulator:9090/token>
```

### 5. `tests/integration_test.go` (Go test files)

Go tests using the standard `testing` package + `testify` for assertions:

**Test 1: Send SMS E2E**

1. `POST /v1/messages/send` with `from=<test_phone>`, `to=+18005550100`, `content="Hello"` (using user API key `x-api-key` header)
2. Extract message ID from response
3. Poll `GET /v1/messages/{id}` every 200ms with max 15s timeout (using user API key)
4. Assert message status reaches `delivered`
5. Assert message events include both `SENT` and `DELIVERED`

**Test 2: Receive SMS**

1. `POST /v1/messages/receive` (using phone API key auth) with `from=+18005550100`, `to=+18005550199`, `content="Hi there"`, `sim="SIM1"`, `timestamp=<now>`
2. Extract message ID from response
3. `GET /v1/messages/{id}` (using user API key auth)
4. Assert message exists with correct content, from, to fields
5. Assert status is `received`

### 6. `.github/workflows/integration-test.yml`

GitHub Actions workflow:

```yaml
name: integration-test
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - Checkout
      - Docker Compose up (tests/docker-compose.yml)
      - Wait for health checks (API + emulator)
      - Run: cd tests && go test -v -timeout 120s ./...
      - Docker Compose down

  deploy-api:
    needs: integration-test
    # existing deploy logic
```

The `deploy-api` job depends on `integration-test` passing.

## FCM Redirect Implementation Detail

The Firebase Admin Go SDK's messaging client sends HTTP POST requests to:

```
https://fcm.googleapis.com/v1/projects/{project_id}/messages:send
```

We intercept this by providing a custom `http.RoundTripper`:

```go
type fcmRedirectTransport struct {
    target string // e.g., "http://emulator:9090"
    base   http.RoundTripper
}

func (t *fcmRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    // Rewrite: https://fcm.googleapis.com/... → http://emulator:9090/...
    req.URL.Scheme = "http"
    req.URL.Host = strings.TrimPrefix(t.target, "http://")
    return t.base.RoundTrip(req)
}
```

This is injected via `option.WithHTTPClient()` when creating the Firebase App in the DI container.

## Fake Firebase Credentials

For the integration test environment, we provide a minimal fake service account JSON:

```json
{
  "type": "service_account",
  "project_id": "httpsms-test",
  "private_key_id": "test",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n<test key>\n-----END RSA PRIVATE KEY-----\n",
  "client_email": "test@httpsms-test.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "http://emulator:9090/token",
  "auth_provider_x509_cert_url": "http://emulator:9090/certs",
  "client_x509_cert_url": "http://emulator:9090/certs/test"
}
```

The emulator implements:

- `POST /token` — Accepts JWT assertion grant, returns `{"access_token": "fake-token", "token_type": "Bearer", "expires_in": 3600}`
- Does NOT validate the JWT signature — just returns a valid token response

## Docker Health Checks & Orchestration

Services start in order with health dependencies:

1. **postgres** — healthy when `pg_isready` passes
2. **redis** — healthy when accepting connections
3. **emulator** — healthy when `GET /health` returns 200
4. **api** — starts after postgres+redis+emulator healthy, healthy when `GET /v1/` returns (or a dedicated health endpoint)

Test runner waits for all services healthy before executing `go test`.

## File Structure

```
tests/
├── docker-compose.yml
├── seed.sql
├── go.mod
├── go.sum
├── integration_test.go
├── helpers_test.go          # shared HTTP client, polling helpers
├── .env.test                # env vars for the API in test mode
└── emulator/
    ├── Dockerfile
    ├── go.mod
    ├── go.sum
    ├── main.go              # entry point, starts HTTP server
    ├── fcm_handler.go       # fake FCM endpoint
    ├── token_handler.go     # fake OAuth2 token endpoint
    └── events.go            # fires SENT/DELIVERED events to API
```

## Key Design Decisions

1. **DB seeding over Firebase Auth emulator** — Simpler, keeps focus on SMS flow testing. Auth is not what we're validating.
2. **Real FCM code path with redirected transport** — Tests the actual Firebase SDK integration, payload construction, and error handling. More confidence than a noop mock.
3. **Emulator as separate Go project** — Clean separation, own Dockerfile, own module. Doesn't pollute the API codebase.
4. **Test runner runs on host (not in Docker)** — Simpler debugging, standard `go test` output, easier CI integration.
5. **Polling with timeout for async assertions** — The send flow is async (event-driven). Polling with backoff is the pragmatic approach.

## Out of Scope

- Testing the web frontend
- Testing the Android app
- Load/performance testing
- Testing auth flows (login, registration)
- Testing billing/entitlements
- MMS/attachment testing (can be added later)
