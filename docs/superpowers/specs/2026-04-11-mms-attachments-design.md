# MMS Attachment Support — Design Spec

## Problem

The Android app now forwards MMS attachments (as base64-encoded data) when receiving MMS messages via `HttpSmsApiService.receive()`. The API server needs to:

1. Accept attachment data in the receive endpoint
2. Upload attachments to cloud storage (GCS or in-memory)
3. Store download URLs in the Message entity
4. Serve a download endpoint for retrieving attachments
5. Include attachment URLs in webhook event payloads

## Approach

**Approach A: Storage Interface + Minimal New Code** — Add an `AttachmentStorage` interface with GCS and memory implementations. Upload logic lives in the existing `MessageService.ReceiveMessage()` flow (synchronous). A new `AttachmentHandler` serves downloads. No new database tables — content type is encoded in the URL file extension.

## Design

### 1. Storage Interface

**New file: `pkg/repositories/attachment_storage.go`**

```go
type AttachmentStorage interface {
    Upload(ctx context.Context, path string, data []byte) error
    Download(ctx context.Context, path string) ([]byte, error)
    Delete(ctx context.Context, path string) error
}
```

**GCS Implementation** (`pkg/repositories/gcs_attachment_storage.go`):

- Uses `cloud.google.com/go/storage` SDK
- Configured with bucket name from `GCS_BUCKET_NAME` env var
- Stores objects at: `attachments/{userID}/{messageID}/{index}/{name}.{ext}`
- Extension derived from content type (e.g., `image/jpeg` → `.jpg`); falls back to `.bin` only when no mapping exists

**Memory Implementation** (`pkg/repositories/memory_attachment_storage.go`):

- `sync.Map`-backed in-memory store
- Used when `GCS_BUCKET_NAME` is empty/unset (local dev, testing)

**DI selection** (in `container.go`):

```go
if os.Getenv("GCS_BUCKET_NAME") != "" {
    return NewGCSAttachmentStorage(bucket, tracer, logger)
}
return NewMemoryAttachmentStorage(tracer, logger)
```

### 2. Environment Variables

| Variable          | Description                                             | Default                            |
| ----------------- | ------------------------------------------------------- | ---------------------------------- |
| `GCS_BUCKET_NAME` | GCS bucket for attachments. Empty = use memory storage. | `httpsms-86c51.appspot.com` (prod) |

The API base URL for constructing download links is derived from `EVENTS_QUEUE_ENDPOINT` by stripping the `/v1/events` suffix.

### 3. Request & Validation Changes

**Updated `MessageReceive` request** (`pkg/requests/`):

```go
type MessageReceive struct {
    From        string              `json:"from"`
    To          string              `json:"to"`
    Content     string              `json:"content"`
    Encrypted   bool                `json:"encrypted"`
    SIM         entities.SIM        `json:"sim"`
    Timestamp   time.Time           `json:"timestamp"`
    Attachments []MessageAttachment `json:"attachments"` // NEW
}

type MessageAttachment struct {
    Name        string `json:"name"`
    ContentType string `json:"content_type"`
    Content     string `json:"content"` // base64-encoded
}
```

**Updated `MessageReceiveParams`** (`pkg/services/`):
The `ToMessageReceiveParams()` method must propagate attachments to the service layer:

```go
type MessageReceiveParams struct {
    // ... existing fields ...
    Attachments []requests.MessageAttachment // NEW — raw attachment data for upload
}
```

