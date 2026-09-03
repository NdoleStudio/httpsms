# URL-backed Phone Notification Adapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow a phone whose existing `fcm_token` is an HTTPS URL to receive message and heartbeat wake-ups over HTTP while preserving the current scheduling, backpressure, outstanding-message, and status-event flows.

**Architecture:** Add transport helpers to `entities.Phone`, inject a map of existing `FCMClient` implementations keyed by transport, and let `PhoneNotificationService` select the client through map lookup. The HTTP path uses a standard OpenTelemetry-wrapped `http.Client`, application retries with `github.com/avast/retry-go/v5`, FCM-compatible JSON, and the existing notification success/failure state transitions.

**Tech Stack:** Go 1.25.8, Fiber v3, Firebase Admin Messaging, OpenTelemetry, `net/http`, `retry-go/v5`, Testify, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-09-02-url-backed-phone-notification-adapter-design.md`

## Final Accepted Architecture

This section supersedes later historical task text and code samples that
introduce an endpoint policy, DNS/IP filtering, custom dialing, or a
private-host allowlist.

- `Phone.NotificationTransport` performs only URL syntax, HTTPS scheme, and
  hostname classification; URL user information remains valid.
- `Container.NotificationHTTPClient` uses the existing
  `go-otelroundtripper` pattern with the `phone_notification_http` name and
  the default transport, matching the webhook client style without custom URL
  redaction.
- `HTTPNotificationSender` creates one reusable `*retry.Retrier` during
  initialization. It owns exactly three application attempts with exponential
  backoff, a fresh request/body, and a five-second child context per attempt;
  caller contexts are enforced per operation rather than stored on the
  reusable retrier.
- Payload encoding, request creation, one-attempt delivery, and retry
  configuration are separate focused methods. OpenTelemetry HTTP telemetry is
  owned by the injected round-tripper, with no sender-specific attempt metrics
  or spans.
- No endpoint DNS/IP/SSRF policy, custom notification transport, or private-host
  allowlist is part of the final implementation.
- `PhoneNotificationService` receives
  `map[entities.NotificationTransport]FCMClient`, determines transport once,
  selects the client by map lookup, sets the trimmed destination in
  `message.Token`, and passes the same pointer to `FCMClient.Send`. There is no
  extra sender interface, Firebase wrapper, or dispatcher class.
- `HTTPNotificationSender` implements `FCMClient` exactly and returns
  `http/success` after a successful callback response.
- Domain events continue to use the existing `EventDispatcher` directly.

## Global Constraints

- Reuse the existing `Phone.FcmToken` database field and `fcm_token` API field; add no transport or endpoint columns.
- A valid `https://` URL with a hostname selects HTTP; an opaque non-URL token selects Firebase.
- URL-like malformed or unsupported tokens are invalid and must never fall through to Firebase.
- Send both outstanding-message and heartbeat notifications through the selected transport.
- HTTP callback requests are unsigned and contain no message content, user API key, phone API key, or other credentials.
- HTTPS callback URLs may include standard URL user information.
- Accept any HTTP `2xx`; ignore response content.
- Make at most three HTTP attempts with a five-second timeout per attempt.
- Retry network failures, `408`, `429`, and `5xx`; do not retry other non-`2xx` responses.
- Use the standard OpenTelemetry-wrapped HTTP transport without destination
  DNS/IP filtering, custom dialing, or a private-host allowlist.
- Preserve existing schedules, per-minute backpressure, message expiration, send-attempt counting, outstanding-message fetching, and message event routes.
- Use `stacktrace.Propagate` or `stacktrace.Propagatef` for returned errors.
- Use GORM query builders with context propagation; this feature requires no database query changes or migration.
- Format Go code with `go-fumpt` through the repository's existing tooling.

---

## File Structure

### Create

- `api/pkg/entities/phone_test.go` - table-driven transport classification tests.
- `api/pkg/services/notification_endpoint_policy.go` - public HTTPS URL validation, reserved-IP rejection, and validated dialing.
- `api/pkg/services/notification_endpoint_policy_test.go` - deterministic resolver/dialer tests, including DNS rebinding protection.
- `api/pkg/services/fcm_client.go` - common phone notification client contract and Firebase client.
- `api/pkg/services/phone_notification_service.go` - message construction and transport-client lookup.
- `api/pkg/services/http_notification_sender.go` - HTTP request encoding, retry classification, and timeout.
- `api/pkg/services/http_notification_sender_test.go` - payload, retry, and response tests.
- `api/pkg/services/phone_notification_service_test.go` - message and heartbeat integration tests with hand-written fakes.
- `api/pkg/validators/phone_handler_validator_test.go` - URL token validation tests for both phone update routes.
- `tests/adapter-emulator/Dockerfile` - container image for the HTTPS adapter emulator.
- `tests/adapter-emulator/go.mod` - isolated emulator module.
- `tests/adapter-emulator/main.go` - HTTPS callback and HTTP control server startup.
- `tests/adapter-emulator/emulator.go` - gateway registry, deduplication, and callback records.
- `tests/adapter-emulator/api_client.go` - existing httpSMS phone API calls.
- `tests/adapter-emulator/notification_handler.go` - FCM-envelope message and heartbeat handling.
- `tests/adapter-emulator/control_handler.go` - test registration, incoming-message, and record endpoints.
- `tests/adapter_integration_test.go` - outgoing, incoming, and heartbeat end-to-end tests.
- `tests/generate-adapter-certificates.sh` - throwaway CA and server-certificate generation.

### Modify

- `api/pkg/entities/phone.go` - add `NotificationTransport`, `NotificationTransport()`, and `NotificationURL()`.
- `api/pkg/services/fcm_client.go` - keep the SDK wrapper; document its role as the low-level Firebase client.
- `api/pkg/services/phone_notification_service.go` - replace direct Firebase messages with neutral notifications and transport-aware failure text.
- `api/pkg/validators/phone_handler_validator.go` - inject and apply the endpoint policy for URL-like tokens.
- `api/pkg/di/container.go` - construct the HTTP client, transport-client map,
  and updated service dependencies.
- `api/pkg/requests/phone_update_request.go` - document dual-purpose `fcm_token`.
- `api/pkg/requests/phone_fcm_token_request.go` - document dual-purpose `fcm_token`.
- `api/pkg/entities/phone_notification.go` - update FCM-specific comments to transport-neutral wording.
- `api/docs/docs.go` - regenerate with `swag`.
- `api/docs/swagger.json` - regenerate with `swag`.
- `api/docs/swagger.yaml` - regenerate with `swag`.
- `tests/docker-compose.yml` - run the emulator and mount TLS material.
- `tests/.env.test` - allowlist the emulator hostname only in local mode.
- `tests/helpers_test.go` - adapter setup, control client, and internal-event helpers.
- `tests/README.md` - document emulator architecture and commands.
- `.github/workflows/api.yml` - generate adapter certificates before Docker startup.
- `.gitignore` - ignore generated adapter certificates.

---

### Task 1: Classify the Existing Token Field

**Files:**
- Modify: `api/pkg/entities/phone.go`
- Create: `api/pkg/entities/phone_test.go`

**Interfaces:**
- Consumes: `Phone.FcmToken *string`
- Produces:

