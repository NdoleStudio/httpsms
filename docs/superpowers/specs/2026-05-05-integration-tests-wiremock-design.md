# Integration Tests: WireMock + httpsms-go Client Refactor

## Problem

The current integration tests use raw `net/http` calls and a custom emulator (120+ lines of Go) to simulate phone behavior. This makes tests harder to maintain and doesn't cover encryption, rate limiting, or webhook verification. We need to:

1. Refactor tests to use the official `httpsms-go` client SDK
2. Replace the custom emulator with WireMock (stub server + request journal)
3. Add E2E encryption tests (outgoing + incoming)
4. Add rate-limit verification test
5. Assert webhook delivery with JWT authentication in all tests

## Architecture

```
┌────────────────────────────────────────────────────────┐
│  Docker Compose (tests/docker-compose.yml)             │
│                                                        │
│  ┌──────────┐  ┌───────┐  ┌─────────────────────────┐ │
│  │PostgreSQL│  │ Redis │  │  API                    │ │
│  └──────────┘  └───────┘  └────────────┬────────────┘ │
│                                         │FCM push      │
│                                         │Webhook calls │
│                                         ▼              │
│                            ┌─────────────────────────┐ │
│                            │  WireMock 3.x (:8080)   │ │
│                            │  - Fake FCM endpoint    │ │
│                            │  - Fake OAuth token     │ │
│                            │  - Webhook receiver     │ │
│                            │  - Request journal      │ │
│                            └─────────────────────────┘ │
└────────────────────────────────────────────────────────┘
         ▲
         │ httpsms-go client + go-wiremock client
┌────────┴──────────┐
│  Test Runner (Go) │
│  go test ./...    │
└───────────────────┘
```

### Key Design Decisions

- **WireMock replaces the custom emulator entirely**. It serves as both the fake FCM endpoint (receives push notifications from the API) and the webhook receiver (captures webhook events).
- **Tests fire SENT/DELIVERED events directly** to the API via HTTP. No WireMock callbacks needed — the test controls the flow deterministically.
- **Each test creates its own phone** with a random phone number for parallel test isolation.
- **go-wiremock** (`github.com/wiremock/go-wiremock`) is used to configure stubs and query the request journal from test code.

## Test Flow (per test)

```
1. SETUP
   ├─ Create phone (random number, test-specific messages_per_minute)
   ├─ Create phone API key for that phone
   ├─ Create webhook pointing to WireMock with a signing key
   └─ Configure WireMock stubs (if not pre-loaded)

2. ACT
   ├─ Send/receive message via httpsms-go client
   └─ (For send tests) Query WireMock journal → extract KEY_MESSAGE_ID from FCM push

3. SIMULATE PHONE
   ├─ Fire SENT event to API (POST /v1/messages/{id}/events)
   └─ Fire DELIVERED event to API

4. ASSERT
   ├─ Verify message reached expected status via httpsms-go client
   ├─ Query WireMock journal for webhook events
   ├─ Validate JWT token: signature (HMAC-SHA256), issuer, subject, audience, expiry
   └─ Validate webhook payload contains correct event type and message data
```

## Components

### 1. Docker Compose Changes

**Remove:**

- `tests/emulator/` directory entirely (Dockerfile, Go source, go.mod)

**Replace with WireMock:**

```yaml
wiremock:
  image: wiremock/wiremock:3x
  ports:
    - "8080:8080"
  volumes:
    - ./wiremock/mappings:/home/wiremock/mappings:ro
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/__admin/health"]
    interval: 5s
    timeout: 5s
    retries: 10
```

**Pre-loaded WireMock mappings** (`tests/wiremock/mappings/`):

- `fcm-send.json` — Stub for `POST /v1/projects/*/messages:send` → returns `{"name": "projects/httpsms-test/messages/fake-id"}`
- `oauth-token.json` — Stub for `POST /token` → returns `{"access_token": "fake-access-token", "token_type": "Bearer", "expires_in": 3600}`
- `webhook-receiver.json` — Stub for `POST /webhooks/test` → returns 200 (catches all webhook calls)

### 2. API Configuration Updates

**`.env.test` changes:**

