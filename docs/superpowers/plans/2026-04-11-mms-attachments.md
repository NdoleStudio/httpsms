# MMS Attachment Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add MMS attachment upload/download support to the httpSMS API so received MMS attachments are stored in cloud storage and downloadable via a public URL.

**Architecture:** Android sends base64-encoded attachments in the receive request. The API decodes and uploads them to GCS (or in-memory storage) via a storage interface, stores download URLs in the existing `Message.Attachments` field, and exposes an unauthenticated download endpoint. The webhook event payload includes attachment URLs.

**Tech Stack:** Go, Fiber v2, GORM, `cloud.google.com/go/storage`, `errgroup`, `stacktrace`

---

## File Structure

**New files:**

| File                                                | Responsibility                                                                   |
| --------------------------------------------------- | -------------------------------------------------------------------------------- |
| `api/pkg/repositories/attachment_storage.go`        | `AttachmentStorage` interface + content-type-to-extension mapping + sanitization |
| `api/pkg/repositories/gcs_attachment_storage.go`    | GCS implementation of `AttachmentStorage`                                        |
| `api/pkg/repositories/memory_attachment_storage.go` | In-memory implementation of `AttachmentStorage`                                  |
| `api/pkg/repositories/attachment_storage_test.go`   | Unit tests for content-type mapping and filename sanitization                    |
| `api/pkg/handlers/attachment_handler.go`            | Download endpoint handler (`GET /v1/attachments/...`)                            |

**Modified files:**

| File                                              | Change                                                                          |
| ------------------------------------------------- | ------------------------------------------------------------------------------- |
| `api/pkg/requests/message_receive_request.go`     | Add `Attachments` field + `MessageAttachment` struct                            |
| `api/pkg/services/message_service.go`             | Add `Attachments` to params, upload logic in `ReceiveMessage()`, set on message |
| `api/pkg/validators/message_handler_validator.go` | Add attachment count/size/content-type validation                               |
| `api/pkg/events/message_phone_received_event.go`  | Add `Attachments []string` to payload                                           |
| `api/pkg/di/container.go`                         | Wire `AttachmentStorage`, `AttachmentHandler`, `RegisterAttachmentRoutes()`     |
| `api/.env.docker`                                 | Add `GCS_BUCKET_NAME`                                                           |
| `api/go.mod` / `api/go.sum`                       | Add `cloud.google.com/go/storage` and `golang.org/x/sync` (errgroup)            |

---

### Task 1: Add GCS SDK and errgroup dependencies

**Files:**

- Modify: `api/go.mod`

- [ ] **Step 1: Add dependencies**

```bash
cd api && go get cloud.google.com/go/storage && go get golang.org/x/sync
```

- [ ] **Step 2: Verify build still works**

Run: `cd api && go build ./...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
cd api && git add go.mod go.sum && git commit -m "chore: add cloud.google.com/go/storage and golang.org/x/sync deps"
```

---

### Task 2: Storage interface, content-type mapping, and filename sanitization

**Files:**

- Create: `api/pkg/repositories/attachment_storage.go`
- Create: `api/pkg/repositories/attachment_storage_test.go`

- [ ] **Step 1: Write the test file**

Create `api/pkg/repositories/attachment_storage_test.go`:

```go
package repositories

import "testing"

func TestExtensionFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/bmp", ".bmp"},
		{"video/mp4", ".mp4"},
		{"video/3gpp", ".3gp"},
		{"audio/mpeg", ".mp3"},
		{"audio/ogg", ".ogg"},
		{"audio/amr", ".amr"},
		{"application/pdf", ".pdf"},
		{"text/vcard", ".vcf"},
		{"text/x-vcard", ".vcf"},
		{"application/octet-stream", ".bin"},
		{"unknown/type", ".bin"},
		{"", ".bin"},
	}
	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := ExtensionFromContentType(tt.contentType)
			if got != tt.expected {
				t.Errorf("ExtensionFromContentType(%q) = %q, want %q", tt.contentType, got, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		expected string
	}{
		{"photo.jpg", 0, "photo"},
		{"../../etc/passwd", 0, "etcpasswd"},
		{"hello/world\\test", 0, "helloworldtest"},
		{"normal_file", 0, "normal_file"},
		{"", 0, "attachment-0"},
		{"   ", 0, "attachment-0"},
		{"...", 1, "attachment-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.name, tt.index)
			if got != tt.expected {
				t.Errorf("SanitizeFilename(%q, %d) = %q, want %q", tt.name, tt.index, got, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd api && go test ./pkg/repositories/ -run "TestExtensionFromContentType|TestSanitizeFilename" -v`