```go
type NotificationTransport string

const (
    NotificationTransportFCM  NotificationTransport = "fcm"
    NotificationTransportHTTP NotificationTransport = "http"
)

func (phone *Phone) NotificationTransport() (NotificationTransport, error)
func (phone *Phone) NotificationURL() (*url.URL, error)
```

- [ ] **Step 1: Write failing transport-classification tests**

Create `api/pkg/entities/phone_test.go`:

```go
package entities

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func stringPointer(value string) *string {
    return &value
}

func TestPhoneNotificationTransport(t *testing.T) {
    tests := []struct {
        name      string
        token     *string
        transport NotificationTransport
        hasError  bool
    }{
        {name: "firebase token", token: stringPointer("fcm-token:value"), transport: NotificationTransportFCM},
        {name: "public https url", token: stringPointer("https://adapter.example.com/notify"), transport: NotificationTransportHTTP},
        {name: "missing token", token: nil, hasError: true},
        {name: "empty token", token: stringPointer("  "), hasError: true},
        {name: "http url", token: stringPointer("http://adapter.example.com/notify"), hasError: true},
        {name: "ftp url", token: stringPointer("ftp://adapter.example.com/notify"), hasError: true},
        {name: "missing host", token: stringPointer("https:///notify"), hasError: true},
        {name: "embedded credentials", token: stringPointer("https://user:pass@adapter.example.com/notify"), hasError: true},
        {name: "malformed url", token: stringPointer("https://[::1"), hasError: true},
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            phone := &Phone{FcmToken: test.token}

            transport, err := phone.NotificationTransport()

            if test.hasError {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, test.transport, transport)
        })
    }
}

func TestPhoneNotificationURL(t *testing.T) {
    phone := &Phone{FcmToken: stringPointer("https://adapter.example.com/notify?tenant=42")}

    endpoint, err := phone.NotificationURL()

    require.NoError(t, err)
    assert.Equal(t, "https", endpoint.Scheme)
    assert.Equal(t, "adapter.example.com", endpoint.Hostname())
    assert.Equal(t, "/notify", endpoint.Path)
    assert.Equal(t, "tenant=42", endpoint.RawQuery)
}

func TestPhoneNotificationURLRejectsFCMToken(t *testing.T) {
    phone := &Phone{FcmToken: stringPointer("fcm-token:value")}

    _, err := phone.NotificationURL()

    require.Error(t, err)
}
```

- [ ] **Step 2: Run the entity tests and confirm the new API is missing**

Run:

```bash
cd api
go test ./pkg/entities -run 'TestPhoneNotification' -count=1
```

Expected: compilation fails because `NotificationTransport`,
`NotificationTransportFCM`, `NotificationTransportHTTP`,
`Phone.NotificationTransport`, and `Phone.NotificationURL` do not exist.

- [ ] **Step 3: Implement token classification once on `Phone`**

Add imports for `fmt`, `net/url`, and `strings` in
`api/pkg/entities/phone.go`, then add:

```go
// NotificationTransport identifies how a phone receives wake-up notifications.
type NotificationTransport string

const (
    // NotificationTransportFCM sends notifications through Firebase.
    NotificationTransportFCM NotificationTransport = "fcm"
    // NotificationTransportHTTP sends notifications to a public HTTPS endpoint.
    NotificationTransportHTTP NotificationTransport = "http"
)

// NotificationTransport returns the transport encoded by FcmToken.
func (phone *Phone) NotificationTransport() (NotificationTransport, error) {
    if phone.FcmToken == nil || strings.TrimSpace(*phone.FcmToken) == "" {
        return "", fmt.Errorf("phone has no notification token")
    }

    token := strings.TrimSpace(*phone.FcmToken)
    endpoint, err := url.Parse(token)
    if err != nil {
        if strings.Contains(token, "://") {
            return "", fmt.Errorf("invalid notification URL: %w", err)
        }
        return NotificationTransportFCM, nil
    }

    if endpoint.Scheme == "" {
        return NotificationTransportFCM, nil
    }
    if endpoint.Scheme != "https" {
        return "", fmt.Errorf("notification URL must use https")
    }
    if endpoint.Hostname() == "" {
        return "", fmt.Errorf("notification URL must include a hostname")
    }
    return NotificationTransportHTTP, nil
}

// NotificationURL returns the parsed endpoint for an HTTP notification token.
func (phone *Phone) NotificationURL() (*url.URL, error) {
    transport, err := phone.NotificationTransport()
    if err != nil {
        return nil, err
    }
    if transport != NotificationTransportHTTP {
        return nil, fmt.Errorf("phone notification transport is [%s], not HTTP", transport)
    }

    endpoint, err := url.Parse(strings.TrimSpace(*phone.FcmToken))
    if err != nil {
        return nil, fmt.Errorf("cannot parse notification URL: %w", err)
    }
    return endpoint, nil
}
```

Before finishing this step, replace the plain `fmt.Errorf` wrappers with
`stacktrace.Propagatef` or `stacktrace.NewErrorf` to match repository error
conventions. Preserve the exact public method signatures above.

- [ ] **Step 4: Run the focused entity tests**

Run:

```bash
cd api
go test ./pkg/entities -run 'TestPhoneNotification' -count=1
```

Expected: PASS.

- [ ] **Step 5: Format and commit**

Run:

```bash
cd api
go-fumpt -w pkg/entities/phone.go pkg/entities/phone_test.go
git add pkg/entities/phone.go pkg/entities/phone_test.go
git commit -m "feat(api): classify phone notification tokens"
```

---

### Task 2: Enforce Public HTTPS Endpoint Policy

**Files:**
- Create: `api/pkg/services/notification_endpoint_policy.go`
- Create: `api/pkg/services/notification_endpoint_policy_test.go`

**Interfaces:**
- Consumes: parsed HTTPS endpoints from `Phone.NotificationURL()`
- Produces:

```go
type HostResolver interface {
    LookupNetIP(ctx context.Context, network string, host string) ([]netip.Addr, error)
}

type NotificationEndpointPolicy struct {
    resolver            HostResolver
    allowedPrivateHosts map[string]struct{}
}

func NewNotificationEndpointPolicy(resolver HostResolver, allowedPrivateHosts []string) *NotificationEndpointPolicy
func (policy *NotificationEndpointPolicy) Validate(ctx context.Context, endpoint *url.URL) ([]netip.Addr, error)
func (policy *NotificationEndpointPolicy) DialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error)
```

- [ ] **Step 1: Write failing policy tests with a deterministic resolver**

Create `api/pkg/services/notification_endpoint_policy_test.go` with:

```go
package services

import (
    "context"
    "net/netip"
    "net/url"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type staticHostResolver struct {
    addresses map[string][]netip.Addr
    err       error
}

func (resolver *staticHostResolver) LookupNetIP(_ context.Context, _ string, host string) ([]netip.Addr, error) {
    if resolver.err != nil {
        return nil, resolver.err
    }
    return resolver.addresses[host], nil
}

func TestNotificationEndpointPolicyValidate(t *testing.T) {
    tests := []struct {
        name      string
        rawURL    string
        addresses []netip.Addr
        hasError  bool
    }{
        {name: "public IPv4", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}},
        {name: "public IPv6", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("2606:4700:4700::1111")}},
        {name: "loopback", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}, hasError: true},
        {name: "private", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("10.0.0.5")}, hasError: true},
        {name: "link local", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("169.254.169.254")}, hasError: true},
        {name: "carrier grade NAT", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("100.64.0.1")}, hasError: true},
        {name: "documentation range", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("203.0.113.1")}, hasError: true},
        {name: "unique local IPv6", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("fd00::1")}, hasError: true},
        {name: "mixed public and private", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.5")}, hasError: true},
        {name: "embedded credentials", rawURL: "https://user:pass@adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}, hasError: true},
        {name: "insecure scheme", rawURL: "http://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}, hasError: true},
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            endpoint, err := url.Parse(test.rawURL)
            require.NoError(t, err)
            policy := NewNotificationEndpointPolicy(&staticHostResolver{
                addresses: map[string][]netip.Addr{endpoint.Hostname(): test.addresses},
            }, nil)

            addresses, err := policy.Validate(context.Background(), endpoint)

            if test.hasError {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, test.addresses, addresses)
        })
    }
}
```

Add a connection-time test whose resolver returns a public address during the
first `Validate` call and `127.0.0.1` during `DialContext`. Assert that the
recording dialer is never invoked. Implement the recording dialer as a local
function injected through an unexported `dialValidated` helper so the test does
not make a real network connection.

Add an exact-host allowlist test:

```go
func TestNotificationEndpointPolicyAllowsPrivateAddressForExactLocalHost(t *testing.T) {
    endpoint, err := url.Parse("https://adapter-emulator:9091/notifications/gateway-1")
    require.NoError(t, err)
    policy := NewNotificationEndpointPolicy(&staticHostResolver{
        addresses: map[string][]netip.Addr{
            "adapter-emulator": {netip.MustParseAddr("172.20.0.8")},
        },
    }, []string{"adapter-emulator"})

    addresses, err := policy.Validate(context.Background(), endpoint)

    require.NoError(t, err)
    assert.Equal(t, []netip.Addr{netip.MustParseAddr("172.20.0.8")}, addresses)
}
```

Also assert that `adapter-emulator.example.com`, a private IP-literal URL, and
any non-allowlisted private hostname remain rejected.

- [ ] **Step 2: Run the policy tests and confirm the types are missing**

Run:

```bash
cd api
go test ./pkg/services -run 'TestNotificationEndpointPolicy' -count=1
```

Expected: compilation fails because `NotificationEndpointPolicy` and its
constructor do not exist.

- [ ] **Step 3: Implement reserved-range checks**

Create `api/pkg/services/notification_endpoint_policy.go`. Define the resolver
interface above and these blocked prefixes:

```go
var blockedNotificationPrefixes = []netip.Prefix{
    netip.MustParsePrefix("0.0.0.0/8"),
    netip.MustParsePrefix("10.0.0.0/8"),
    netip.MustParsePrefix("100.64.0.0/10"),
    netip.MustParsePrefix("127.0.0.0/8"),
    netip.MustParsePrefix("169.254.0.0/16"),
    netip.MustParsePrefix("172.16.0.0/12"),
    netip.MustParsePrefix("192.0.0.0/24"),
    netip.MustParsePrefix("192.0.2.0/24"),
    netip.MustParsePrefix("192.168.0.0/16"),
    netip.MustParsePrefix("198.18.0.0/15"),
    netip.MustParsePrefix("198.51.100.0/24"),
    netip.MustParsePrefix("203.0.113.0/24"),
    netip.MustParsePrefix("224.0.0.0/4"),
    netip.MustParsePrefix("240.0.0.0/4"),
    netip.MustParsePrefix("::/128"),
    netip.MustParsePrefix("::1/128"),
    netip.MustParsePrefix("100::/64"),
    netip.MustParsePrefix("2001:db8::/32"),
    netip.MustParsePrefix("fc00::/7"),
    netip.MustParsePrefix("fe80::/10"),
    netip.MustParsePrefix("ff00::/8"),
}
```

Implement:

```go
func isPublicNotificationAddress(address netip.Addr) bool {
    address = address.Unmap()
    if !address.IsValid() || !address.IsGlobalUnicast() {
        return false
    }
    for _, prefix := range blockedNotificationPrefixes {
        if prefix.Contains(address) {
            return false
        }
    }
    return true
}
```

`Validate` must verify HTTPS, hostname presence, at least one DNS result, and
every resolved address passing
`isPublicNotificationAddress`. Private addresses are accepted only when the
lowercased hostname exactly matches `allowedPrivateHosts`; never wildcard or
suffix-match. Wrap resolver and validation errors with stacktrace context
without including the raw URL.

- [ ] **Step 4: Implement validated dialing**

Add:

```go
func (policy *NotificationEndpointPolicy) DialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
    return func(ctx context.Context, network string, address string) (net.Conn, error) {
        host, port, err := net.SplitHostPort(address)
        if err != nil {
            return nil, stacktrace.Propagatef(err, "cannot split notification endpoint address")
        }

        endpoint := &url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}
        addresses, err := policy.Validate(ctx, endpoint)
        if err != nil {
            return nil, stacktrace.Propagatef(err, "notification endpoint is not public")
        }

        var lastErr error
        for _, resolved := range addresses {
            connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.String(), port))
            if dialErr == nil {
                return connection, nil
            }
            lastErr = dialErr
        }
        return nil, stacktrace.Propagatef(lastErr, "cannot connect to notification endpoint")
    }
}
```

Factor the final connection loop through an unexported function variable or
method that accepts a dial function, allowing the DNS-rebinding test to assert
the selected address without opening a socket. Do not perform a normal second
hostname dial after validation.

- [ ] **Step 5: Run focused tests**

Run:

```bash
cd api
go test ./pkg/services -run 'TestNotificationEndpointPolicy' -count=1
```

Expected: PASS.

- [ ] **Step 6: Format and commit**

Run:

```bash
cd api
go-fumpt -w pkg/services/notification_endpoint_policy.go pkg/services/notification_endpoint_policy_test.go
git add pkg/services/notification_endpoint_policy.go pkg/services/notification_endpoint_policy_test.go
git commit -m "feat(api): validate adapter endpoints"
```

---

### Task 3: Add the Transport Client Map

**Files:**
- Modify: `api/pkg/services/fcm_client.go`
- Modify: `api/pkg/services/phone_notification_service.go`
- Modify: `api/pkg/services/phone_notification_service_test.go`

**Interfaces:**
- Consumes:

```go
func (phone *entities.Phone) NotificationTransport() (entities.NotificationTransport, error)
```

- Produces:

```go
type FCMClient interface {
    Send(context.Context, *messaging.Message) (string, error)
}
```

- [ ] **Step 1: Write failing transport-client map tests**

Create a recording client in `phone_notification_service_test.go`:

```go
type recordingPhoneNotificationClient struct {
    message *messaging.Message
    result  string
    err     error
    calls   int
}

func (client *recordingPhoneNotificationClient) Send(_ context.Context, message *messaging.Message) (string, error) {
    client.calls++
    client.message = message
    return client.result, client.err
}
```

Add tests for map-based FCM and HTTP selection, the same message pointer,
trimmed `message.Token`, absent transport entries, invalid tokens, and nil
messages.

- [ ] **Step 2: Run the service tests and confirm the map dependency is missing**

Run:

