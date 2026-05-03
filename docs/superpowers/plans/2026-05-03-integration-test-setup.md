# Integration Test Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a CI-gated integration test that validates the full SMS send/receive flow using Docker, a phone emulator, and real FCM code paths redirected to the emulator.

**Architecture:** Docker Compose brings up PostgreSQL + Redis + API + Emulator. The API's Firebase SDK is configured to route FCM traffic to the emulator via a custom HTTP transport. A Go test runner on the host exercises the API and asserts on message state.

**Tech Stack:** Go, Docker Compose, PostgreSQL, Redis, Firebase Admin Go SDK, GitHub Actions

---

## File Structure

```
tests/
├── docker-compose.yml          # orchestrates all services
├── seed.sql                    # seeds test user, phone, API keys
├── .env.test                   # API environment config for tests
├── firebase-credentials.json   # fake service account JSON
├── go.mod                      # test runner Go module
├── go.sum
├── integration_test.go         # test cases (send SMS, receive SMS)
├── helpers_test.go             # HTTP client, polling, constants
└── emulator/
    ├── Dockerfile              # builds emulator binary
    ├── go.mod                  # emulator Go module
    ├── go.sum
    ├── main.go                 # entry point, HTTP server setup
    ├── fcm_handler.go          # fake FCM endpoint handler
    ├── token_handler.go        # fake OAuth2 token endpoint
    └── events.go               # fires SENT/DELIVERED events to API

api/pkg/di/container.go         # modified: FCM transport redirect
.github/workflows/integration-test.yml  # new CI workflow
```

---

### Task 1: Create Feature Branch

**Files:**

- None (git operations only)

- [ ] **Step 1: Create and switch to feature branch from main**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com
git checkout main
git pull origin main
git checkout -b feature/integration-tests
```

- [ ] **Step 2: Verify branch**

Run: `git branch --show-current`
Expected: `feature/integration-tests`

---

### Task 2: API Modification — FCM Transport Override

**Files:**

- Modify: `api/pkg/di/container.go:396-405` (FirebaseApp method)

- [ ] **Step 1: Add the FCM redirect transport and modify FirebaseApp**

In `api/pkg/di/container.go`, modify the `FirebaseApp()` method to check for `FCM_ENDPOINT` env var. When set, use ONLY a custom HTTP client (no credentials). When not set, use credentials as before.

**Important:** `option.WithHTTPClient()` takes precedence over all other options in the Firebase SDK. Do NOT combine it with `option.WithAuthCredentialsJSON()`. Use one or the other.

Create a new file `api/pkg/di/fcm_transport.go`:

```go
package di

import (
	"net/http"
	"net/url"
)

// fcmRedirectTransport rewrites Firebase SDK HTTP requests to a custom endpoint.
// Used in integration tests to redirect FCM traffic to the emulator.
type fcmRedirectTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t *fcmRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return t.base.RoundTrip(req)
}
```

Then modify `FirebaseApp()` in `container.go`:

```go
// FirebaseApp creates a new instance of firebase.App
func (container *Container) FirebaseApp() (app *firebase.App) {
	container.logger.Debug(fmt.Sprintf("creating %T", app))

	var opts []option.ClientOption

	if fcmEndpoint := os.Getenv("FCM_ENDPOINT"); fcmEndpoint != "" {
		container.logger.Info(fmt.Sprintf("using FCM endpoint override: %s", fcmEndpoint))
		targetURL, err := url.Parse(fcmEndpoint)
		if err != nil {
			container.logger.Fatal(stacktrace.Propagate(err, "cannot parse FCM_ENDPOINT"))
		}
		opts = append(opts, option.WithHTTPClient(&http.Client{
			Transport: &fcmRedirectTransport{
				target: targetURL,
				base:   http.DefaultTransport,
			},
		}))
	} else {
		opts = append(opts, option.WithAuthCredentialsJSON(option.ServiceAccount, container.FirebaseCredentials()))
	}

	app, err := firebase.NewApp(context.Background(), nil, opts...)
	if err != nil {
		msg := "cannot initialize firebase application"
		container.logger.Fatal(stacktrace.Propagate(err, msg))
	}
	return app
}
```

- [ ] **Step 2: Add `net/url` import if not already present**

Ensure the `net/url` package is imported in `container.go` (or the new file).

- [ ] **Step 3: Verify API still builds**

Run: `cd api && go build ./...`
Expected: Build succeeds with no errors.

- [ ] **Step 4: Commit**

```bash
git add api/pkg/di/
git commit -m "feat(api): add FCM_ENDPOINT transport override for integration tests"
```

---

### Task 3: Emulator — Project Scaffolding

**Files:**

- Create: `tests/emulator/go.mod`
- Create: `tests/emulator/emulator.go`
- Create: `tests/emulator/Dockerfile`

Note: `main.go` references `NewEmulator()` and handlers, so we create the struct first. `main.go` is created AFTER all handlers exist (Task 6b).

- [ ] **Step 1: Initialize emulator Go module**

```bash
mkdir -p tests/emulator
cd tests/emulator
go mod init github.com/NdoleStudio/httpsms/tests/emulator
```

- [ ] **Step 2: Create `tests/emulator/emulator.go`**

```go
package main