Expected: FAIL — functions not defined

- [ ] **Step 3: Write the storage interface and utility functions**

Create `api/pkg/repositories/attachment_storage.go`:

```go
package repositories

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// AttachmentStorage is the interface for storing and retrieving message attachments
type AttachmentStorage interface {
	// Upload stores attachment data at the given path
	Upload(ctx context.Context, path string, data []byte) error
	// Download retrieves attachment data from the given path
	Download(ctx context.Context, path string) ([]byte, error)
	// Delete removes an attachment at the given path
	Delete(ctx context.Context, path string) error
}

// contentTypeExtensions maps MIME types to file extensions
var contentTypeExtensions = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"image/bmp":       ".bmp",
	"video/mp4":       ".mp4",
	"video/3gpp":      ".3gp",
	"audio/mpeg":      ".mp3",
	"audio/ogg":       ".ogg",
	"audio/amr":       ".amr",
	"application/pdf": ".pdf",
	"text/vcard":      ".vcf",
	"text/x-vcard":    ".vcf",
}

// AllowedContentTypes returns the set of allowed MIME types for attachments
func AllowedContentTypes() map[string]bool {
	allowed := make(map[string]bool, len(contentTypeExtensions))
	for ct := range contentTypeExtensions {
		allowed[ct] = true
	}
	return allowed
}

// ExtensionFromContentType returns the file extension for a MIME content type.
// Returns ".bin" if the content type is not recognized.
func ExtensionFromContentType(contentType string) string {
	if ext, ok := contentTypeExtensions[contentType]; ok {
		return ext
	}
	return ".bin"
}

// ContentTypeFromExtension returns the MIME content type for a file extension.
// Returns "application/octet-stream" if the extension is not recognized.
func ContentTypeFromExtension(ext string) string {
	for ct, e := range contentTypeExtensions {
		if e == ext {
			return ct
		}
	}
	return "application/octet-stream"
}

// SanitizeFilename removes path separators and traversal sequences from a filename.
// Returns "attachment-{index}" if the sanitized name is empty.
func SanitizeFilename(name string, index int) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "..", "")
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Sprintf("attachment-%d", index)
	}
	return name
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd api && go test ./pkg/repositories/ -run "TestExtensionFromContentType|TestSanitizeFilename" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd api && git add -A && git commit -m "feat: add AttachmentStorage interface and content-type utilities"
```

---

### Task 3: Memory storage implementation

**Files:**

- Create: `api/pkg/repositories/memory_attachment_storage.go`

- [ ] **Step 1: Write the implementation**

Create `api/pkg/repositories/memory_attachment_storage.go`:

```go
package repositories

import (
	"context"
	"fmt"
	"sync"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// MemoryAttachmentStorage stores attachments in memory
type MemoryAttachmentStorage struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	data   sync.Map
}

// NewMemoryAttachmentStorage creates a new MemoryAttachmentStorage
func NewMemoryAttachmentStorage(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) *MemoryAttachmentStorage {
	return &MemoryAttachmentStorage{
		logger: logger.WithService(fmt.Sprintf("%T", &MemoryAttachmentStorage{})),
		tracer: tracer,
	}
}

// Upload stores attachment data at the given path
func (s *MemoryAttachmentStorage) Upload(ctx context.Context, path string, data []byte) error {
	_, span := s.tracer.Start(ctx)
	defer span.End()

	s.data.Store(path, data)
	s.logger.Info(fmt.Sprintf("stored attachment at path [%s] with size [%d]", path, len(data)))
	return nil
}

// Download retrieves attachment data from the given path
func (s *MemoryAttachmentStorage) Download(ctx context.Context, path string) ([]byte, error) {
	_, span := s.tracer.Start(ctx)
	defer span.End()

	value, ok := s.data.Load(path)
	if !ok {
		return nil, stacktrace.NewError(fmt.Sprintf("attachment not found at path [%s]", path))
	}
	return value.([]byte), nil
}

// Delete removes an attachment at the given path
func (s *MemoryAttachmentStorage) Delete(ctx context.Context, path string) error {
	_, span := s.tracer.Start(ctx)
	defer span.End()

	s.data.Delete(path)
	s.logger.Info(fmt.Sprintf("deleted attachment at path [%s]", path))
	return nil
}
```