```bash
cd api
go test ./pkg/services -run 'TestPhoneNotificationService' -count=1
```

Expected: compilation fails because the service does not accept the client map.

- [ ] **Step 3: Implement focused map-based delivery**

Store `map[entities.NotificationTransport]FCMClient` on
`PhoneNotificationService` and implement:

```go
func (service *PhoneNotificationService) sendPhoneNotification(
    ctx context.Context,
    phone *entities.Phone,
    message *messaging.Message,
) (string, entities.NotificationTransport, error) {
    // Validate message, classify once, map lookup, set trimmed token, send.
}
```

Update comments in `api/pkg/services/fcm_client.go` to describe `FCMClient` as
the common phone notification transport boundary. Use the Firebase client
directly; do not add a wrapper or dispatcher.

- [ ] **Step 4: Run focused tests**

Run:

```bash
cd api
go test ./pkg/services -run 'TestPhoneNotificationService' -count=1
```

Expected: PASS.

- [ ] **Step 5: Format and commit**

Run:

```bash
cd api
go-fumpt -w pkg/services/fcm_client.go pkg/services/phone_notification_service.go pkg/services/phone_notification_service_test.go
git add pkg/services/fcm_client.go pkg/services/phone_notification_service.go pkg/services/phone_notification_service_test.go
git commit -m "refactor(api): dispatch gateway notifications"
```

---

### Task 4: Implement the SSRF-safe HTTP Sender

**Files:**
- Create: `api/pkg/services/http_notification_sender.go`
- Create: `api/pkg/services/http_notification_sender_test.go`

**Interfaces:**
- Consumes:

```go
type FCMClient interface {
    Send(context.Context, *messaging.Message) (string, error)
}
```

- Produces:

```go
type HTTPNotificationSender struct {
    logger  telemetry.Logger
    client  *http.Client
    retrier *retry.Retrier
    timeout time.Duration
}

func NewHTTPNotificationSender(
    logger telemetry.Logger,
    client *http.Client,
) *HTTPNotificationSender

func (sender *HTTPNotificationSender) Send(
    ctx context.Context,
    message *messaging.Message,
) (string, error)
```

- [ ] **Step 1: Write failing payload and success tests**

Create `api/pkg/services/http_notification_sender_test.go`. Use a custom
`roundTripFunc`:

```go
type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
    return roundTrip(request)
}
```

Construct the sender in tests with a reusable retrier configured through
`newHTTPNotificationRetrier(0)` so retry delay is zero from initialization.
Use a public address from the test resolver, and a client whose custom
RoundTripper does not dial.

Assert the request:

```go
assert.Equal(t, http.MethodPost, request.Method)
assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
```

Keep a test-only decoding struct and assert this structure:

```go
type httpNotificationRequest struct {
    Message struct {
        Token   string            `json:"token"`
        Data    map[string]string `json:"data"`
        Android struct {
            Priority string `json:"priority"`
            TTL      string `json:"ttl,omitempty"`
        } `json:"android"`
    } `json:"message"`
}
```

Return `204 No Content` and assert `Send` succeeds with result `http/success`.

- [ ] **Step 2: Write failing retry-classification tests**

Add table-driven tests for:

- network error then `202`: two calls, success;
- `408` then `200`: two calls, success;
- `429` then `204`: two calls, success;
- `500`, `502`, then `204`: three calls, success;
- `400`: one call, error;
- three `503` responses: three calls, error;
- redirects use Go's default client behavior; do not add terminal redirect
  handling;
- response with a body larger than the discard limit: success without reading
  unbounded content.

Assert retry attempts receive fresh requests and bodies.

- [ ] **Step 3: Run the HTTP sender tests and confirm the type is missing**

Run:

```bash
cd api
go test ./pkg/services -run 'TestHTTPNotificationSender' -count=1
```

Expected: compilation fails because `HTTPNotificationSender` does not exist.

- [ ] **Step 4: Implement the FCM-compatible HTTP payload**

Create `api/pkg/services/http_notification_sender.go` and build the one-use
payload directly with `map[string]any`:

```go
payload := map[string]any{
    "message": message,
}
```

Obtain the destination from `message.Token`. Rely on
`messaging.Message.MarshalJSON` and `messaging.AndroidConfig.MarshalJSON` for
the FCM shape and protobuf TTL, including `10*time.Minute` as `"600s"`. Build a
new request body for every attempt so retries never reuse a consumed reader.
Reject nil messages explicitly with a stacktrace-consistent error.

- [ ] **Step 5: Implement bounded retries**

Implement these helpers:

```go
func isRetryableNotificationStatus(statusCode int) bool {
    return statusCode == http.StatusRequestTimeout ||
        statusCode == http.StatusTooManyRequests ||
        statusCode >= http.StatusInternalServerError
}

func notificationRetryDelay(attempt uint) time.Duration {
    return time.Duration(1<<(attempt-1)) * 250 * time.Millisecond
}
```

Create one reusable retrier during sender initialization:

```go
func newHTTPNotificationRetrier(delay time.Duration) *retry.Retrier {
    return retry.New(
        retry.Attempts(3),
        retry.Delay(delay),
        retry.DelayType(retry.BackOffDelay),
        retry.LastErrorOnly(true),
        retry.RetryIf(isRetryableNotificationError),
    )
}
```

Do not configure this retrier with `retry.Context`: each `Send` has a different
caller context. Instead, check caller cancellation in the operation and return
a terminal error, while deriving each attempt's five-second context from that
caller.

Focused sender methods must:

1. parse `message.Token`;
2. marshal the request body once, then create a fresh reader per request;
3. create a child context with the configured timeout per attempt;
4. set `Content-Type`;
5. call `client.Do`;
6. close each response body after copying at most 4 KiB to `io.Discard`;
7. return `http/success` for any `2xx`;
8. retry only network errors, `408`, `429`, and `5xx` while attempts remain;
9. treat request construction, caller cancellation, and other statuses as
   terminal;
10. return a stacktrace-wrapped error.

Do not use the container's retrying HTTP client; retries belong in this sender
so status classification and attempt count are explicit.

- [ ] **Step 6: Add heartbeat tests**

Add a heartbeat payload test with `KEY_HEARTBEAT_ID`, high priority, and nil
TTL; assert the `ttl` field is omitted.
Use the standard HTTP telemetry and logging path without feature-specific URL
redaction.

- [ ] **Step 7: Run focused tests**

Run:

```bash
cd api
go test ./pkg/services -run 'TestHTTPNotificationSender' -count=1
```

Expected: PASS.

- [ ] **Step 8: Format and commit**

Run:

```bash
cd api
go-fumpt -w pkg/services/http_notification_sender.go pkg/services/http_notification_sender_test.go
git add pkg/services/http_notification_sender.go pkg/services/http_notification_sender_test.go
git commit -m "feat(api): send notifications to adapters"
```

---

### Task 5: Integrate Message and Heartbeat Notifications

**Files:**
- Modify: `api/pkg/services/phone_notification_service.go`
- Create: `api/pkg/services/phone_notification_service_test.go`
- Modify: `api/pkg/entities/phone_notification.go`

**Interfaces:**
- Consumes:

```go
map[entities.NotificationTransport]FCMClient
```

- Produces:

```go
type NotificationEventDispatcher interface {
    Dispatch(ctx context.Context, event cloudevents.Event) error
    DispatchWithTimeout(ctx context.Context, event cloudevents.Event, timeout time.Duration) (string, error)
}

func NewNotificationService(
    logger telemetry.Logger,
    tracer telemetry.Tracer,
    phoneNotificationClients map[entities.NotificationTransport]FCMClient,
    phoneRepository repositories.PhoneRepository,
    phoneNotificationRepository repositories.PhoneNotificationRepository,
    messageSendScheduleRepository repositories.MessageSendScheduleRepository,
    dispatcher NotificationEventDispatcher,
) *PhoneNotificationService
```

- [ ] **Step 1: Write failing service tests with hand-written fakes**

Create `api/pkg/services/phone_notification_service_test.go`.

Define repository fakes by embedding the interfaces and overriding only methods
used by the tests:

```go
type phoneNotificationPhoneRepository struct {
    repositories.PhoneRepository
    phone *entities.Phone
    err   error
}

func (repository *phoneNotificationPhoneRepository) LoadByID(
    _ context.Context,
    _ entities.UserID,
    _ uuid.UUID,
) (*entities.Phone, error) {
    return repository.phone, repository.err
}

type phoneNotificationRepository struct {
    repositories.PhoneNotificationRepository
    notificationID uuid.UUID
    status         entities.PhoneNotificationStatus
}

func (repository *phoneNotificationRepository) UpdateStatus(
    _ context.Context,
    notificationID uuid.UUID,
    status entities.PhoneNotificationStatus,
) error {
    repository.notificationID = notificationID
    repository.status = status
    return nil
}
```

Add a fake event dispatcher that records CloudEvents and returns no error. Add
recording `FCMClient` implementations through a transport-keyed map.

Test `Send` with an HTTPS token and assert:

- `KEY_MESSAGE_ID` equals `params.MessageID.String()`;
- priority is `normal`;
- TTL equals `phone.MessageExpirationDuration()`;
- the same `messaging.Message` pointer reaches the HTTP client;
- a `message.notification.sent` event is dispatched;
- phone-notification status becomes sent.

Test an HTTP sender error and assert:

- `message.notification.failed` is dispatched;
- status becomes failed;
- the payload error says the adapter endpoint could not be notified;
- the payload does not tell the user to reinstall Android.

Test an FCM sender error and assert the existing Android reinstallation guidance
is preserved.

Test `SendHeartbeatFCM` with an HTTPS token and assert:

- `KEY_HEARTBEAT_ID` parses as RFC3339;
- priority is `high`;
- TTL is nil;
- heartbeat sender errors are logged and return nil, preserving current
  heartbeat behavior.

Also test explicit nil-message handling and a missing transport map entry.

- [ ] **Step 2: Run the service tests and confirm the constructor mismatch**

Run:

```bash
cd api
go test ./pkg/services -run 'TestPhoneNotificationService' -count=1
```

Expected: compilation fails because `PhoneNotificationService` still consumes
`FCMClient` and a concrete `*EventDispatcher`.

- [ ] **Step 3: Replace direct Firebase message creation**

In `api/pkg/services/phone_notification_service.go`:

- construct Firebase `messaging.Message` values directly;
- replace the direct client with
  `phoneNotificationClients map[entities.NotificationTransport]FCMClient`;
- change the constructor to the exact signature above;
- change `eventDispatcher` to `NotificationEventDispatcher`.

Add a private `sendPhoneNotification` helper that validates the message,
classifies transport once, performs a map lookup, sets the trimmed token, calls
the selected client, and returns the transport with the result or error.

For message notifications, construct:

```go
ttl := phone.MessageExpirationDuration()
message := &messaging.Message{
    Data: map[string]string{
        "KEY_MESSAGE_ID": params.MessageID.String(),
    },
    Android: &messaging.AndroidConfig{
        Priority: "normal",
        TTL:      &ttl,
    },
}
```

For heartbeat notifications, call:

```go
message := &messaging.Message{
    Data: map[string]string{
        "KEY_HEARTBEAT_ID": time.Now().UTC().Format(time.RFC3339),
    },
    Android: &messaging.AndroidConfig{Priority: "high"},
}
```

- [ ] **Step 4: Add transport-aware failure text**

Use the transport returned by `sendPhoneNotification`; do not classify the
phone a second time.

For HTTP use:

```go
msg := fmt.Sprintf(
    "cannot notify the configured adapter for phone [%s]. Check the adapter URL and availability.",
    phone.PhoneNumber,
)
```

For Firebase preserve:

```go
msg := fmt.Sprintf(
    "cannot send notification to your phone [%s]. Reinstall the httpSMS app on your Android phone.",
    phone.PhoneNumber,
)
```

Log the technical wrapped error without logging the raw token. If transport
classification unexpectedly fails here, send that error through
`handleNotificationFailed` with a generic notification-configuration message.

- [ ] **Step 5: Update transport-specific comments**

In `api/pkg/entities/phone_notification.go`, change:

```go
// PhoneNotification represents an FCM notification to a mobile phone
```

to:

```go
// PhoneNotification represents a scheduled wake-up notification for a phone gateway.
```

Update `PhoneNotificationService` and `SendHeartbeatFCM` comments so they refer
to phone gateway notifications rather than only mobile phones. Keep the method
name `SendHeartbeatFCM` in this change to avoid an unrelated listener rename.

- [ ] **Step 6: Run focused tests**

Run:

```bash
cd api
go test ./pkg/services -run 'TestPhoneNotificationService|TestHTTPNotificationSender' -count=1
```

Expected: PASS.

- [ ] **Step 7: Format and commit**

Run:

```bash
cd api
go-fumpt -w pkg/services/phone_notification_service.go pkg/services/phone_notification_service_test.go pkg/entities/phone_notification.go
git add pkg/services/phone_notification_service.go pkg/services/phone_notification_service_test.go pkg/entities/phone_notification.go
git commit -m "feat(api): route phone gateway wake-ups"
```

---

### Task 6: Validate URL Tokens, Wire Dependencies, and Regenerate Swagger

**Files:**
- Modify: `api/pkg/validators/phone_handler_validator.go`
- Create: `api/pkg/validators/phone_handler_validator_test.go`
- Modify: `api/pkg/di/container.go`
- Modify: `api/pkg/requests/phone_update_request.go`
- Modify: `api/pkg/requests/phone_fcm_token_request.go`
- Modify: `api/docs/docs.go`
- Modify: `api/docs/swagger.json`
- Modify: `api/docs/swagger.yaml`

**Interfaces:**
- Consumes:

```go
func (container *Container) FCMClient() services.FCMClient
func NewHTTPNotificationSender(logger telemetry.Logger, client *http.Client) *HTTPNotificationSender
```

- Produces container factories:

```go
func (container *Container) NotificationHTTPClient() *http.Client
func (container *Container) PhoneNotificationClients() map[entities.NotificationTransport]services.FCMClient
```

- Produces validator constructor:

```go
func NewPhoneHandlerValidator(
    logger telemetry.Logger,
    tracer telemetry.Tracer,
    scheduleService *services.MessageSendScheduleService,
    endpointPolicy *services.NotificationEndpointPolicy,
) *PhoneHandlerValidator
```

- [ ] **Step 1: Write failing validator tests**

