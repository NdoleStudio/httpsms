# URL-backed Phone Notification Adapter

- Date: 2026-09-02
- Status: Approved (design)
- Scope: `api/` Go backend plus `tests/` integration infrastructure and API CI.
  Web and Android clients are unchanged.

## Problem

httpSMS currently wakes an Android phone through Firebase Cloud Messaging when
an outgoing message is ready. The phone then fetches the outstanding message,
sends it, and reports sent, delivered, or failed events through the phone API.

Users should be able to register a non-Android gateway, such as a WhatsApp
adapter, as a phone number. The gateway must reuse the existing message queue,
send schedules, per-phone backpressure, expiration, retry, incoming-message,
and status-event flows. The only behavioral difference is how httpSMS wakes the
gateway: when the stored `fcm_token` is a URL, httpSMS calls that URL instead of
Firebase.

Customer-controlled callback URLs create additional security and delivery
requirements:

- callbacks are wake-up hints, not proof that a message was sent;
- callback delivery is at least once and must have an idempotency identity;
- transient endpoint failures need bounded retries;
- existing Android registrations must retain their current behavior.

## Decisions

- Reuse the existing `Phone.FcmToken` database field and `fcm_token` API field.
  Do not add transport or callback URL columns.
- Determine transport through helper methods on `Phone`; callers do not inspect
  or parse `FcmToken` directly.
- A valid `https://` URL with a hostname selects HTTP delivery. Non-URL tokens
  select Firebase. URL-like but malformed or unsupported values are rejected.
- Use `PhoneNotificationDispatcher` with separate Firebase and HTTP senders;
  domain events continue to use the existing `EventDispatcher`.
- Send both outstanding-message and heartbeat notifications to URL-backed
  phones.
- POST an FCM-compatible JSON envelope to adapter endpoints.
- Treat any `2xx` response as successful wake-up acceptance and ignore its
  body.
- Make up to three total HTTP attempts with `retry-go/v5` and a five-second
  timeout per attempt.
- Retry network failures, HTTP `408`, HTTP `429`, and `5xx` responses. Other
  non-`2xx` responses fail immediately.
- After callback retries are exhausted, use the current notification failure
  path and mark the message failed.
- Do not sign or authenticate callback requests. The payload contains no
  message content or API credentials.
- Use a standard OpenTelemetry-wrapped `http.Client` without endpoint DNS/IP
  filtering, custom dialing, or a private-host allowlist.
- Preserve all existing phone API-key authorization and message-processing
  behavior.

## Existing Flow

The current backend already isolates scheduling from phone wake-up delivery:

1. `MessageService.SendMessage` creates the outgoing message event using the
   phone's configured send attempts, SIM, and rate settings.
2. `PhoneNotificationService.Schedule` persists a `PhoneNotification` and uses
   `PhoneNotificationRepository.Schedule` or `ScheduleExact` to apply
   per-minute limits and send schedules.
3. `message.notification.send` invokes `PhoneNotificationService.Send`.
4. The service sends an FCM data notification containing `KEY_MESSAGE_ID`.
5. A successful notification increments the message send-attempt count and
   schedules the existing expiration check.
6. The Android phone fetches
   `GET /v1/messages/outstanding?message_id=<id>` using a phone API key scoped
   to its number.
7. The phone reports sent, delivered, or failed events and submits inbound
   messages through the existing phone API routes.

The adapter feature changes step 4 only. The same dispatcher also applies to
the heartbeat notification currently sent by `SendHeartbeatFCM`.

## Design

### 1. Phone transport helpers

Add helpers to `api/pkg/entities/phone.go` that classify and expose the
notification destination while continuing to store only `FcmToken`.

The helpers provide three outcomes:

- **Firebase:** the token has no URL syntax and is passed unchanged to Firebase.
- **HTTP:** the token is an absolute, syntactically valid `https://` URL.
- **Invalid:** the value is URL-like but malformed, uses another scheme, or has
  no hostname.

Use these entity-level types and methods:

```go
type NotificationTransport string

const (
    NotificationTransportFCM  NotificationTransport = "fcm"
    NotificationTransportHTTP NotificationTransport = "http"
)

func (phone *Phone) NotificationTransport() (NotificationTransport, error)
func (phone *Phone) NotificationURL() (*url.URL, error)
```

`NotificationTransport` returns an error for a missing token or a URL-like
invalid token. `NotificationURL` succeeds only for the HTTP transport.
`PhoneNotificationService`, validation, and tests must not duplicate
string-prefix checks.

A token is considered URL-like when it declares a URI scheme. This prevents an
invalid `http://`, `ftp://`, or malformed HTTPS endpoint from falling through
to Firebase as if it were an FCM token. Ordinary FCM tokens remain opaque.

### 2. Shared Firebase message contract