- [ ] **Step 2: Verify build**

Run: `cd api && go build ./...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
cd api && git add -A && git commit -m "feat: add MemoryAttachmentStorage implementation"
```

---

### Task 4: GCS storage implementation

**Files:**

- Create: `api/pkg/repositories/gcs_attachment_storage.go`

- [ ] **Step 1: Write the implementation**

Create `api/pkg/repositories/gcs_attachment_storage.go`:

```go
package repositories

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// GCSAttachmentStorage stores attachments in Google Cloud Storage
type GCSAttachmentStorage struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	client *storage.Client
	bucket string
}

// NewGCSAttachmentStorage creates a new GCSAttachmentStorage
func NewGCSAttachmentStorage(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *storage.Client,
	bucket string,
) *GCSAttachmentStorage {
	return &GCSAttachmentStorage{
		logger: logger.WithService(fmt.Sprintf("%T", &GCSAttachmentStorage{})),
		tracer: tracer,
		client: client,
		bucket: bucket,
	}
}

// Upload stores attachment data at the given path in GCS
func (s *GCSAttachmentStorage) Upload(ctx context.Context, path string, data []byte) error {
	ctx, span := s.tracer.Start(ctx)
	defer span.End()

	writer := s.client.Bucket(s.bucket).Object(path).NewWriter(ctx)
	if _, err := writer.Write(data); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot write attachment to GCS path [%s]", path)))
	}

	if err := writer.Close(); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot close GCS writer for path [%s]", path)))
	}

	s.logger.Info(fmt.Sprintf("uploaded attachment to GCS path [%s/%s] with size [%d]", s.bucket, path, len(data)))
	return nil
}

// Download retrieves attachment data from the given path in GCS
func (s *GCSAttachmentStorage) Download(ctx context.Context, path string) ([]byte, error) {
	ctx, span := s.tracer.Start(ctx)
	defer span.End()

	reader, err := s.client.Bucket(s.bucket).Object(path).NewReader(ctx)
	if err != nil {
		return nil, s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot open GCS reader for path [%s]", path)))
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot read attachment from GCS path [%s]", path)))
	}

	return data, nil
}

// Delete removes an attachment at the given path in GCS
func (s *GCSAttachmentStorage) Delete(ctx context.Context, path string) error {
	ctx, span := s.tracer.Start(ctx)
	defer span.End()

	if err := s.client.Bucket(s.bucket).Object(path).Delete(ctx); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot delete GCS object at path [%s]", path)))
	}

	s.logger.Info(fmt.Sprintf("deleted attachment from GCS path [%s/%s]", s.bucket, path))
	return nil
}
```

- [ ] **Step 2: Verify build**

Run: `cd api && go build ./...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
cd api && git add -A && git commit -m "feat: add GCSAttachmentStorage implementation"
```

---

### Task 5: Update request and event structs

**Files:**

- Modify: `api/pkg/requests/message_receive_request.go` (full file)
- Modify: `api/pkg/services/message_service.go:290-300` (MessageReceiveParams)
- Modify: `api/pkg/events/message_phone_received_event.go:14-23` (payload struct)

**Important:** The `requests` package already imports `services` (for `ToMessageReceiveParams`), so we **cannot** import `requests` from `services`. Define a `ServiceAttachment` struct in the services package to avoid a circular import.