Create `api/pkg/validators/phone_handler_validator_test.go`.

Build the validator with a static public resolver and nil schedule service for
requests without a schedule ID. Add tests for both `ValidateUpsert` and
`ValidateFCMToken`:

```go
func TestPhoneHandlerValidatorAcceptsPublicHTTPSNotificationURL(t *testing.T) {
    validator := newPhoneHandlerValidatorWithAddresses(map[string][]netip.Addr{
        "adapter.example.com": {netip.MustParseAddr("8.8.8.8")},
    })

    errors := validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
        PhoneNumber: "+18005550199",
        FcmToken:    "https://adapter.example.com/notify",
        SIM:         entities.SIM1.String(),
    })

    assert.Empty(t, errors)
}
```

Add rejection tests for `http://`, loopback resolution, private resolution,
mixed public/private resolution, and malformed HTTPS.
Add an opaque FCM token test to prove the resolver is not required for Firebase
tokens.

- [ ] **Step 2: Run validator tests and confirm unsafe URLs are accepted**

Run:

```bash
cd api
go test ./pkg/validators -run 'TestPhoneHandlerValidator.*Notification' -count=1
```

Expected: tests fail because the validator only checks token length.

- [ ] **Step 3: Inject and apply endpoint policy**

Add `endpointPolicy *services.NotificationEndpointPolicy` to
`PhoneHandlerValidator` and its constructor.

Add:

```go
func (validator *PhoneHandlerValidator) validateNotificationToken(
    ctx context.Context,
    token string,
    result url.Values,
) {
    token = strings.TrimSpace(token)
    if token == "" {
        return
    }

    phone := &entities.Phone{FcmToken: &token}
    transport, err := phone.NotificationTransport()
    if err != nil {
        result.Add("fcm_token", err.Error())
        return
    }
    if transport != entities.NotificationTransportHTTP {
        return
    }

    endpoint, err := phone.NotificationURL()
    if err != nil {
        result.Add("fcm_token", err.Error())
        return
    }
    if _, err = validator.endpointPolicy.Validate(ctx, endpoint); err != nil {
        result.Add("fcm_token", "fcm_token must be a public HTTPS adapter URL")
    }
}
```

Call it after structural validation succeeds in both `ValidateUpsert` and
`ValidateFCMToken`. Change `ValidateFCMToken` to use its context argument.

- [ ] **Step 4: Add SSRF-safe container factories**

In `api/pkg/di/container.go`, add:

```go
func (container *Container) NotificationEndpointPolicy() *services.NotificationEndpointPolicy {
    if container.notificationEndpointPolicy != nil {
        return container.notificationEndpointPolicy
    }

    allowedPrivateHosts := []string{} // Superseded: no private-host allowlist is configured.
    container.notificationEndpointPolicy = services.NewNotificationEndpointPolicy(
        net.DefaultResolver,
        allowedPrivateHosts,
    )
    return container.notificationEndpointPolicy
}
```

Add a client factory:

```go
func (container *Container) NotificationHTTPClient() *http.Client {
    policy := container.NotificationEndpointPolicy()
    transport := &http.Transport{
        Proxy: nil,
        DialContext: policy.DialContext(&net.Dialer{
            Timeout:   5 * time.Second,
            KeepAlive: 30 * time.Second,
        }),
        ForceAttemptHTTP2: true,
        TLSClientConfig: &tls.Config{
            MinVersion: tls.VersionTLS12,
        },
    }

    return &http.Client{
        Transport: otelroundtripper.New(
            otelroundtripper.WithName("phone_notification_http"),
            otelroundtripper.WithParent(transport),
            otelroundtripper.WithMeter(otel.GetMeterProvider().Meter(container.projectID)),
        ),
    }
}
```

Do not set a client-wide timeout; the sender creates the approved five-second
context for each attempt.

Add:

```go
func (container *Container) PhoneNotificationClients() map[entities.NotificationTransport]services.FCMClient {
    return map[entities.NotificationTransport]services.FCMClient{
        entities.NotificationTransportFCM: container.FCMClient(),
        entities.NotificationTransportHTTP: services.NewHTTPNotificationSender(
            container.Logger(),
            container.NotificationHTTPClient(),
        ),
    }
}
```

Add this field to `Container`:

```go
notificationEndpointPolicy *services.NotificationEndpointPolicy
```

The cached policy ensures validation and connection-time checks use the same
allowlist. Do not cache per-request sender state.

- [ ] **Step 5: Wire service and validator constructors**

Change `container.NotificationService()` to pass
`container.PhoneNotificationClients()`.

Find `container.PhoneHandlerValidator()` and pass
`container.NotificationEndpointPolicy()` as its fourth argument. Keep existing
logger, tracer, and schedule-service arguments unchanged.

- [ ] **Step 6: Run focused package tests**

Run:

```bash
cd api
go test ./pkg/entities ./pkg/services ./pkg/validators ./pkg/di -count=1
```

Expected: PASS.

- [ ] **Step 7: Update API descriptions**

In both phone request structs, replace the generic FCM token comment with:

```go
// FcmToken is either a Firebase registration token or a public HTTPS adapter callback URL.
FcmToken string `json:"fcm_token" example:"https://adapter.example.com/notifications"`
```

Update handler Swagger descriptions for phone upsert and FCM-token upsert to
state that URL-backed phones receive FCM-compatible HTTP wake-ups. Do not add or
rename routes or JSON fields.

- [ ] **Step 8: Regenerate Swagger**

Run:

```bash
cd api
swag init --requiredByDefault --parseDependency --parseInternal
```

Expected: `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml` update
with the dual-purpose `fcm_token` descriptions.

- [ ] **Step 9: Run the complete API test suite**

Run:

```bash
cd api
go test ./...
```

Expected: PASS.

- [ ] **Step 10: Build the API**

Run:

```bash
cd api
go build -o ./tmp/main.exe .
```

Expected: build succeeds.

- [ ] **Step 11: Inspect the final diff for forbidden changes**

Run:

```bash
git diff --check
git diff --stat
git grep -n "fcm_token" -- api/pkg/entities/phone.go api/pkg/requests/phone_update_request.go api/pkg/requests/phone_fcm_token_request.go
```

Confirm:

- no database field or migration was added;
- `fcm_token` remains the persisted/API field;
- no message content or API key is added to the HTTP callback payload;
- scheduling and repository code are unchanged;

- [ ] **Step 12: Format and commit**

Run:

```bash
cd api
go-fumpt -w pkg/validators/phone_handler_validator.go pkg/validators/phone_handler_validator_test.go pkg/di/container.go pkg/requests/phone_update_request.go pkg/requests/phone_fcm_token_request.go
git add pkg/validators/phone_handler_validator.go pkg/validators/phone_handler_validator_test.go pkg/di/container.go pkg/requests/phone_update_request.go pkg/requests/phone_fcm_token_request.go pkg/handlers/phone_handler.go docs/docs.go docs/swagger.json docs/swagger.yaml
git commit -m "feat(api): enable URL-backed phone gateways"
```

---

### Task 7: Build the Adapter Emulator and End-to-End Scenarios