Use Firebase's `messaging.Message` as the notification contract for both
transports instead of maintaining a duplicate DTO:

```go
type NotificationSender interface {
    Send(
        ctx context.Context,
        message *messaging.Message,
        notificationID uuid.UUID,
    ) (string, error)
}
```

`PhoneNotificationService` constructs `messaging.Message` directly with the
data and Android configuration. The notification ID remains a separate
argument: it is the persisted `PhoneNotification.ID` for outgoing messages,
while heartbeats generate a request UUID for delivery identity.

Add a dispatcher that:

1. uses the phone helper to determine the transport;
2. rejects a nil message with a stacktrace error;
3. trims the phone's configured token and assigns it to `message.Token`;
4. delegates the same message pointer and notification ID to the selected
   sender;
5. returns a transport-neutral delivery result string or an error.

The result string is used only by existing notification event bookkeeping.
Firebase keeps the message name returned by the SDK. HTTP delivery uses a
generated identifier that does not expose the callback URL.

### 3. Firebase sender

Adapt the existing `FCMClient` behind the transport-neutral sender interface.
`FCMNotificationSender` validates that the message is non-nil and passes the
same `*messaging.Message` directly to `FCMClient.Send` without reconstructing
the payload.

The production Firebase client and emulator client remain available. Android
tokens follow the same SDK path, payload keys, priorities, TTL values, success
events, and failure events as before.

### 4. HTTP sender and request contract

The HTTP sender posts `Content-Type: application/json` with an FCM-compatible
envelope:

```json
{
  "message": {
    "token": "https://adapter.example.com/notifications",
    "data": {
      "KEY_MESSAGE_ID": "32343a19-da5e-4b1b-a767-3298a73703cb"
    },
    "android": {
      "priority": "normal",
      "ttl": "600s"
    }
  }
}
```

Heartbeat notifications use the same shape with:

```json
{
  "data": {
    "KEY_HEARTBEAT_ID": "2026-09-02T19:22:20Z"
  },
  "android": {
    "priority": "high"
  }
}
```

Adapters should depend on `message.data`; the Android object exists for payload
compatibility and communicates priority and expiration hints.

`HTTPNotificationSender` obtains the destination from `message.Token` and
marshals `map[string]any{"message": message}`. Firebase's
`messaging.Message` and `messaging.AndroidConfig.MarshalJSON` own the FCM JSON
shape and protobuf duration formatting, including a ten-minute TTL as `600s`.

Every request includes:

```text
X-httpSMS-Notification-ID: <delivery UUID>
Content-Type: application/json
```

For an outgoing message, the header value is the persisted phone-notification
ID. Retries of the same HTTP delivery reuse that ID. If the normal message
expiration flow schedules another send attempt, it creates a new
`PhoneNotification` and therefore a new ID. The adapter can distinguish a
duplicate HTTP request from an intentional later send attempt.

The response contract is deliberately small:

- any `2xx` status means the adapter accepted the wake-up;
- response headers and body do not affect message state;
- the sender reads at most a small bounded amount needed to safely reuse or
  close the connection, then discards the body.

The callback is not sent the message content, phone API key, user ID, or
credentials.

### 5. HTTP retry and state semantics

Each HTTP delivery makes no more than three total attempts. Each attempt has a
five-second context timeout and uses short exponential backoff.

Retryable failures are:

- connection, DNS, TLS, and timeout failures;
- HTTP `408 Request Timeout`;
- HTTP `429 Too Many Requests`;
- HTTP `5xx`.

Other non-`2xx` statuses are terminal for that notification and are not
retried. Ignore `Retry-After`; use the sender's bounded exponential backoff so
a customer response cannot extend the listener into an unbounded worker.

When a callback returns `2xx`, `PhoneNotificationService` uses its existing
success path:

1. dispatch `message.notification.sent`;
2. set the `PhoneNotification` status to sent;
3. increment the message send-attempt count;
4. schedule the configured message expiration check.

This means callback success is only proof that the adapter was notified. The
message remains pending/scheduled until the adapter fetches it, at which point
the existing `message.phone.sending` path runs.

When callback delivery reaches a terminal failure or exhausts retries,
`PhoneNotificationService` uses its existing failed-notification path:

1. dispatch `message.notification.failed`;
2. set the `PhoneNotification` status to failed;
3. store a failed message event through the existing message listener.

The HTTP-specific error message tells the user that the configured adapter
endpoint could not be notified. It must not reuse the current Android
reinstallation guidance.

### 6. At-least-once delivery and adapter idempotency

HTTP wake-up delivery is at least once. A request may reach the adapter even if
httpSMS observes a timeout or connection failure while receiving the response.
The retry then delivers the same notification ID again.

Adapters must:

- deduplicate callback requests by `X-httpSMS-Notification-ID`;
- treat callbacks as hints to fetch work, not as message content;
- avoid sending the external message twice for the same notification ID;
- retain their own provider-level idempotency and reconciliation where the
  external channel supports it.

httpSMS does not add a new acknowledgement endpoint. Fetching the outstanding
message and posting existing message events remain the source of truth.

### 7. Adapter API flow

A URL-backed adapter uses the existing public API in the same way as the
Android gateway:

1. A user creates or updates a phone with an E.164 number and sets `fcm_token`
   to the adapter's HTTPS URL.
2. The user creates a phone API key assigned to that phone/number and configures
   the adapter with it.
3. httpSMS schedules outgoing messages with the existing rate limit and send
   schedule.
4. When a message becomes due, httpSMS POSTs the callback containing
   `KEY_MESSAGE_ID`.
5. The adapter fetches the message through `/v1/messages/outstanding` using its
   phone API key.
6. The adapter sends the message through WhatsApp or another external channel.
7. The adapter posts the existing sent, delivered, or failed message events.
8. The adapter posts inbound messages through `/v1/messages/receive`.
9. For `KEY_HEARTBEAT_ID`, the adapter performs the same heartbeat callback flow
   expected from the Android application.

The phone API key's existing phone-number scope prevents an adapter from
fetching messages belonging to another number. The adapter must not use a
general user API key for gateway operations.

Encrypted message content remains unchanged. If a user enables encryption, the
adapter is responsible for implementing the same compatible encryption and
decryption behavior expected of the Android gateway.

### 8. HTTP client and retry ownership

Phone registration validates only URL syntax, the HTTPS scheme, and the
presence of a hostname. URL user information remains valid. The feature does
not perform endpoint DNS/IP classification, custom dialing, or private-host
allowlisting.

`Container.NotificationHTTPClient` is a standard `http.Client` using the
existing webhook-style `go-otelroundtripper` pattern without transport-level
retries or custom URL redaction. The client preserves Go's default transport
and redirect behavior.

`HTTPNotificationSender` creates one reusable `retry-go/v5` retrier during
initialization. The retrier owns exactly three total attempts and exponential
backoff, but does not bind a caller context because it is reused across sends.
Each delivery checks its caller context, creates a fresh request and body, and
applies a five-second child context per attempt. Payload encoding, request
creation, one-attempt delivery, and retry configuration remain focused
operations.

### 9. Validation and API compatibility

Keep these public fields and routes unchanged:

- `Phone.FcmToken` / JSON `fcm_token`;
- `PUT /v1/phones`;
- `PUT /v1/phones/fcm-token`;
- all outstanding-message, event, receive-message, and heartbeat routes.

Extend phone validation only when `fcm_token` is URL-like:

- enforce valid HTTPS syntax and require a hostname;
- preserve the existing maximum token length;
- return field-level `fcm_token` validation errors for malformed or non-HTTPS
  URL-like values.

Opaque FCM token validation remains unchanged. Existing stored Android tokens
require no migration.

Update request and Swagger descriptions to explain that `fcm_token` accepts
either an FCM registration token or an HTTPS adapter callback URL.
Regenerate Swagger documentation after implementation.

### 10. Observability

Use the existing request, database, and OpenTelemetry logging behavior without
special redaction for notification tokens or callback URLs. Rely on the
existing OpenTelemetry HTTP round-tripper for outbound request spans and
metrics; do not add sender-specific attempt spans or metrics.

## Components and Expected Files

Implementation is expected to touch:

- `api/pkg/entities/phone.go` for transport and URL helpers;
- `api/pkg/validators/phone_handler_validator.go` for URL-token validation;
- `api/pkg/services/phone_notification_service.go` to build generic
  notifications and preserve existing state transitions;
- `api/pkg/services/fcm_client.go` to adapt Firebase to the neutral sender;
- `api/pkg/services/notification_sender.go` for the neutral notification,
  sender interface, and Firebase sender;
- `api/pkg/services/phone_notification_dispatcher.go` for phone transport
  routing;
- `api/pkg/services/http_notification_sender.go` for HTTP payload encoding and
  delivery;
- `api/pkg/services/emulator_fcm_client.go` only as needed to preserve the
  emulator behind the adapted interface;
- `api/pkg/di/container.go` for dispatcher, OpenTelemetry HTTP client, and
  sender construction;
- phone request annotations and generated Swagger files.
- `tests/adapter-emulator/` for an HTTPS gateway emulator that consumes
  callbacks and exercises existing phone API routes;
- `tests/adapter_integration_test.go`, `tests/docker-compose.yml`, test
  certificate generation, CI setup, and `tests/README.md` for end-to-end
  coverage.

The implementation must not move scheduling, message expiration, message event
handling, or phone API-key authorization into the new transport code.