- [ ] **Step 1: Add ServiceAttachment to services package**

In `api/pkg/services/message_service.go`, add after the imports (before `MessageService` struct at line 22):

```go
// ServiceAttachment represents attachment data passed to the service layer
type ServiceAttachment struct {
	Name        string
	ContentType string
	Content     string // base64-encoded
}
```

Update `MessageReceiveParams` (lines 290-300) to add `Attachments`:

```go
type MessageReceiveParams struct {
	Contact     string
	UserID      entities.UserID
	Owner       phonenumbers.PhoneNumber
	Content     string
	SIM         entities.SIM
	Timestamp   time.Time
	Encrypted   bool
	Source      string
	Attachments []ServiceAttachment
}
```

- [ ] **Step 2: Add MessageAttachment struct and update MessageReceive request**

In `api/pkg/requests/message_receive_request.go`, add the `MessageAttachment` struct before the `MessageReceive` struct, and add the `Attachments` field:

```go
// MessageAttachment represents a single MMS attachment in a receive request
type MessageAttachment struct {
	// Name is the original filename of the attachment
	Name string `json:"name" example:"photo.jpg"`
	// ContentType is the MIME type of the attachment
	ContentType string `json:"content_type" example:"image/jpeg"`
	// Content is the base64-encoded attachment data
	Content string `json:"content" example:"base64data..."`
}

// MessageReceive is the payload for receiving an SMS/MMS message
type MessageReceive struct {
	request
	From    string `json:"from" example:"+18005550199"`
	To      string `json:"to" example:"+18005550100"`
	Content string `json:"content" example:"This is a sample text message received on a phone"`
	// Encrypted is used to determine if the content is end-to-end encrypted
	Encrypted bool `json:"encrypted" example:"false"`
	// SIM card that received the message
	SIM entities.SIM `json:"sim" example:"SIM1"`
	// Timestamp is the time when the event was emitted
	Timestamp time.Time `json:"timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
	// Attachments is the list of MMS attachments received with the message
	Attachments []MessageAttachment `json:"attachments"`
}
```

Update `ToMessageReceiveParams` to convert attachments:

```go
func (input *MessageReceive) ToMessageReceiveParams(userID entities.UserID, source string) *services.MessageReceiveParams {
	phone, _ := phonenumbers.Parse(input.To, phonenumbers.UNKNOWN_REGION)

	attachments := make([]services.ServiceAttachment, len(input.Attachments))
	for i, a := range input.Attachments {
		attachments[i] = services.ServiceAttachment{
			Name:        a.Name,
			ContentType: a.ContentType,
			Content:     a.Content,
		}
	}

	return &services.MessageReceiveParams{
		Source:      source,
		Contact:     input.From,
		UserID:      userID,
		Timestamp:   input.Timestamp,
		Encrypted:   input.Encrypted,
		Owner:       *phone,
		Content:     input.Content,
		SIM:         input.SIM,
		Attachments: attachments,
	}
}
```

- [ ] **Step 3: Update MessagePhoneReceivedPayload**

In `api/pkg/events/message_phone_received_event.go`, add `Attachments` field to the payload struct (after the `SIM` field):

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
	Attachments []string        `json:"attachments"`
}
```

- [ ] **Step 4: Verify build compiles (will fail until service constructor is updated)**

Run: `cd api && go vet ./pkg/requests/... ./pkg/events/...`
Expected: No errors in these packages

- [ ] **Step 5: Commit**

```bash
cd api && git add -A && git commit -m "feat: add attachment fields to request, params, and event structs"
```

---

### Task 6: Add attachment validation

**Files:**

- Modify: `api/pkg/validators/message_handler_validator.go:49-77`

- [ ] **Step 1: Update ValidateMessageReceive to validate attachments**

In `api/pkg/validators/message_handler_validator.go`, replace the `ValidateMessageReceive` method (lines 49-77) with:

```go
const (
	maxAttachmentCount = 10
	maxAttachmentSize  = (3 * 1024 * 1024) / 2 // 1.5 MB
)

// ValidateMessageReceive validates the requests.MessageReceive request
func (validator MessageHandlerValidator) ValidateMessageReceive(_ context.Context, request requests.MessageReceive) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"to": []string{
				"required",
				phoneNumberRule,
			},
			"from": []string{
				"required",
			},
			"content": []string{
				"required",
				"min:1",
				"max:2048",
			},
			"sim": []string{
				"required",
				"in:" + strings.Join([]string{
					string(entities.SIM1),
					string(entities.SIM2),
				}, ","),
			},
		},
	})

	errors := v.ValidateStruct()

	if len(request.Attachments) > 0 {
		attachmentErrors := validator.validateAttachments(request.Attachments)
		for key, values := range attachmentErrors {
			for _, value := range values {
				errors.Add(key, value)
			}
		}
	}

	return errors
}

func (validator MessageHandlerValidator) validateAttachments(attachments []requests.MessageAttachment) url.Values {
	errors := url.Values{}
	allowedTypes := repositories.AllowedContentTypes()

	if len(attachments) > maxAttachmentCount {
		errors.Add("attachments", fmt.Sprintf("attachment count [%d] exceeds maximum of [%d]", len(attachments), maxAttachmentCount))
		return errors
	}

	for i, attachment := range attachments {
		if !allowedTypes[attachment.ContentType] {
			errors.Add("attachments", fmt.Sprintf("attachment [%d] has unsupported content type [%s]", i, attachment.ContentType))
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(attachment.Content)
		if err != nil {
			errors.Add("attachments", fmt.Sprintf("attachment [%d] has invalid base64 content", i))
			continue
		}

		if len(decoded) > maxAttachmentSize {
			errors.Add("attachments", fmt.Sprintf("attachment [%d] size [%d] exceeds maximum of [%d] bytes", i, len(decoded), maxAttachmentSize))
		}
	}

	return errors
}
```

Add these imports to the file: `"encoding/base64"`, `"github.com/NdoleStudio/httpsms/pkg/repositories"`.

- [ ] **Step 2: Verify build**

Run: `cd api && go vet ./pkg/validators/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
cd api && git add -A && git commit -m "feat: add attachment count, size, and content-type validation"
```

---

### Task 7: Upload logic in MessageService.ReceiveMessage()

**Files:**

- Modify: `api/pkg/services/message_service.go:22-47` (struct + constructor)
- Modify: `api/pkg/services/message_service.go:302-337` (ReceiveMessage)
- Modify: `api/pkg/services/message_service.go:550-581` (storeReceivedMessage)

- [ ] **Step 1: Add AttachmentStorage and apiBaseURL to MessageService**

Update the `MessageService` struct (lines 22-30):

```go
type MessageService struct {
	service
	logger            telemetry.Logger
	tracer            telemetry.Tracer
	eventDispatcher   *EventDispatcher
	phoneService      *PhoneService
	repository        repositories.MessageRepository
	attachmentStorage repositories.AttachmentStorage
	apiBaseURL        string
}
```

Update `NewMessageService` (lines 33-47) to accept the new parameters:

```go
func NewMessageService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.MessageRepository,
	eventDispatcher *EventDispatcher,
	phoneService *PhoneService,
	attachmentStorage repositories.AttachmentStorage,
	apiBaseURL string,
) (s *MessageService) {
	return &MessageService{
		logger:            logger.WithService(fmt.Sprintf("%T", s)),
		tracer:            tracer,
		repository:        repository,
		phoneService:      phoneService,
		eventDispatcher:   eventDispatcher,
		attachmentStorage: attachmentStorage,
		apiBaseURL:        apiBaseURL,
	}
}
```

- [ ] **Step 2: Add the uploadAttachments helper method**

Add this after `storeReceivedMessage`. Add imports: `"encoding/base64"`, `"golang.org/x/sync/errgroup"`:

```go
func (service *MessageService) uploadAttachments(ctx context.Context, userID entities.UserID, messageID uuid.UUID, attachments []ServiceAttachment) ([]string, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	g, gCtx := errgroup.WithContext(ctx)
	urls := make([]string, len(attachments))
	paths := make([]string, len(attachments))

	for i, attachment := range attachments {
		i, attachment := i, attachment
		g.Go(func() error {
			decoded, err := base64.StdEncoding.DecodeString(attachment.Content)
			if err != nil {
				return stacktrace.Propagate(err, fmt.Sprintf("cannot decode base64 content for attachment [%d]", i))
			}

			sanitizedName := repositories.SanitizeFilename(attachment.Name, i)
			ext := repositories.ExtensionFromContentType(attachment.ContentType)
			filename := sanitizedName + ext

			path := fmt.Sprintf("attachments/%s/%s/%d/%s", userID, messageID, i, filename)
			paths[i] = path

			if err = service.attachmentStorage.Upload(gCtx, path, decoded); err != nil {
				return stacktrace.Propagate(err, fmt.Sprintf("cannot upload attachment [%d] to path [%s]", i, path))
			}

			urls[i] = fmt.Sprintf("%s/v1/attachments/%s/%s/%d/%s", service.apiBaseURL, userID, messageID, i, filename)
			ctxLogger.Info(fmt.Sprintf("uploaded attachment [%d] to [%s]", i, path))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		for _, path := range paths {
			if path != "" {
				_ = service.attachmentStorage.Delete(ctx, path)
			}
		}
		return nil, stacktrace.Propagate(err, "cannot upload attachments")
	}

	return urls, nil
}
```

- [ ] **Step 3: Update ReceiveMessage to upload attachments before event dispatch**

Replace the `ReceiveMessage` method (lines 302-337):

```go
func (service *MessageService) ReceiveMessage(ctx context.Context, params *MessageReceiveParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	messageID := uuid.New()
	var attachmentURLs []string

	if len(params.Attachments) > 0 {
		ctxLogger.Info(fmt.Sprintf("uploading [%d] attachments for message [%s]", len(params.Attachments), messageID))
		var err error
		attachmentURLs, err = service.uploadAttachments(ctx, params.UserID, messageID, params.Attachments)
		if err != nil {
			msg := fmt.Sprintf("cannot upload attachments for message [%s]", messageID)
			return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}
	}

	eventPayload := events.MessagePhoneReceivedPayload{
		MessageID:   messageID,
		UserID:      params.UserID,
		Encrypted:   params.Encrypted,
		Owner:       phonenumbers.Format(&params.Owner, phonenumbers.E164),
		Contact:     params.Contact,
		Timestamp:   params.Timestamp,
		Content:     params.Content,
		SIM:         params.SIM,
		Attachments: attachmentURLs,
	}

	ctxLogger.Info(fmt.Sprintf("creating cloud event for received with ID [%s]", eventPayload.MessageID))

	event, err := service.createMessagePhoneReceivedEvent(params.Source, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot create %T from payload with message id [%s]", event, eventPayload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] and message id [%s]", event.Type(), event.ID(), eventPayload.MessageID))

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s]", event.Type(), event.ID())
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	ctxLogger.Info(fmt.Sprintf("event [%s] dispatched successfully", event.ID()))

	return service.storeReceivedMessage(ctx, eventPayload)
}
```

- [ ] **Step 4: Update storeReceivedMessage to set Attachments on message**

In the `storeReceivedMessage` method (lines 550-581), add `Attachments` to the message construction:

```go
	message := &entities.Message{
		ID:                params.MessageID,
		Owner:             params.Owner,
		UserID:            params.UserID,
		Contact:           params.Contact,
		Content:           params.Content,
		Attachments:       params.Attachments,
		SIM:               params.SIM,
		Encrypted:         params.Encrypted,
		Type:              entities.MessageTypeMobileOriginated,
		Status:            entities.MessageStatusReceived,
		RequestReceivedAt: params.Timestamp,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
		OrderTimestamp:    params.Timestamp,
		ReceivedAt:        &params.Timestamp,
	}
```

- [ ] **Step 5: Verify the services package compiles**

