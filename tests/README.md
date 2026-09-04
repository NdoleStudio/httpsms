# Integration Tests

End-to-end integration tests for the httpSMS API and the hosted httpSMS MCP server. These tests validate the complete SMS lifecycle, and the complete MCP OAuth/tool surface, by running the full application stack in Docker.

## Architecture

```
┌──────────────┐     HTTP      ┌──────────────┐
│  Test Runner │──────────────▶│   API (Go)   │
│   (Go test)  │               │  Port 8000   │
└──────┬───────┘               └──────┬───────┘
       │                              │
       │  MCP (Streamable HTTP)       │  FCM push / events (HTTP)
       │  + OAuth 2.1                 │
       ▼                              ▼
┌──────────────┐   delegated   ┌──────────────┐
│  MCP server  │──────────────▶│   WireMock   │
│  Port 8082   │   JWT (JWKS)  │  Port 8080   │
└──────┬───────┘               └──────────────┘
       │
       │  OAuth state, confirmations, rate limits
       ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ CockroachDB  │   │    Redis     │   │   MongoDB    │
│  Port 26257  │   │  Port 6379   │   │  Port 27017  │
└──────────────┘   └──────────────┘   └──────────────┘
```

### Components

| Component       | Description                                                                       |
| --------------- | --------------------------------------------------------------------------------- |
| **API**         | The httpSMS Go API server running in Docker                                        |
| **MCP**         | The hosted httpSMS MCP server (`mcp/Dockerfile`), OAuth + MCP tools, port 8082     |
| **WireMock**    | Fake FCM, OAuth token, webhook receiver, and Firebase certificate endpoints        |
| **CockroachDB** | Database for the API (single-node, insecure mode)                                  |
| **Redis**       | Standalone Redis: API cache/queue plus MCP OAuth state, confirmations, rate limits |
| **MongoDB**     | Heartbeat and contact storage backend                                              |
| **Seed**        | One-shot container that seeds test data into CockroachDB                           |
| **Test Runner** | Go test binary that runs on the host machine                                       |

### How It Works

1. **Send SMS flow**: Test sends `POST /v1/messages/send` → API pushes an FCM notification to WireMock → test fires `SENT` and `DELIVERED` events as the phone → test polls `GET /v1/messages/{id}` until status is `delivered`

2. **Receive SMS flow**: Test sends `POST /v1/messages/receive` (as the phone) → API stores message → test verifies via `GET /v1/messages/{id}`

3. **MCP flow**: Test completes a real PKCE authorization-code flow against the MCP server (signing a Firebase ID token the MCP server verifies against the WireMock certificate endpoint) → calls MCP tools over Streamable HTTP → the MCP server mints a short-lived, operation-bound delegation JWT per call → the API verifies it against the MCP server's JWKS document

### FCM Redirect

The API's Firebase SDK is configured (via `FCM_ENDPOINT` env var) to redirect all FCM HTTP requests to WireMock instead of Google's servers. WireMock serves:

- `/token` — Fake OAuth2 token endpoint (Firebase SDK requests tokens before sending)
- `/v1/projects/:project/messages:send` — Fake FCM push endpoint
- `/webhooks/*` — Webhook receiver
- `/firebase-certs` — Firebase signing certificate map (generated, see below)

### MCP identity and signing keys

`generate-firebase-credentials.sh` generates every credential the stack needs, all throwaway and all git-ignored:

| Artifact                                         | Used by                                                                        |
| ------------------------------------------------ | ------------------------------------------------------------------------------ |
| `firebase-credentials.json`                       | API, as `FIREBASE_CREDENTIALS`                                                 |
| `mcp-test-signing-key.pem`                        | MCP container (`MCP_SIGNING_PRIVATE_KEY_FILE`) and the tests, which sign test Firebase ID tokens and delegation tokens with it |
| `mcp-test-signing-cert.pem`                       | The self-signed certificate matching that key                                  |
| `wiremock/mappings/firebase-certs.generated.json` | WireMock stub serving `{"mcp-test-key": "<certificate>"}` at `/firebase-certs` |

No secret is committed: the key, certificate, and generated mapping are listed in [`.gitignore`](./.gitignore).

## Test Coverage