import "net/http"

// Emulator acts as a fake Android phone that receives FCM pushes
// and responds with message events.
type Emulator struct {
	apiBaseURL  string
	phoneAPIKey string
	httpClient  *http.Client
}

// NewEmulator creates a new Emulator instance.
func NewEmulator(apiBaseURL, phoneAPIKey string) *Emulator {
	return &Emulator{
		apiBaseURL:  apiBaseURL,
		phoneAPIKey: phoneAPIKey,
		httpClient:  &http.Client{},
	}
}

// HealthHandler returns 200 OK for health checks.
func (e *Emulator) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
```

- [ ] **Step 3: Create `tests/emulator/Dockerfile`**

```dockerfile
FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/emulator .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/emulator /bin/emulator
EXPOSE 9090
ENTRYPOINT ["/bin/emulator"]
```

- [ ] **Step 4: Commit**

```bash
git add tests/emulator/
git commit -m "feat(tests): scaffold emulator Go project"
```

---

### Task 4: Emulator — Token Handler

**Files:**

- Create: `tests/emulator/token_handler.go`

- [ ] **Step 1: Create `tests/emulator/token_handler.go`**

```go
package main

import (
	"encoding/json"
	"net/http"
)

// TokenHandler returns a fake OAuth2 access token.
// The Firebase Admin SDK calls this endpoint to get an access token
// before making FCM API calls.
func (e *Emulator) TokenHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"access_token": "fake-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
```

- [ ] **Step 2: Commit**

```bash
git add tests/emulator/token_handler.go
git commit -m "feat(tests): add fake OAuth2 token handler to emulator"
```

---

### Task 5: Emulator — FCM Handler

**Files:**

- Create: `tests/emulator/fcm_handler.go`

- [ ] **Step 1: Create `tests/emulator/fcm_handler.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// fcmRequest represents the FCM v1 API request body
type fcmRequest struct {
	Message struct {
		Data    map[string]string `json:"data"`
		Token   string            `json:"token"`
		Android struct {
			Priority string `json:"priority"`
		} `json:"android"`
	} `json:"message"`
}

// fcmResponse represents the FCM v1 API response
type fcmResponse struct {
	Name string `json:"name"`
}