**Files:**
- Create: `tests/adapter-emulator/Dockerfile`
- Create: `tests/adapter-emulator/go.mod`
- Create: `tests/adapter-emulator/main.go`
- Create: `tests/adapter-emulator/emulator.go`
- Create: `tests/adapter-emulator/api_client.go`
- Create: `tests/adapter-emulator/notification_handler.go`
- Create: `tests/adapter-emulator/control_handler.go`
- Create: `tests/adapter_integration_test.go`
- Create: `tests/generate-adapter-certificates.sh`
- Modify: `tests/docker-compose.yml`
- Modify: `tests/.env.test`
- Modify: `tests/helpers_test.go`
- Modify: `tests/README.md`
- Modify: `.github/workflows/api.yml`
- Modify: `.gitignore`

**Interfaces:**
- Emulator callback: `POST https://adapter-emulator:9091/notifications/{gatewayID}`
- Emulator control:

```text
PUT  http://localhost:9092/test/gateways/{gatewayID}
POST http://localhost:9092/test/gateways/{gatewayID}/incoming
GET  http://localhost:9092/test/gateways/{gatewayID}/notifications
GET  http://localhost:9092/health
```

- Gateway registration:

```go
type gatewayRegistration struct {
    PhoneNumber string `json:"phone_number"`
    PhoneAPIKey string `json:"phone_api_key"`
}
```

- Incoming control payload:

```go
type incomingMessageRequest struct {
    Contact   string `json:"contact"`
    Content   string `json:"content"`
    Encrypted bool   `json:"encrypted"`
}
```

- Callback record:

```go
type notificationRecord struct {
    GatewayID string            `json:"gateway_id"`
    Data      map[string]string `json:"data"`
    MessageID string            `json:"message_id,omitempty"`
    Kind      string            `json:"kind"`
    Processed bool              `json:"processed"`
    Error     string            `json:"error,omitempty"`
}
```

- [ ] **Step 1: Generate throwaway TLS material**

Create `tests/generate-adapter-certificates.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

output_dir="${1:-certs}"
mkdir -p "$output_dir"

openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout "$output_dir/ca-key.pem" \
  -out "$output_dir/ca.pem" \
  -days 2 \
  -subj "/CN=httpSMS integration adapter CA"

openssl req -newkey rsa:2048 -nodes \
  -keyout "$output_dir/server-key.pem" \
  -out "$output_dir/server.csr" \
  -subj "/CN=adapter-emulator"

cat >"$output_dir/server.ext" <<'EOF'
subjectAltName=DNS:adapter-emulator
extendedKeyUsage=serverAuth
EOF

openssl x509 -req \
  -in "$output_dir/server.csr" \
  -CA "$output_dir/ca.pem" \
  -CAkey "$output_dir/ca-key.pem" \
  -CAcreateserial \
  -out "$output_dir/server.pem" \
  -days 2 \
  -extfile "$output_dir/server.ext"
```

Add `tests/certs/` to `.gitignore`. Do not commit generated keys or
certificates.

- [ ] **Step 2: Scaffold the isolated emulator module**

Create `tests/adapter-emulator/go.mod`:

```go
module github.com/NdoleStudio/httpsms/tests/adapter-emulator

go 1.25.0
```

Use only the standard library. Run:

```bash
cd tests/adapter-emulator
go mod tidy
```

Expected: the command succeeds without adding dependencies.

- [ ] **Step 3: Implement emulator state and deduplication**

In `tests/adapter-emulator/emulator.go`, implement:

```go
type gateway struct {
    PhoneNumber string
    PhoneAPIKey string
}

type emulator struct {
    apiBaseURL string
    client     *http.Client
    mu         sync.RWMutex
    gateways   map[string]gateway
    records    []*notificationRecord
}

func newEmulator(apiBaseURL string, client *http.Client) *emulator {
    return &emulator{
        apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
        client:     client,
        gateways:   make(map[string]gateway),
    }
}
```

Add locked methods to register a gateway, load a gateway, record each callback,
mark it processed/failed, and list copied records for one gateway.

- [ ] **Step 4: Implement existing phone API calls**

In `tests/adapter-emulator/api_client.go`, implement:

```go
func (emulator *emulator) fetchOutstanding(
    ctx context.Context,
    gateway gateway,
    messageID string,
) (map[string]any, error)

func (emulator *emulator) fireMessageEvent(
    ctx context.Context,
    gateway gateway,
    messageID string,
    eventName string,
) error

func (emulator *emulator) receiveMessage(
    ctx context.Context,
    gateway gateway,
    request incomingMessageRequest,
) (map[string]any, error)

func (emulator *emulator) storeHeartbeat(
    ctx context.Context,
    gateway gateway,
) error
```

Use the existing routes:

```text
GET  /v1/messages/outstanding?message_id={messageID}
POST /v1/messages/{messageID}/events
POST /v1/messages/receive
POST /v1/heartbeats
```

Every request sets `x-api-key`. Message events use `SENT` then `DELIVERED` with
UTC RFC3339 timestamps. Incoming messages use the gateway phone number as `to`,
the control request contact as `from`, `SIM1`, and the requested encryption
flag. Return contextual `fmt.Errorf` errors because the emulator module
intentionally does not depend on the production API module.

- [ ] **Step 5: Implement FCM-compatible callback handling**

In `tests/adapter-emulator/notification_handler.go`, decode:

```go
type callbackEnvelope struct {
    Message struct {
        Token string            `json:"token"`
        Data  map[string]string `json:"data"`
    } `json:"message"`
}
```

The handler must:

1. load `gatewayID` from the route;
2. return `404` for unknown gateways;
3. return `400` for unsupported data;
4. record every callback;
5. for `KEY_MESSAGE_ID`, fetch outstanding, fire `SENT`, fire `DELIVERED`, and
   mark the record processed with kind `message`;
6. for `KEY_HEARTBEAT_ID`, store a heartbeat and mark the record processed with
   kind `heartbeat`;
7. return `500` and retain the error string in the record when processing
   fails, allowing the API sender's retry behavior to be exercised.

Processing may be synchronous because the API accepts any `2xx` and the
integration stack controls response time.

- [ ] **Step 6: Implement control and server endpoints**

In `tests/adapter-emulator/control_handler.go`, implement registration,
incoming-message, record listing, and health handlers using `http.ServeMux`.

In `main.go`, read:

```text
API_BASE_URL=http://api:8000
ADAPTER_TLS_CERT=/certs/server.pem
ADAPTER_TLS_KEY=/certs/server-key.pem
```

Start:

- HTTPS callback server on `:9091`;
- HTTP control server on `:9092`.

Use `http.Server` with finite `ReadHeaderTimeout`, `ReadTimeout`,
`WriteTimeout`, and `IdleTimeout`. Shut both servers down on SIGINT/SIGTERM.

- [ ] **Step 7: Add the emulator container**

Create `tests/adapter-emulator/Dockerfile`:

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/adapter-emulator .

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/adapter-emulator /usr/local/bin/adapter-emulator
ENTRYPOINT ["adapter-emulator"]
```

In `tests/docker-compose.yml`, add `adapter-emulator`:

```yaml
adapter-emulator:
  build:
    context: ./adapter-emulator
  ports:
    - "9092:9092"
  environment:
    API_BASE_URL: http://api:8000
    ADAPTER_TLS_CERT: /certs/server.pem
    ADAPTER_TLS_KEY: /certs/server-key.pem
  volumes:
    - ./certs:/certs:ro
  healthcheck:
    test: ["CMD", "wget", "-qO-", "http://localhost:9092/health"]
    interval: 5s
    timeout: 5s
    retries: 10