- `FCM_ENDPOINT=http://wiremock:8080` (was `http://emulator:9090`)

**Firebase credentials** `token_uri` points to `http://wiremock:8080/token`

### 3. Seed SQL (simplified)

Only seeds:

- Test user (`test-user-id`, `api_key='test-user-api-key'`)
- System user (`system-user-id`, for event queue auth)

Phones, phone API keys, and webhooks are created per-test via the API.

### 4. httpsms-go Client Additions

New services to add to `github.com/NdoleStudio/httpsms-go`:

#### `PhoneService`

```go
type PhoneUpsertParams struct {
    PhoneNumber              string `json:"phone_number"`
    FcmToken                 string `json:"fcm_token"`
    MessagesPerMinute        uint   `json:"messages_per_minute"`
    MaxSendAttempts          uint   `json:"max_send_attempts"`
    MessageExpirationSeconds uint   `json:"message_expiration_seconds"`
    SIM                      string `json:"sim"`
}

func (service *PhoneService) Upsert(ctx, params) → (*PhoneResponse, *Response, error)
// PUT /v1/phones — authenticated with user API key
```

#### `PhoneService` (FCM Token binding)

```go
type PhoneFCMTokenParams struct {
    PhoneNumber string `json:"phone_number"`
    FcmToken    string `json:"fcm_token"`
    SIM         string `json:"sim"`
}

func (service *PhoneService) UpsertFCMToken(ctx, params) → (*PhoneResponse, *Response, error)
// PUT /v1/phones/fcm-token — authenticated with phone API key
// This binds the phone to the phone API key via the auth context
```

#### `PhoneAPIKeyService`

```go
type PhoneAPIKeyStoreParams struct {
    Name string `json:"name"`
}

func (service *PhoneAPIKeyService) Store(ctx, params) → (*PhoneAPIKeyResponse, *Response, error)
// POST /v1/phone-api-keys/ — authenticated with user API key
// Returns the created phone API key including its api_key value
```

#### `WebhookService`

```go
type WebhookStoreParams struct {
    SigningKey    string   `json:"signing_key"`
    URL          string   `json:"url"`
    PhoneNumbers []string `json:"phone_numbers"`
    Events       []string `json:"events"`
}

func (service *WebhookService) Store(ctx, params) → (*WebhookResponse, *Response, error)
// POST /v1/webhooks — authenticated with user API key
```

#### Phone Setup Flow (per test)

The real Android phone registers via this flow, and tests must replicate it:

1. `PUT /v1/phones` (user API key) — creates phone with phone_number + fcm_token + messages_per_minute
2. `POST /v1/phone-api-keys/` (user API key) — creates a phone API key, returns the `api_key` value
3. `PUT /v1/phones/fcm-token` (phone API key) — re-registers FCM token, which binds the phone to the API key via `PhoneAPIKeyListener.onPhoneUpdated`

After step 3, the phone API key is authorized to act on behalf of that phone (fire events, receive messages, etc.).

### 5. Test Cases

#### `TestSendSMS_Encrypted`

1. Generate random encryption key
2. Create phone + phone API key + webhook
3. Encrypt plaintext using `client.Cipher.Encrypt(key, "secret message")`
4. Send message with `Encrypted: true` and encrypted content
5. Query WireMock journal → verify FCM push arrived with `KEY_MESSAGE_ID` (FCM only carries the message ID, not content)
6. Call `GET /v1/messages/outstanding?message_id={id}` (phone API key) — verify response has `encrypted: true` and content is ciphertext (not plaintext)
7. Fire SENT + DELIVERED events
8. Fetch message via user API key → verify `encrypted: true`, content is ciphertext
9. Decrypt with `client.Cipher.Decrypt(key, content)` → assert equals original plaintext
10. Verify webhook event in WireMock with valid JWT

#### `TestReceiveSMS_Encrypted`

1. Generate random encryption key
2. Create phone + phone API key + webhook
3. Encrypt plaintext using `client.Cipher.Encrypt(key, "incoming secret")`
4. Simulate receiving an encrypted SMS (POST /v1/messages/receive with phone API key)
5. Fetch message via user API key → verify `encrypted: true`
6. Decrypt content → assert equals original plaintext
7. Verify webhook event (`message.phone.received`) in WireMock with valid JWT