**Filename sanitization:**
The `Name` field from the Android client must be sanitized to prevent path traversal attacks. Strip all path separators (`/`, `\`), directory traversal sequences (`..`), and non-printable characters. If the sanitized name is empty, use a fallback like `attachment-{index}`.

**Content type allowlist:**
Only allow known-safe MIME types from the extension mapping table (Section 5). Reject attachments with unrecognized content types with a 400 error.

**Validation rules** (in `pkg/validators/`):

- Attachment count must be ≤ 10
- Each decoded attachment must be ≤ 1.5 MB (1,572,864 bytes)
- Content type must be in the allowlist
- If any limit is exceeded → **reject entire request with 400 Bad Request**
- Validation happens before any upload or storage

### 4. Upload Flow (Synchronous in Receive)

In `MessageService.ReceiveMessage()`:

1. Validate attachment count, sizes, and content types
2. Upload attachments **in parallel** using `errgroup`:
   a. Decode base64 content
   b. Sanitize `name` (strip path separators, `..`, non-printable chars; fallback to `attachment-{index}`)
   c. Map `content_type` → file extension (e.g., `image/jpeg` → `.jpg`, unknown → `.bin`)
   d. Upload to storage at path: `attachments/{userID}/{messageID}/{index}/{sanitizedName}.{ext}`
   e. Build download URL: `{apiBaseURL}/v1/attachments/{userID}/{messageID}/{index}/{sanitizedName}.{ext}`
3. If any upload fails → best-effort delete of already-uploaded files, then return 500
4. Collect download URLs into `message.Attachments` (existing `pq.StringArray` field)
5. Set `Attachments` on `MessagePhoneReceivedPayload` before dispatching event
6. `storeReceivedMessage()` copies `payload.Attachments` → `message.Attachments`
7. Store message in database
8. Fire `message.phone.received` event (includes attachment URLs)

### 5. Content Type → Extension Mapping

A utility function maps MIME types to file extensions:

| Content Type      | Extension |
| ----------------- | --------- |
| `image/jpeg`      | `.jpg`    |
| `image/png`       | `.png`    |
| `image/gif`       | `.gif`    |
| `image/webp`      | `.webp`   |
| `image/bmp`       | `.bmp`    |
| `video/mp4`       | `.mp4`    |
| `video/3gpp`      | `.3gp`    |
| `audio/mpeg`      | `.mp3`    |
| `audio/ogg`       | `.ogg`    |
| `audio/amr`       | `.amr`    |
| `application/pdf` | `.pdf`    |
| `text/vcard`      | `.vcf`    |
| `text/x-vcard`    | `.vcf`    |
| _(default)_       | `.bin`    |

This covers common MMS content types. New mappings can be added as needed.

### 6. Download Handler

**New file: `pkg/handlers/attachment_handler.go`**

**Route:** `GET /v1/attachments/:userID/:messageID/:attachmentIndex/:filename`

- Registered **without authentication middleware** — publicly accessible, consistent with outgoing attachment URLs
- The `{userID}/{messageID}/{attachmentIndex}` path components provide sufficient obscurity (UUIDs are unguessable)

**Download flow:**

1. Parse URL params (userID, messageID, attachmentIndex, filename)
2. Construct storage path: `attachments/{userID}/{messageID}/{attachmentIndex}/{filename}`
3. Fetch bytes from `AttachmentStorage.Download(ctx, path)`
4. Derive `Content-Type` from filename extension
5. Set security headers: `Content-Disposition: attachment`, `X-Content-Type-Options: nosniff`
6. Respond with binary data + correct `Content-Type` header
7. Return 404 if attachment not found in storage

### 7. Webhook Event Changes

**Updated `MessagePhoneReceivedPayload`** (`pkg/events/message_phone_received_event.go`):

```go
type MessagePhoneReceivedPayload struct {
    MessageID   uuid.UUID       `json:"message_id"`
    UserID      entities.UserID `json:"user_id"`
    Owner       string          `json:"owner"`
    Encrypted   bool            `json:"encrypted"`
    Contact     string          `json:"contact"`
    Timestamp   time.Time       `json:"timestamp"`
    Content     string          `json:"content"`
    SIM         entities.SIM    `json:"sim"`
    Attachments []string        `json:"attachments"` // NEW — download URLs
}
```

Webhook subscribers will receive the array of download URLs. They can `GET` each URL directly — no authentication required.

### 8. Files Changed / Created

**New files:**

- `pkg/repositories/attachment_storage.go` — Interface definition
- `pkg/repositories/gcs_attachment_storage.go` — GCS implementation
- `pkg/repositories/memory_attachment_storage.go` — Memory implementation
- `pkg/handlers/attachment_handler.go` — Download endpoint handler
- `pkg/validators/attachment_handler_validator.go` — Download param validation

**Modified files:**

- `pkg/requests/message_receive.go` (or wherever `MessageReceive` is defined) — Add `Attachments` field
- `pkg/validators/message_handler_validator.go` — Add attachment count/size validation
- `pkg/services/message_service.go` — Add upload logic to `ReceiveMessage()`
- `pkg/events/message_phone_received_event.go` — Add `Attachments` field to payload
- `pkg/di/container.go` — Wire storage, new handler, pass storage to message service
- `api/.env.docker` — Add `GCS_BUCKET_NAME` variable
- `go.mod` / `go.sum` — Add `cloud.google.com/go/storage` dependency

### 9. Validation Constraints

| Constraint                      | Value                    | Behavior                                             |
| ------------------------------- | ------------------------ | ---------------------------------------------------- |
| Max attachment count            | 10                       | 400 Bad Request                                      |
| Max attachment size (decoded)   | 1.5 MB (1,572,864 bytes) | 400 Bad Request                                      |
| Content type not in allowlist   | —                        | 400 Bad Request                                      |
| Missing/empty attachments array | —                        | Message stored without attachments (normal SMS flow) |

### 10. Error Handling

- Storage upload failure → Best-effort delete of already-uploaded attachments, then return 500; message is NOT stored
- Storage download failure → Return 404 or 500 depending on error type
- Invalid base64 content → Return 400 Bad Request
- All errors wrapped with `stacktrace.Propagate()` per project convention