```

Make `api` depend on the healthy emulator. Mount `./certs/ca.pem` into the API
container and set:

```yaml
SSL_CERT_FILE: /adapter-certs/ca.pem
```

Do not add a private-host allowlist; Docker DNS is used through the standard Go
HTTP transport. Keep `SSL_CERT_FILE` so the emulator certificate remains
trusted.

- [ ] **Step 8: Add adapter test helpers**

In `tests/helpers_test.go`, add `adapterControlURL`,
`systemAPIKey`, and:

```go
type adapterTestPhone struct {
    testPhone
    PhoneID   string
    GatewayID string
}

func setupAdapterPhone(ctx context.Context, t *testing.T, messagesPerMinute uint) adapterTestPhone
func dispatchInternalEvent(ctx context.Context, t *testing.T, event map[string]any)
func waitForAdapterMessageRecords(t *testing.T, gatewayID string, messageID string, timeout time.Duration) []notificationRecord
func waitForAdapterHeartbeatRecord(t *testing.T, gatewayID string, timeout time.Duration) notificationRecord
func triggerAdapterIncoming(ctx context.Context, t *testing.T, phone adapterTestPhone, contact string, content string) string
```

`setupAdapterPhone` must:

1. generate a gateway UUID and phone number;
2. create a phone API key;
3. register the gateway with the emulator control API;
4. use callback URL
   `https://adapter-emulator:9091/notifications/{gatewayID}`;
5. upsert the phone through the user API and capture its phone ID;
6. bind the same callback through the phone API-key route;
7. wait for phone authorization using the existing helper.

`dispatchInternalEvent` posts a valid CloudEvent JSON body to `/v1/events` with
`x-api-key: system-user-api-key`.

- [ ] **Step 9: Write the outgoing adapter integration test**

Create `tests/adapter_integration_test.go`:

```go
func TestAdapterGatewayOutgoingMessage(t *testing.T) {
    ctx := context.Background()
    phone := setupAdapterPhone(ctx, t, 60)
    contact := randomPhoneNumber()
    content := "Adapter outgoing " + randomEncryptionKey()

    response, httpResponse, err := newAPIClient().Messages.Send(ctx, &httpsms.MessageSendParams{
        From:    phone.PhoneNumber,
        To:      contact,
        Content: content,
    })
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, httpResponse.HTTPResponse.StatusCode)

    messageID := response.Data.ID.String()
    message := pollMessageStatus(ctx, t, messageID, "delivered", 30*time.Second)

    assert.Equal(t, phone.PhoneNumber, message.Owner)
    assert.Equal(t, contact, message.Contact)
    assert.Equal(t, content, message.Content)
    records := waitForAdapterMessageRecords(t, phone.GatewayID, messageID, 30*time.Second)
    require.Len(t, records, 1)
    assert.Equal(t, "message", records[0].Kind)
    assert.True(t, records[0].Processed)
    assert.Equal(t, messageID, records[0].Data["KEY_MESSAGE_ID"])
}
```

The record-list helper queries by message ID. Assert one processed record for
the successful callback flow.

- [ ] **Step 10: Write the incoming adapter integration test**

Add:

```go
func TestAdapterGatewayIncomingMessage(t *testing.T) {
    ctx := context.Background()
    phone := setupAdapterPhone(ctx, t, 60)
    contact := randomPhoneNumber()
    content := "Adapter incoming " + randomEncryptionKey()

    messageID := triggerAdapterIncoming(ctx, t, phone, contact, content)
    message := pollMessageStatus(ctx, t, messageID, "received", 15*time.Second)

    assert.Equal(t, phone.PhoneNumber, message.Owner)
    assert.Equal(t, contact, message.Contact)
    assert.Equal(t, content, message.Content)
    assert.Equal(t, "received", message.Status)
}
```

This test must call the emulator control endpoint; it must not post
`/v1/messages/receive` directly from the test runner.

- [ ] **Step 11: Write the heartbeat callback integration test**

Add:

```go
func TestAdapterGatewayHeartbeatWakeUp(t *testing.T) {
    ctx := context.Background()
    phone := setupAdapterPhone(ctx, t, 60)
    monitorID := uuid.NewString()

    dispatchInternalEvent(ctx, t, map[string]any{
        "specversion":     "1.0",
        "id":              uuid.NewString(),
        "source":          "/tests/adapter-emulator",
        "type":            "phone.heartbeat.missed",
        "time":            time.Now().UTC().Format(time.RFC3339),
        "datacontenttype": "application/json",
        "data": map[string]any{
            "phone_id":                 phone.PhoneID,
            "user_id":                  "test-user-id",
            "last_heartbeat_timestamp": time.Now().UTC().Add(-20 * time.Minute).Format(time.RFC3339),
            "timestamp":                time.Now().UTC().Format(time.RFC3339),
            "monitor_id":               monitorID,
            "owner":                    phone.PhoneNumber,
        },
    })

    record := waitForAdapterHeartbeatRecord(t, phone.GatewayID, 30*time.Second)
    assert.Equal(t, "heartbeat", record.Kind)
    assert.NotEmpty(t, record.Data["KEY_HEARTBEAT_ID"])

    heartbeats, response, err := newAPIClient().Heartbeats.Index(ctx, &httpsms.HeartbeatIndexParams{
        Owner: phone.PhoneNumber,
        Limit: 1,
    })
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
    require.NotEmpty(t, heartbeats.Data)
    assert.Equal(t, phone.PhoneNumber, heartbeats.Data[0].Owner)
}
```

The direct internal event avoids waiting for the production 16-minute monitor
interval while still exercising `PhoneNotificationListener`,
`PhoneNotificationService`, the HTTP client, the emulator callback, and the
existing heartbeat API.

- [ ] **Step 12: Update local and CI commands**

In `.github/workflows/api.yml`, after Firebase credential generation, add:

```yaml
- name: Generate adapter certificates
  run: bash tests/generate-adapter-certificates.sh tests/certs
```

Update `tests/README.md` architecture, project tree, coverage checklist,
troubleshooting logs, and setup commands. The documented local sequence must
run both credential scripts before `docker compose up`.

- [ ] **Step 13: Run the complete integration suite**

Run:

```bash
cd tests
bash generate-firebase-credentials.sh firebase-credentials.json
bash generate-adapter-certificates.sh certs
docker compose up -d --build --wait
docker compose wait seed
go test -v -timeout 300s ./...
docker compose down -v
```

Expected: existing FCM/WireMock tests and all three adapter tests pass.

- [ ] **Step 14: Inspect failure logs if a scenario times out**

Run:

```bash
cd tests
docker compose logs --tail 200 api adapter-emulator
```

The outgoing logs must show callback receipt, outstanding fetch, `SENT`, and
`DELIVERED`. Incoming logs must show the emulator calling the receive route.
Heartbeat logs must show `KEY_HEARTBEAT_ID` and a successful heartbeat POST.

- [ ] **Step 15: Commit integration coverage**

Run:

```bash
git add .gitignore .github/workflows/api.yml tests
git commit -m "test(api): cover URL-backed phone gateways"
```