#### `TestSendSMS_RateLimit`

1. Create phone with `messages_per_minute: 10` (= 6s gap)
2. Create phone API key + webhook
3. Send 2 messages simultaneously
4. Query WireMock journal for FCM pushes (correlate by message IDs from send responses)
5. Assert the timestamps of the two FCM pushes have ≥6 second gap
6. Fire SENT + DELIVERED for both messages
7. Verify both messages reach `delivered` status
8. Verify webhook events for both messages

#### `TestSendSMS_OutstandingFlow`

Validates the real phone flow (`/v1/messages/outstanding`):

1. Create phone + phone API key + webhook
2. Send message via httpsms-go client
3. Query WireMock journal → extract `KEY_MESSAGE_ID` from FCM push
4. Call `GET /v1/messages/outstanding?message_id={id}` (phone API key) — assert returns the message with correct content, owner, contact
5. Fire SENT + DELIVERED events
6. Verify message reaches `delivered` status
7. Verify webhook events

#### Webhook Verification (shared helper)

For all tests, a helper function:

```go
func assertWebhookEvent(t *testing.T, wiremockClient *wiremock.Client, signingKey string, expectedEventType string) {
    // 1. Query WireMock journal for POST /webhooks/test requests
    // 2. Find request with X-Event-Type header matching expectedEventType
    // 3. Extract Authorization header → parse JWT
    // 4. Validate signature with signingKey (HMAC-SHA256)
    // 5. Assert claims:
    //    - Issuer == "api.httpsms.com"
    //    - Subject == "test-user-id"
    //    - Audience contains webhook URL
    //    - ExpiresAt is in the future
    //    - NotBefore is in the past
}
```

### 6. Test Helper Structure

```
tests/
├── docker-compose.yml          (updated: wiremock replaces emulator)
├── wiremock/
│   └── mappings/
│       ├── fcm-send.json
│       ├── oauth-token.json
│       └── webhook-receiver.json
├── seed.sql                    (simplified: user + system user only)
├── .env.test                   (updated: FCM_ENDPOINT → wiremock)
├── go.mod                      (add httpsms-go, go-wiremock, golang-jwt)
├── helpers_test.go             (shared constants, setup helpers)
├── webhook_helpers_test.go     (JWT verification helpers)
├── integration_test.go         (all test cases)
└── README.md
```

### 7. Dependencies

**Test module (`tests/go.mod`):**

- `github.com/NdoleStudio/httpsms-go` — API client
- `github.com/wiremock/go-wiremock` — WireMock stub configuration + journal queries
- `github.com/golang-jwt/jwt/v5` — JWT parsing and validation
- `github.com/stretchr/testify` — assertions (already present)

### 8. Parallel Test Execution & Request Correlation

Each test creates its own phone with a unique random number (e.g. `+1800555XXXX` where XXXX is random). This ensures:

- No message cross-contamination between tests
- Webhooks scoped to specific phone numbers don't fire for other tests

**Correlation strategy for WireMock journal queries:**

- **FCM pushes**: Correlate by message ID. The test gets the message ID from the send response, then searches WireMock journal for FCM push requests containing that `KEY_MESSAGE_ID` in the JSON body.
- **Webhook events**: Each test uses a **unique webhook URL path** (e.g. `/webhooks/{testUUID}`). This ensures journal queries for webhook assertions only match events for that specific test. Additionally, match on `X-Event-Type` header and message ID in payload body.
- **Unique FCM token per phone**: Each test generates a unique `fcm_token` string. Since WireMock captures the FCM push including the `token` field, this can be used as a secondary correlation key if needed.

Tests use `t.Parallel()` where safe (encryption tests can run in parallel; rate-limit test may need serial execution due to timing assertions).

## Migration Notes

- The `tests/emulator/` directory is deleted entirely
- The CI workflow (`.github/workflows/integration-test.yml`) needs updating to remove emulator references
- Firebase credentials `token_uri` must point to `http://wiremock:8080/token`
- WireMock image is Java-based (~300MB) vs the old Alpine emulator (~15MB), but eliminates maintenance of custom code