- [x] **Send SMS E2E** — Full send lifecycle: API → FCM push → SENT/DELIVERED events → message reaches `delivered` status
- [x] **Receive SMS E2E** — Phone submits received message to API → message is stored and retrievable via GET endpoint
- [x] **Message thread unread count E2E** — Incoming SMS and missed calls increment the unread count, the existing thread update endpoint clears it, and outbound activity preserves the count
- [x] **Unarchive Thread on Receive E2E** — Archived thread returns to the inbox on inbound message when the phone's `unarchive_thread` setting is enabled, and stays archived when disabled
- [x] **Contacts E2E** — JSON CRUD, search and pagination totals, CSV import normalization, and contact details attached to message threads
- [x] **MCP readiness** — `/health` and `/healthz`, plus the secure response headers and request ID every response carries
- [x] **MCP discovery** — RFC 9728 protected-resource metadata (root and `/mcp`-suffixed), RFC 8414 authorization-server metadata, JWKS, CORS preflight
- [x] **MCP authentication challenge** — unauthenticated, malformed, and wrong-audience tokens are refused with `WWW-Authenticate` pointing at the resource metadata
- [x] **MCP DCR and CIMD** — dynamic client registration, rejected client metadata, and a CIMD `client_id` resolving to a private host
- [x] **MCP OAuth** — PKCE authorization-code exchange through real Firebase ID token verification, code replay, consent replay, mismatched client/redirect/resource/verifier, unverifiable identity tokens, refresh rotation and replay, scope narrowing and escalation
- [x] **MCP protocol** — `2026-07-28` discovery/tool listing through the official SDK client, `2025-11-25` initialize/tools-list/tools-call over the raw wire protocol, and the exact seven-tool catalog on both
- [x] **MCP tools** — `list_phones`, `send_sms` (through the FCM push and delivery events), `list_message_threads`, `list_thread_messages`, `list_incoming_messages` (received SMS present, missed calls absent), `create_phone_api_key`, `rotate_user_api_key`
- [x] **MCP scopes** — every tool is refused when its scope was not granted
- [x] **MCP delegation binding** — the API enforces the exact method, path, audience, issuer, expiry, and scope of every delegation token
- [x] **MCP incoming vs. search** — `/v1/messages/incoming` serves a delegated token while the CAPTCHA-protected `/v1/messages/search` stays protected
- [x] **MCP rotation confirmation** — an unconfirmed call never rotates, a legacy confirmation handle and an MRTR elicitation each rotate exactly once, the previous primary key stops working, and a redeemed handle can never be replayed
- [x] **MCP rate limits** — an exhausted per-user/per-tool budget is rejected before the tool runs, with a structured retry hint
- [x] **MCP secret handling** — minted API keys, access tokens, and refresh tokens never appear in the MCP server's logs

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Docker Compose
- [Go 1.25+](https://go.dev/dl/)
- [jq](https://jqlang.github.io/jq/download/) (for Firebase credentials generation)
- [OpenSSL](https://www.openssl.org/) (for RSA key and certificate generation)

## Running Locally

### 1. Generate Credentials

```bash
cd tests
bash generate-firebase-credentials.sh
```

This creates `firebase-credentials.json`, the MCP signing key and certificate, and the WireMock Firebase certificate mapping. Re-run it any time; every artifact is disposable.

### 2. Set Environment Variable

```bash
export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json)
```

### 3. Start the Stack

```bash
docker compose up -d --build --wait
```

This starts CockroachDB, Redis, MongoDB, WireMock, the API, and the MCP server. The `--wait` flag blocks until all health checks pass.

### 4. Wait for Seeding

```bash
docker compose wait seed
sleep 2
```

The seed container inserts test users and API keys into CockroachDB after the API has run its GORM migrations.

### 5. Run Tests

```bash
go test -v -timeout 300s ./...
```

Only the MCP suite:

```bash
go test -v -timeout 300s -run 'TestMCP' ./...
```

### 6. Tear Down

```bash
docker compose down -v
```

The `-v` flag removes volumes (database data) for a clean slate next run. **Always tear the stack down between runs**: the MCP rotation tests invalidate the seeded MCP primary API key, and the MCP rate-limit test consumes an hourly Redis budget, so both expect a freshly seeded database and an empty Redis.

### One-Liner

```bash
cd tests && \
  bash generate-firebase-credentials.sh && \
  export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json) && \
  docker compose up -d --build --wait && \
  docker compose wait seed && \
  sleep 2 && \
  go test -v -timeout 300s ./... ; \
  docker compose down -v
```

## Ports

| Port    | Service                                    |
| ------- | ------------------------------------------ |
| `8000`  | API                                        |
| `8080`  | WireMock (FCM, webhooks, Firebase certs)   |
| `8081`  | CockroachDB admin UI                       |
| `8082`  | MCP server (`/mcp`, OAuth, discovery)      |
| `6379`  | Redis                                      |
| `26257` | CockroachDB SQL                            |
| `27017` | MongoDB                                    |

## CI/CD

Integration tests run automatically via GitHub Actions (`.github/workflows/integration-test.yml`):

- **Trigger**: Push to `main` or pull request targeting `main`
- **Flow**: Generates credentials (including the MCP signing key, certificate, and WireMock certificate mapping) → Builds the API and MCP images → Starts Docker stack → Seeds DB → Runs tests → Collects logs on failure → Tears down
- **Gate**: Deployment should only proceed if integration tests pass

## Test Data

| Entity                | Value                        |
| --------------------- | ---------------------------- |
| User API Key          | `test-user-api-key`          |
| User ID               | `test-user-id`               |
| Rotation User API Key | `rotate-test-api-key`        |
| Rotation User ID      | `rotate-test-user-id`        |
| MCP User API Key      | `mcp-test-user-api-key`      |
| MCP User ID           | `mcp-test-user-id`           |
| MCP Rate-limit User   | `mcp-rate-limit-user-id`     |
| MCP signing key ID    | `mcp-test-key`               |
| Firebase project      | `httpsms-test`               |

Phones, phone API keys, and message threads are created at runtime by the tests themselves.

See [`seed.sql`](./seed.sql) for the complete seed data.

## Project Structure

```
tests/
├── docker-compose.yml       # Full stack orchestration
├── seed.sql                 # Database seed data
├── .env.test                # API and MCP environment variables
├── .gitignore               # Generated credentials and key material
├── generate-firebase-credentials.sh  # Generates credentials and MCP signing key material
├── go.mod
├── go.sum
├── helpers_test.go          # API test utilities (HTTP client, polling)
├── integration_test.go      # API E2E test cases
├── contacts_integration_test.go
├── read_receipts_test.go
├── unarchive_thread_integration_test.go
├── mcp_helpers_test.go      # MCP/OAuth test utilities (token signing, OAuth flow, MCP client)
├── mcp_integration_test.go  # MCP E2E test cases
└── wiremock/mappings/       # FCM, OAuth token, webhook, and Firebase certificate stubs
```

## Troubleshooting

### API fails to start

Check the API logs:

```bash
docker compose logs api
```

Common issues:

- `FIREBASE_CREDENTIALS` env var not set or malformed
- CockroachDB not ready (increase `start_period` in healthcheck)

### MCP server fails to start

```bash
docker compose logs mcp
```

Common issues:

- `mcp-test-signing-key.pem` missing — run `bash generate-firebase-credentials.sh`
- The key file is not readable by the container's unprivileged `mcp` user (the generator sets mode `0644`)
- A configuration error: the MCP server names every missing or invalid setting in a single startup error

### MCP tests fail with `access_denied` on the consent step

WireMock is not serving the generated certificate, or the certificate no longer matches the signing key. Re-run `bash generate-firebase-credentials.sh` and restart WireMock so it reloads its mappings:

```bash
docker compose restart wiremock mcp
```

### MCP tests fail with 401 on the API

The API could not verify the delegation token. Check that `MCP_AUTH_ISSUER`, `MCP_AUTH_AUDIENCE`, and `MCP_AUTH_JWKS_URL` in `.env.test` still match the MCP service's `MCP_BASE_URL`, `API_AUDIENCE`, and JWKS route.

### MCP rotation or rate-limit tests fail on a re-run

Both mutate state that outlives a test run (the seeded primary API key, and an hourly Redis counter). Tear the stack down with `docker compose down -v` before running again.

### Tests timeout waiting for `delivered` status

Check the API and WireMock logs:

```bash
docker compose logs api wiremock
```

If no FCM request appears, the API isn't reaching WireMock (check `FCM_ENDPOINT` in `.env.test`).

### Seed container fails

```bash
docker compose logs seed
```

If you see "relation does not exist" errors, the API hasn't finished GORM migrations yet. Increase the API's `start_period` in `docker-compose.yml`.

## Adding New Tests

1. Add test functions to `integration_test.go` (or create new `*_test.go` files); MCP cases belong in `mcp_integration_test.go`
2. Use `requestJSON()`/`requestJSONAs()` for authenticated HTTP calls
3. Use `pollMessageStatus()`/`pollMessageStatusAs()` to wait for async state changes
4. For MCP cases, use `completeOAuthCodeFlow()` and `newMCPClient()`; never rotate `test-user-api-key`
5. Update the test coverage checklist in this README

