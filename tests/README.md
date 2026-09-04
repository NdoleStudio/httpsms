# Integration Tests

End-to-end tests for the httpSMS API. The suite runs the API and its data
stores in Docker, keeps the existing Firebase/WireMock coverage, and adds a
standard-library-only HTTPS adapter emulator for URL-backed phone gateways.

## Architecture

```text
                              ┌──────────────────┐
                              │    API (Go)      │
                              │    Port 8000     │
                              └───────┬──────────┘
                                      │
                FCM HTTP              │ HTTPS callbacks
           ┌──────────────────┐       │       ┌──────────────────────┐
           │ WireMock         │◀──────┘       │ Adapter emulator     │
           │ Port 8080        │               │ HTTPS callback :9091│
           └──────────────────┘               │ HTTP control   :9092│
                                              └──────────┬───────────┘
                                                         │ phone API calls
                                                         └──────────▶ API

┌──────────────────┐        HTTP        ┌────────────────────────────┐
│ Test runner      │───────────────────▶│ API, WireMock, and adapter │
│ Go test on host  │                    │ control endpoints          │
└──────────────────┘                    └────────────────────────────┘

Data stores: CockroachDB, Redis, and MongoDB.
```

### Components

| Component | Description |
| --- | --- |
| **API** | The httpSMS Go API server |
| **WireMock** | Existing fake Firebase and webhook endpoints |
| **Adapter emulator** | HTTPS URL-backed phone gateway with an HTTP-only host control API |
| **CockroachDB** | Relational database for API data |
| **Redis** | Cache and local event queue backend |
| **MongoDB** | Heartbeat and contact backend |
| **Seed** | One-shot container that inserts integration users and API keys |
| **Test runner** | Go tests running on the host |

### Gateway Flows

1. **Existing FCM flow:** the API sends Firebase-compatible requests to
   WireMock. Existing tests fetch outstanding messages and submit phone events
   without changing their transport.
2. **URL-backed outgoing flow:** the API posts an FCM-compatible envelope to
   `https://adapter-emulator:9091/notifications/{gatewayID}`. The adapter uses
   its registered phone API key to fetch the outstanding message and post
   `SENT` followed by `DELIVERED`.
3. **URL-backed incoming flow:** the test calls the adapter control API on
   host port `9092`; the adapter posts `/v1/messages/receive` as the registered
   phone.
4. **Heartbeat wake-up:** the test dispatches `phone.heartbeat.missed` through
   `/v1/events`. The API sends an HTTPS callback containing
   `KEY_HEARTBEAT_ID`, and the adapter posts `/v1/heartbeats`.

The HTTPS endpoint uses a two-day throwaway CA and server certificate with the
DNS SAN `adapter-emulator`. The API container trusts only that generated CA via
`SSL_CERT_FILE`; HTTPS verification is never bypassed. The notification sender
uses the standard OpenTelemetry-instrumented Go HTTP transport.

## Test Coverage

- [x] Existing encrypted send/receive phone scenarios through WireMock
- [x] Existing rate-limit, webhook, contacts, bulk, and thread scenarios
- [x] URL-backed outgoing message reaches `delivered`
- [x] URL-backed incoming message reaches `received`
- [x] URL-backed heartbeat callback stores a heartbeat
- [x] Adapter callback notification IDs are deduplicated in memory
- [x] HTTPS certificate trust is exercised

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Docker Compose
- [Go 1.25+](https://go.dev/dl/)
- [jq](https://jqlang.github.io/jq/download/)
- [OpenSSL](https://www.openssl.org/)

On Windows, the scripts can be run with Git Bash, for example
`C:\Program Files\Git\bin\bash.exe`.

## Running Locally

### 1. Generate throwaway credentials and certificates

Run both scripts before starting Docker:

```bash
cd tests
bash generate-firebase-credentials.sh firebase-credentials.json
bash generate-adapter-certificates.sh certs
export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json)
```

The generated Firebase credential and the complete `certs/` directory are
ignored by Git.

### 2. Start the stack and wait for seeding

```bash
docker compose up -d --build --wait
docker compose wait seed
sleep 2
```

### 3. Run the complete suite

```bash
go test -v -timeout 300s ./...
```

### 4. Tear down

```bash
docker compose down -v
```

### One-liner

```bash
cd tests && \
  bash generate-firebase-credentials.sh firebase-credentials.json && \
  bash generate-adapter-certificates.sh certs && \
  export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json) && \
  docker compose up -d --build --wait && \
  docker compose wait seed && \
  sleep 2 && \
  go test -v -timeout 300s ./... ; \
  docker compose down -v
```

## CI/CD

`.github/workflows/api.yml` generates both the fake Firebase credential and
the adapter CA/server certificate before building the Compose stack. The
workflow runs API handler integration tests and this complete host-side suite,
collects service logs on failure, and always tears the stack down.

## Test Data

| Entity | Value |
| --- | --- |
| User API key | `test-user-api-key` |
| System API key | `system-user-api-key` |
| User ID | `test-user-id` |
| System user ID | `system-user-id` |

Adapter tests create a unique gateway UUID, phone number, phone API key, and
callback path per test. See [`seed.sql`](./seed.sql) for shared seed data.

## Project Structure

```text
tests/
├── adapter-emulator/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go
│   ├── emulator.go
│   ├── api_client.go
│   ├── notification_handler.go
│   ├── control_handler.go
│   └── emulator_test.go
├── wiremock/
│   └── mappings/
├── adapter_integration_test.go
├── integration_test.go
├── helpers_test.go
├── docker-compose.yml
├── .env.test
├── seed.sql
├── generate-firebase-credentials.sh
├── generate-adapter-certificates.sh
├── go.mod
└── go.sum
```

## Troubleshooting

### API or adapter fails to start

```bash
docker compose logs --tail 200 api adapter-emulator
```

Confirm `tests/certs/ca.pem`, `server.pem`, and `server-key.pem` exist. TLS
errors should be fixed by regenerating certificates; do not disable HTTPS
verification.

### URL-backed outgoing message times out

```bash
docker compose logs --tail 200 api adapter-emulator
```

Adapter logs should show callback receipt, the outstanding-message fetch,
`SENT`, and `DELIVERED`. Confirm `SSL_CERT_FILE=/adapter-certs/ca.pem` is
present in the API container.

### URL-backed incoming message times out

Adapter logs should show the control request followed by a call to
`/v1/messages/receive`. The gateway registration contains the per-test phone
number and phone API key.

### Heartbeat callback times out

API logs should show `phone.heartbeat.missed`. Adapter logs should show
`KEY_HEARTBEAT_ID` followed by a successful heartbeat POST.

### Existing FCM scenario times out

```bash
docker compose logs --tail 200 api wiremock
```

Keep `FCM_ENDPOINT=http://wiremock:8080`; the adapter service does not replace
or weaken the WireMock phone tests.

### Seed container fails

```bash
docker compose logs seed
```

If a relation does not exist, inspect API migration/startup logs before
increasing health-check timing.