// FCMHandler handles fake FCM send requests from the Firebase Admin SDK.
func (e *Emulator) FCMHandler(w http.ResponseWriter, r *http.Request) {
	var req fcmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	messageID := req.Message.Data["KEY_MESSAGE_ID"]
	if messageID == "" {
		http.Error(w, "missing KEY_MESSAGE_ID in data", http.StatusBadRequest)
		return
	}

	log.Printf("received FCM push for message: %s", messageID)

	// Respond with success immediately (like real FCM would)
	resp := fcmResponse{
		Name: fmt.Sprintf("projects/httpsms-test/messages/fake-%s", messageID),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	// Process the message asynchronously (like a real phone would)
	go e.processMessage(messageID)
}
```

- [ ] **Step 3: Commit**

```bash
git add tests/emulator/emulator.go tests/emulator/fcm_handler.go
git commit -m "feat(tests): add FCM handler to emulator"
```

---

### Task 6: Emulator — Event Firing

**Files:**

- Create: `tests/emulator/events.go`

- [ ] **Step 1: Create `tests/emulator/events.go`**

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// messageEvent is the payload for posting a message event to the API
type messageEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventName string    `json:"event_name"`
}

// processMessage simulates a phone receiving an FCM push and sending the SMS.
// It calls /messages/outstanding, then fires SENT and DELIVERED events.
func (e *Emulator) processMessage(messageID string) {
	// Step 1: Fetch outstanding message (like real phone does)
	e.fetchOutstanding(messageID)

	// Step 2: Wait briefly then fire SENT
	time.Sleep(200 * time.Millisecond)
	if err := e.fireEvent(messageID, "SENT"); err != nil {
		log.Printf("error firing SENT event for message %s: %v", messageID, err)
		return
	}

	// Step 3: Wait briefly then fire DELIVERED
	time.Sleep(200 * time.Millisecond)
	if err := e.fireEvent(messageID, "DELIVERED"); err != nil {
		log.Printf("error firing DELIVERED event for message %s: %v", messageID, err)
		return
	}

	log.Printf("completed processing message: %s", messageID)
}

// fetchOutstanding calls GET /v1/messages/outstanding to mimic the real phone behavior
func (e *Emulator) fetchOutstanding(messageID string) {
	url := fmt.Sprintf("%s/v1/messages/outstanding?message_id=%s", e.apiBaseURL, messageID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("x-api-key", e.phoneAPIKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		log.Printf("error fetching outstanding message %s: %v", messageID, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("fetched outstanding message %s: status %d", messageID, resp.StatusCode)
}

// fireEvent posts a message event (SENT or DELIVERED) to the API
func (e *Emulator) fireEvent(messageID, eventName string) error {
	url := fmt.Sprintf("%s/v1/messages/%s/events", e.apiBaseURL, messageID)

	event := messageEvent{
		Timestamp: time.Now().UTC(),
		EventName: eventName,
	}

	body, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.phoneAPIKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned status %d for %s event", resp.StatusCode, eventName)
	}

	log.Printf("fired %s event for message %s: status %d", eventName, messageID, resp.StatusCode)
	return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add tests/emulator/events.go
git commit -m "feat(tests): add event firing to emulator"
```

---

### Task 6b: Emulator — Main Entry Point

**Files:**

- Create: `tests/emulator/main.go`

- [ ] **Step 1: Create `tests/emulator/main.go`**

Now that all handlers exist (HealthHandler, TokenHandler, FCMHandler), create the entry point:

```go
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://api:8000"
	}

	phoneAPIKey := os.Getenv("PHONE_API_KEY")
	if phoneAPIKey == "" {
		phoneAPIKey = "pk_test-phone-api-key"
	}

	emulator := NewEmulator(apiBaseURL, phoneAPIKey)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", emulator.HealthHandler)
	mux.HandleFunc("POST /token", emulator.TokenHandler)
	mux.HandleFunc("POST /v1/projects/{project}/messages:send", emulator.FCMHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	log.Printf("emulator listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 2: Verify emulator builds**

```bash
cd tests/emulator
go build ./...
```

Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add tests/emulator/main.go
git commit -m "feat(tests): add emulator main entry point"
```

---

### Task 7: Test Infrastructure — Seed Data & Config

**Files:**

- Create: `tests/seed.sql`
- Create: `tests/.env.test`
- Create: `tests/firebase-credentials.json`

- [ ] **Step 1: Create `tests/seed.sql`**

This script must match the exact table schema from entities. The tables are auto-migrated by GORM, so we insert after API startup. Actually — since we need the user to exist BEFORE the API processes requests, we seed via Docker's postgres init scripts.

Note: GORM auto-migrates tables on API startup. The seed SQL runs AFTER table creation. We use a Docker healthcheck + depends_on to ensure ordering. Alternatively, we can use a startup script that waits for the API to be ready, then seeds. The simplest approach: mount `seed.sql` as a Postgres init script — but that runs before GORM migrates.

**Better approach:** Create a `tests/seed.sh` script that waits for the API to start (which runs GORM migrations), then seeds the database via `psql`.

```sql
-- tests/seed.sql
-- Seed test data for integration tests
-- Run AFTER GORM has migrated the schema (i.e., after API starts)

-- Test user
INSERT INTO users (id, email, api_key, timezone, subscription_name, created_at, updated_at)
VALUES (
    'test-user-id',
    'test@httpsms.com',
    'test-user-api-key',
    'UTC',
    'pro-monthly',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- System user (for event queue auth)
INSERT INTO users (id, email, api_key, timezone, subscription_name, created_at, updated_at)
VALUES (
    'system-user-id',
    'system@httpsms.com',
    'system-user-api-key',
    'UTC',
    'pro-monthly',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Test phone
INSERT INTO phones (id, user_id, fcm_token, phone_number, messages_per_minute, sim, max_send_attempts, message_expiration_seconds, created_at, updated_at)
VALUES (
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    'test-user-id',
    'fake-fcm-token',
    '+18005550199',
    60,
    'SIM1',
    2,
    600,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Phone API key (for emulator to authenticate as phone)
INSERT INTO phone_api_keys (id, name, user_id, user_email, phone_numbers, phone_ids, api_key, created_at, updated_at)
VALUES (
    'b2c3d4e5-f6a7-8901-bcde-f12345678901',
    'Integration Test Phone Key',
    'test-user-id',
    'test@httpsms.com',
    '{"+18005550199"}',
    '{"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}',
    'pk_test-phone-api-key',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;
```

- [ ] **Step 2: Create `tests/.env.test`**

```env
ENV=production
GCP_PROJECT_ID=httpsms-test
USE_HTTP_LOGGER=true
ENTITLEMENT_ENABLED=false
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
APP_NAME=httpSMS
APP_URL=http://localhost:8000
SWAGGER_HOST=localhost:8000
SMTP_FROM_NAME=httpSMS
SMTP_FROM_EMAIL=test@httpsms.com
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_HOST=localhost
SMTP_PORT=2525
PUSHER_APP_ID=
PUSHER_KEY=
PUSHER_SECRET=
PUSHER_CLUSTER=
GCS_BUCKET_NAME=
UPTRACE_DSN=
CLOUDFLARE_TURNSTILE_SECRET_KEY=
```

- [ ] **Step 3: Create `tests/firebase-credentials.json`**

Generate an RSA private key for the fake service account. This must be a valid RSA key so the Firebase SDK can sign JWT tokens (even though the emulator won't validate them).

```bash
cd tests
openssl genrsa -out /tmp/test-key.pem 2048
```

Then create the JSON file with the key embedded:

```json
{
  "type": "service_account",
  "project_id": "httpsms-test",
  "private_key_id": "test-key-id",
  "private_key": "<contents of /tmp/test-key.pem with newlines as \\n>",
  "client_email": "test@httpsms-test.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "http://emulator:9090/auth",
  "token_uri": "http://emulator:9090/token",
  "auth_provider_x509_cert_url": "http://emulator:9090/certs",
  "client_x509_cert_url": "http://emulator:9090/certs/test"
}
```

Note: The `FIREBASE_CREDENTIALS` env var in `.env.test` should be set to the full contents of this JSON file (single-line). The docker-compose will handle this.

- [ ] **Step 4: Commit**

```bash
git add tests/seed.sql tests/.env.test tests/firebase-credentials.json
git commit -m "feat(tests): add seed data and test environment config"
```

---

### Task 8: Docker Compose for Tests

**Files:**

- Create: `tests/docker-compose.yml`

- [ ] **Step 1: Create `tests/docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:alpine
    environment:
      POSTGRES_DB: httpsms
      POSTGRES_PASSWORD: dbpassword
      POSTGRES_USER: dbusername
    ports:
      - "5435:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dbusername -d httpsms"]
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 5s

  redis:
    image: redis:latest
    command: redis-server
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 10

  emulator:
    build:
      context: ./emulator
    ports:
      - "9090:9090"
    environment:
      API_BASE_URL: http://api:8000
      PHONE_API_KEY: pk_test-phone-api-key
      PORT: "9090"
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:9090/health"]
      interval: 5s
      timeout: 5s
      retries: 10

  api:
    build:
      context: ../api
    ports:
      - "8000:8000"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      emulator:
        condition: service_healthy
    env_file:
      - .env.test
    environment:
      FIREBASE_CREDENTIALS: "${FIREBASE_CREDENTIALS}"
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8000/"]
      interval: 5s
      timeout: 10s
      retries: 20
      start_period: 30s

  seed:
    image: postgres:alpine
    depends_on:
      api:
        condition: service_healthy
    environment:
      PGPASSWORD: dbpassword
    volumes:
      - ./seed.sql:/seed.sql:ro
    entrypoint:
      [
        "psql",
        "-h",
        "postgres",
        "-U",
        "dbusername",
        "-d",
        "httpsms",
        "-f",
        "/seed.sql",
      ]
    restart: "no"
```

- [ ] **Step 2: Commit**

```bash
git add tests/docker-compose.yml
git commit -m "feat(tests): add docker-compose for integration test stack"
```

---

### Task 9: Test Runner — Go Module & Helpers

**Files:**

- Create: `tests/go.mod`
- Create: `tests/helpers_test.go`

- [ ] **Step 1: Initialize test runner Go module**

```bash
cd tests
go mod init github.com/NdoleStudio/httpsms/tests
go get github.com/stretchr/testify
```

- [ ] **Step 2: Create `tests/helpers_test.go`**

```go
package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	apiBaseURL   = "http://localhost:8000"
	userAPIKey   = "test-user-api-key"
	phoneAPIKey  = "pk_test-phone-api-key"
	testPhone    = "+18005550199"
	testContact  = "+18005550100"
)

// apiClient returns an HTTP client configured for API calls
func apiClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// doRequest performs an HTTP request with the given API key
func doRequest(t *testing.T, method, url string, body io.Reader, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	resp, err := apiClient().Do(req)
	require.NoError(t, err)
	return resp
}

// pollMessageStatus polls GET /v1/messages/{id} until the message reaches the target status or times out
func pollMessageStatus(t *testing.T, messageID, targetStatus string, timeout time.Duration) map[string]interface{} {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		url := fmt.Sprintf("%s/v1/messages/%s", apiBaseURL, messageID)
		resp := doRequest(t, "GET", url, nil, userAPIKey)

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			require.NoError(t, json.Unmarshal(body, &result))

			data, ok := result["data"].(map[string]interface{})
			if ok && data["status"] == targetStatus {
				return data
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("message %s did not reach status %q within %v", messageID, targetStatus, timeout)
	return nil
}
```

- [ ] **Step 3: Commit**

```bash
git add tests/go.mod tests/go.sum tests/helpers_test.go
git commit -m "feat(tests): add test runner module and helpers"
```

---

### Task 10: Test Runner — Integration Tests

**Files:**

- Create: `tests/integration_test.go`

- [ ] **Step 1: Create `tests/integration_test.go`**

```go
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendSMS_E2E(t *testing.T) {
	// Step 1: Send an SMS via the API
	sendPayload := map[string]interface{}{
		"from":    testPhone,
		"to":      testContact,
		"content": "Hello from integration test",
	}
	body, _ := json.Marshal(sendPayload)

	url := fmt.Sprintf("%s/v1/messages/send", apiBaseURL)
	resp := doRequest(t, "POST", url, bytes.NewReader(body), userAPIKey)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "send response: %s", string(respBody))

	// Step 2: Extract message ID
	var sendResult map[string]interface{}
	require.NoError(t, json.Unmarshal(respBody, &sendResult))
	data := sendResult["data"].(map[string]interface{})
	messageID := data["id"].(string)
	require.NotEmpty(t, messageID)

	t.Logf("sent message with ID: %s", messageID)

	// Step 3: Poll until message is delivered
	message := pollMessageStatus(t, messageID, "delivered", 15*time.Second)

	// Step 4: Assert final state
	assert.Equal(t, "delivered", message["status"])
	assert.Equal(t, testPhone, message["owner"])
	assert.Equal(t, testContact, message["contact"])
	assert.Equal(t, "Hello from integration test", message["content"])
}

func TestReceiveSMS_E2E(t *testing.T) {
	// Step 1: Simulate receiving an SMS (phone -> API)
	receivePayload := map[string]interface{}{
		"from":      testContact,
		"to":        testPhone,
		"content":   "Hi there from integration test",
		"encrypted": false,
		"sim":       "SIM1",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	body, _ := json.Marshal(receivePayload)

	url := fmt.Sprintf("%s/v1/messages/receive", apiBaseURL)
	resp := doRequest(t, "POST", url, bytes.NewReader(body), phoneAPIKey)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "receive response: %s", string(respBody))

	// Step 2: Extract message ID
	var receiveResult map[string]interface{}
	require.NoError(t, json.Unmarshal(respBody, &receiveResult))
	data := receiveResult["data"].(map[string]interface{})
	messageID := data["id"].(string)
	require.NotEmpty(t, messageID)

	t.Logf("received message with ID: %s", messageID)

	// Step 3: Verify message exists via GET
	getURL := fmt.Sprintf("%s/v1/messages/%s", apiBaseURL, messageID)
	getResp := doRequest(t, "GET", getURL, nil, userAPIKey)
	defer getResp.Body.Close()

	getBody, err := io.ReadAll(getResp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var getMessage map[string]interface{}
	require.NoError(t, json.Unmarshal(getBody, &getMessage))
	messageData := getMessage["data"].(map[string]interface{})

	// Step 4: Assert message fields
	assert.Equal(t, "received", messageData["status"])
	assert.Equal(t, testPhone, messageData["owner"])
	assert.Equal(t, testContact, messageData["contact"])
	assert.Equal(t, "Hi there from integration test", messageData["content"])
}
```

- [ ] **Step 2: Verify test file compiles**

```bash
cd tests
go vet ./...
```

Expected: No errors (tests won't pass yet without the stack running).

- [ ] **Step 3: Commit**

```bash
git add tests/integration_test.go
git commit -m "feat(tests): add send and receive SMS integration tests"
```

---

### Task 11: GitHub Actions Workflow

**Files:**

- Create: `.github/workflows/integration-test.yml`

- [ ] **Step 1: Create `.github/workflows/integration-test.yml`**

```yaml
name: integration-test

on:
  push:
    branches:
      - main
  pull_request:
    branches:
      - main

jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout 🛎
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Load Firebase credentials
        run: |
          echo "FIREBASE_CREDENTIALS=$(jq -c . tests/firebase-credentials.json)" >> $GITHUB_ENV

      - name: Start services 🐳
        working-directory: ./tests
        run: docker compose up -d --build --wait

      - name: Wait for seed to complete
        working-directory: ./tests
        run: |
          echo "Waiting for seed container to finish..."
          docker compose wait seed || true
          sleep 2

      - name: Run integration tests 🧪
        working-directory: ./tests
        run: go test -v -timeout 120s ./...

      - name: Collect logs on failure 📋
        if: failure()
        working-directory: ./tests
        run: |
          docker compose logs api
          docker compose logs emulator

      - name: Stop services 🛑
        if: always()
        working-directory: ./tests
        run: docker compose down -v
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/integration-test.yml
git commit -m "ci: add integration test workflow"
```

---

### Task 12: Local End-to-End Verification

**Files:**

- None (verification only)

- [ ] **Step 1: Generate the fake Firebase credentials file**

```bash
cd tests
openssl genrsa 2048 > /tmp/test-key.pem
# Create firebase-credentials.json with the key (use a script or manually format)
```

- [ ] **Step 2: Build and start the stack**

```bash
cd tests
export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json)
docker compose up -d --build
```

- [ ] **Step 3: Wait for all services to be healthy**

```bash
docker compose ps
# All services should show "healthy" or "exited (0)" for seed
```

- [ ] **Step 4: Run the tests**

```bash
cd tests
go test -v -timeout 120s ./...
```

Expected: Both tests pass.

- [ ] **Step 5: Tear down**

```bash
docker compose down -v
```

- [ ] **Step 6: Push branch and create PR**

```bash
git push -u origin feature/integration-tests
gh pr create --title "feat: add integration test setup for API" --body "Adds E2E integration tests that validate the full SMS send/receive flow using Docker and a phone emulator."
```