Run: `cd api && go vet ./pkg/services/...`
Expected: No errors (the full build may still fail until DI container is updated)

- [ ] **Step 6: Commit**

```bash
cd api && git add -A && git commit -m "feat: add attachment upload logic to MessageService.ReceiveMessage()"
```

---

### Task 8: Attachment download handler

**Files:**

- Create: `api/pkg/handlers/attachment_handler.go`

- [ ] **Step 1: Write the handler**

Create `api/pkg/handlers/attachment_handler.go`:

```go
package handlers

import (
	"fmt"
	"path/filepath"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// AttachmentHandler handles attachment download requests
type AttachmentHandler struct {
	handler
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	storage repositories.AttachmentStorage
}

// NewAttachmentHandler creates a new AttachmentHandler
func NewAttachmentHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	storage repositories.AttachmentStorage,
) (h *AttachmentHandler) {
	return &AttachmentHandler{
		logger:  logger.WithService(fmt.Sprintf("%T", h)),
		tracer:  tracer,
		storage: storage,
	}
}

// RegisterRoutes registers the routes for the AttachmentHandler (no auth middleware — public endpoint)
func (h *AttachmentHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/v1/attachments/:userID/:messageID/:attachmentIndex/:filename", h.GetAttachment)
}

// GetAttachment downloads an attachment
// @Summary      Download a message attachment
// @Description  Download an MMS attachment by its path components
// @Tags         Attachments
// @Produce      octet-stream
// @Param        userID           path  string  true  "User ID"
// @Param        messageID        path  string  true  "Message ID"
// @Param        attachmentIndex  path  string  true  "Attachment index"
// @Param        filename         path  string  true  "Filename with extension"
// @Success      200  {file}  binary
// @Failure      404  {object}  responses.NotFoundResponse
// @Failure      500  {object}  responses.InternalServerError
// @Router       /attachments/{userID}/{messageID}/{attachmentIndex}/{filename} [get]
func (h *AttachmentHandler) GetAttachment(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	userID := c.Params("userID")
	messageID := c.Params("messageID")
	attachmentIndex := c.Params("attachmentIndex")
	filename := c.Params("filename")

	path := fmt.Sprintf("attachments/%s/%s/%s/%s", userID, messageID, attachmentIndex, filename)

	ctxLogger.Info(fmt.Sprintf("downloading attachment from path [%s]", path))

	data, err := h.storage.Download(ctx, path)
	if err != nil {
		msg := fmt.Sprintf("cannot download attachment from path [%s]", path)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseNotFound(c, "attachment not found")
	}

	ext := filepath.Ext(filename)
	contentType := repositories.ContentTypeFromExtension(ext)

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", "attachment")
	c.Set("X-Content-Type-Options", "nosniff")

	return c.Send(data)
}
```

- [ ] **Step 2: Verify build**

Run: `cd api && go vet ./pkg/handlers/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
cd api && git add -A && git commit -m "feat: add AttachmentHandler for downloading attachments"
```

---

### Task 9: Wire everything in the DI container and env config

**Files:**

- Modify: `api/pkg/di/container.go:104-163` (NewContainer)
- Modify: `api/pkg/di/container.go:1424-1434` (MessageService creation)
- Modify: `api/.env.docker`

- [ ] **Step 1: Add GCS_BUCKET_NAME to .env.docker**

In `api/.env.docker`, add after the `REDIS_URL=redis://@redis:6379` line (line 49):

```env

# Google Cloud Storage bucket for MMS attachments. Leave empty to use in-memory storage.
GCS_BUCKET_NAME=
```

- [ ] **Step 2: Add `attachmentStorage` field to Container struct**

In `api/pkg/di/container.go`, add `attachmentStorage` to the `Container` struct (around line 82-90):

```go
type Container struct {
	projectID         string
	db                *gorm.DB
	dedicatedDB       *gorm.DB
	version           string
	app               *fiber.App
	eventDispatcher   *services.EventDispatcher
	logger            telemetry.Logger
	attachmentStorage repositories.AttachmentStorage
}
```