## Testing

### Phone helper tests

Cover:

- ordinary FCM tokens selecting Firebase;
- valid HTTPS URLs selecting HTTP;
- empty and nil tokens;
- `http`, `ftp`, missing host, and malformed URLs being invalid;
- URL-like invalid values never falling through to Firebase.

### HTTP client wiring tests

Cover the standard OpenTelemetry round-tripper wrapping
`http.DefaultTransport` without retryable HTTP transport attempts. URL
validation must not perform DNS or address checks.

### HTTP sender tests

Use an HTTP test server or controlled transport to cover:

- FCM-compatible message and heartbeat JSON;
- message priority and TTL mapping;
- stable `X-httpSMS-Notification-ID` across transport retries;
- any `2xx` response succeeding with the body ignored;
- retrying network errors, `408`, `429`, and `5xx`;
- not retrying other `4xx` responses;
- three-attempt maximum and five-second per-attempt timeout;
- bounded response-body handling;
- errors and logs not exposing the complete URL.

### Dispatcher and service tests

Cover:

- Firebase tokens using the Firebase sender;
- URL tokens using the HTTP sender;
- outgoing messages and heartbeats using the same dispatcher;
- HTTP success using the existing sent-notification path;
- HTTP terminal failure using the existing failed-notification path;
- message send-attempt count and expiration scheduling remaining unchanged;
- scheduling, exact-send time, per-minute limits, and send schedules remaining
  independent of transport.

### Adapter integration emulator

Add a dedicated Go service under `tests/adapter-emulator/`. It exposes:

- an HTTPS callback listener used by the API;
- an HTTP-only test control listener exposed to the host test runner;
- an in-memory gateway registry mapping a unique callback path to a phone
  number and phone API key;
- callback records keyed by `X-httpSMS-Notification-ID`.

For `KEY_MESSAGE_ID`, the emulator:

1. deduplicates the notification ID;
2. fetches `/v1/messages/outstanding` using the registered phone API key;
3. posts the existing `SENT` event;
4. posts the existing `DELIVERED` event;
5. records the fetched message and final adapter action for test assertions.

For `KEY_HEARTBEAT_ID`, the emulator posts `/v1/heartbeats` for the registered
phone and records the heartbeat wake-up. A control endpoint also instructs the
emulator to submit an incoming message through `/v1/messages/receive`.

The integration stack generates a throwaway CA and server certificate whose SAN
contains `adapter-emulator`, mounts the server certificate into the emulator,
and makes the CA available to the API's Go trust store through `SSL_CERT_FILE`.
Docker DNS resolves the private emulator hostname through the standard Go HTTP
transport; no private-host allowlist is configured.

Add end-to-end tests for:

- **Outgoing:** URL-backed phone callback -> outstanding fetch -> sent event ->
  delivered event -> final delivered API status.
- **Incoming:** emulator control request -> existing receive-message API ->
  final received API status and matching owner/contact/content.
- **Heartbeat:** internal `phone.heartbeat.missed` CloudEvent -> URL callback ->
  emulator heartbeat POST -> heartbeat visible through the user API.
- Callback payload keys, notification ID header, unique callback handling, and
  phone API-key scoping.

Run:

```bash
cd api
go test ./...
```

After annotation changes, regenerate Swagger:

```bash
cd api
swag init --requiredByDefault --parseDependency --parseInternal
```

Run the Docker integration suite:

```bash
cd tests
bash generate-firebase-credentials.sh
bash generate-adapter-certificates.sh
docker compose up -d --build --wait
docker compose wait seed
go test -v -timeout 300s ./...
docker compose down -v
```

## Rollout

The feature is backward compatible and requires no data migration. Deploy the
backend before configuring URL tokens.

Operational monitoring should distinguish Firebase and HTTP notification
delivery. Initial rollout should watch:

- HTTP callback success and retry rates;
- terminal failures by status class;
- callback latency;
- transport failures;
- message expiration after a successful HTTP wake-up;
- duplicate notification IDs observed by test adapters.

Rollback is code-only: existing Android tokens continue to be valid, while
URL-backed phones stop receiving wake-ups if the feature is rolled back.

## Out of Scope

- WhatsApp provider integration or any adapter implementation.
- Provider credentials, sessions, QR-code login, templates, or media mapping.
- New polling/list-outstanding APIs.
- A new adapter acknowledgement endpoint.
- Signed callbacks, HMAC, JWT, or mutual TLS.
- Separate transport or callback URL database columns.
- Changes to message scheduling, backpressure, expiration, or retry-count
  algorithms.
- Web UI for configuring adapters.
- Android application changes.
- General-purpose outbound webhook refactoring.
- Destination-specific DNS/IP filtering or allowlists.