- [ ] **Step 3: Add AttachmentStorage, APIBaseURL, and AttachmentHandler getters to container.go**

Add these methods to `api/pkg/di/container.go`. Also add required imports: `"cloud.google.com/go/storage"` and `"context"`:

```go
// AttachmentStorage creates a cached AttachmentStorage based on configuration
func (container *Container) AttachmentStorage() repositories.AttachmentStorage {
	if container.attachmentStorage != nil {
		return container.attachmentStorage
	}

	bucket := os.Getenv("GCS_BUCKET_NAME")
	if bucket != "" {
		container.logger.Debug("creating GCSAttachmentStorage")
		client, err := storage.NewClient(context.Background())
		if err != nil {
			container.logger.Fatal(stacktrace.Propagate(err, "cannot create GCS client"))
		}
		container.attachmentStorage = repositories.NewGCSAttachmentStorage(
			container.Logger(),
			container.Tracer(),
			client,
			bucket,
		)
	} else {
		container.logger.Debug("creating MemoryAttachmentStorage (GCS_BUCKET_NAME not set)")
		container.attachmentStorage = repositories.NewMemoryAttachmentStorage(
			container.Logger(),
			container.Tracer(),
		)
	}

	return container.attachmentStorage
}

// APIBaseURL returns the API base URL derived from EVENTS_QUEUE_ENDPOINT
func (container *Container) APIBaseURL() string {
	endpoint := os.Getenv("EVENTS_QUEUE_ENDPOINT")
	return strings.TrimSuffix(endpoint, "/v1/events")
}

// AttachmentHandler creates a new AttachmentHandler
func (container *Container) AttachmentHandler() (handler *handlers.AttachmentHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", handler))
	return handlers.NewAttachmentHandler(
		container.Logger(),
		container.Tracer(),
		container.AttachmentStorage(),
	)
}

// RegisterAttachmentRoutes registers routes for the /attachments prefix
func (container *Container) RegisterAttachmentRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.AttachmentHandler{}))
	container.AttachmentHandler().RegisterRoutes(container.App())
}
```

- [ ] **Step 3: Update MessageService creation to pass new parameters**

Update the `MessageService()` getter (around line 1424-1434):

```go
func (container *Container) MessageService() (service *services.MessageService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewMessageService(
		container.Logger(),
		container.Tracer(),
		container.MessageRepository(),
		container.EventDispatcher(),
		container.PhoneService(),
		container.AttachmentStorage(),
		container.APIBaseURL(),
	)
}
```

- [ ] **Step 4: Register attachment routes in NewContainer**

In the `NewContainer` function (lines 104-163), add `container.RegisterAttachmentRoutes()` after `container.RegisterMessageRoutes()` (after line 120):

```go
	container.RegisterMessageRoutes()
	container.RegisterAttachmentRoutes()
	container.RegisterBulkMessageRoutes()
```

- [ ] **Step 5: Verify full build**

Run: `cd api && go build ./...`
Expected: Build succeeds — all components are now wired

- [ ] **Step 6: Run all tests**

Run: `cd api && go test ./...`
Expected: All tests pass

- [ ] **Step 7: Commit**

```bash
cd api && git add -A && git commit -m "feat: wire attachment storage and handler in DI container

- Add AttachmentStorage selection (GCS vs memory) based on GCS_BUCKET_NAME env var
- Wire AttachmentHandler for public download endpoint
- Pass storage and API base URL to MessageService
- Add GCS_BUCKET_NAME to .env.docker"
```

---

### Task 10: Final verification

- [ ] **Step 1: Run full build**

Run: `cd api && go build -o ./tmp/main.exe .`
Expected: Build succeeds

- [ ] **Step 2: Run all tests**

Run: `cd api && go test ./... -v`
Expected: All tests pass including `TestExtensionFromContentType` and `TestSanitizeFilename`

- [ ] **Step 3: Verify go vet**

Run: `cd api && go vet ./...`
Expected: No issues

- [ ] **Step 4: Final commit if any remaining changes**

```bash
cd api && git add -A && git diff --cached --stat
```
