# Message Thread Read Receipts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist message-thread read state, mark inbound SMS and missed calls unread, automatically mark opened threads read, and highlight unread threads in the web UI.

**Architecture:** Store `is_read` plus an internal `last_read_at` watermark on each message thread. Replace full-row thread saves with field-scoped GORM updates so concurrent event listeners cannot overwrite a newer read action; inbound updates compare CloudEvent creation time against the stored watermark. Extend the existing thread update endpoint with optional archive/read fields, then have the Nuxt store call it automatically when the thread page opens or refreshes after inbound realtime events.

**Tech Stack:** Go 1.25, Fiber v3, GORM/PostgreSQL, CloudEvents, Pusher, Nuxt 4 SPA, Vue 3, Pinia, Vuetify 4, TypeScript.

## Global Constraints

- Work only in `C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts` on branch `feat/read-receipts`, based on `origin/main`.
- Existing message threads must migrate as read.
- Only incoming SMS and missed-call activity marks a thread unread.
- Outbound messages and delivery/status events preserve the existing read state.
- Opening a thread marks it read automatically; inbound activity while it is open must leave it read.
- Do not add a new endpoint or a manual read/unread control.
- Use the existing `PUT /v1/message-threads/{messageThreadID}` endpoint with optional `is_archived` and `is_read`.
- Use GORM query builders with `WithContext(ctx)`; do not use raw SQL.
- Wrap API errors with `stacktrace.Propagate`/`PropagateWithCode`.
- Web code uses single quotes, no semicolons, and 2-space indentation.
- Every commit must end with:

```text
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
```

- Add end-to-end coverage in the existing `tests/` integration project.

## File Structure

- `api/pkg/entities/message_thread.go`: persisted read fields.
- `api/pkg/entities/message_thread_test.go`: schema-default regression tests.
- `api/pkg/repositories/message_thread_repository.go`: field-scoped update parameter types and interface methods.
- `api/pkg/repositories/gorm_message_thread_repository.go`: transactional activity updates, conditional unread transition, partial user-status update, and partial deleted-message update.
- `api/pkg/repositories/gorm_message_thread_repository_test.go`: update-map ownership tests.
- `api/pkg/services/message_thread_service.go`: read-state business rules and repository orchestration.
- `api/pkg/services/message_thread_service_test.go`: service behavior with a repository stub.
- `api/pkg/requests/message_thread_update_request.go`: optional archive/read request fields.
- `api/pkg/requests/message_thread_update_request_test.go`: request-to-service conversion tests.
- `api/pkg/validators/message_thread_handler_validator.go`: require at least one update field.
- `api/pkg/validators/message_thread_handler_validator_test.go`: empty-payload and valid-payload tests.
- `api/pkg/responses/message_thead_responses.go`: single-thread Swagger response.
- `api/pkg/handlers/message_thread_handler.go`: not-found mapping and corrected Swagger response.
- `api/pkg/listeners/message_thread_listener.go`: incoming SMS and missed-call unread updates.
- `api/pkg/listeners/websocket_listener.go`: missed-call Pusher publication.
- `api/pkg/listeners/read_receipts_test_helpers_test.go`: listener logger and repository test doubles.
- `api/pkg/listeners/message_thread_listener_test.go`: listener route and payload tests.
- `api/pkg/listeners/websocket_listener_test.go`: missed-call route registration test.
- `api/docs/docs.go`, `api/docs/swagger.json`, `api/docs/swagger.yaml`: regenerated API documentation.
- `web/shared/types/api.ts`: regenerated thread/request types.
- `web/app/stores/threads.ts`: automatic read update and local state replacement.
- `web/app/pages/threads/[id]/index.vue`: non-blocking read calls and missed-call realtime refresh.
- `web/app/components/MessageThread.vue`: unread visual treatment.
- `tests/read_receipts_test.go`: Docker-stack read/unread lifecycle coverage.
- `tests/README.md`: integration coverage documentation.

---

### Task 1: Add the read-state API contract

**Files:**

- Modify: `api/pkg/entities/message_thread.go`
- Create: `api/pkg/entities/message_thread_test.go`
- Modify: `api/pkg/requests/message_thread_update_request.go`
- Create: `api/pkg/requests/message_thread_update_request_test.go`
- Modify: `api/pkg/validators/message_thread_handler_validator.go`
- Create: `api/pkg/validators/message_thread_handler_validator_test.go`
- Modify: `api/pkg/responses/message_thead_responses.go`
- Modify: `api/pkg/services/message_thread_service.go`

**Interfaces:**

- Produces: `MessageThread.IsRead bool`, `MessageThread.LastReadAt time.Time`
- Produces: `MessageThreadUpdate.IsArchived *bool`, `MessageThreadUpdate.IsRead *bool`
- Produces: `MessageThreadStatusParams.IsArchived *bool`, `MessageThreadStatusParams.IsRead *bool`
- Produces: `responses.MessageThreadResponse`

- [ ] **Step 1: Write failing entity schema tests**

Create `api/pkg/entities/message_thread_test.go`:

```go
package entities

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageThreadReadFieldsHaveBackwardCompatibleDefaults(t *testing.T) {
	threadType := reflect.TypeOf(MessageThread{})

	isRead, ok := threadType.FieldByName("IsRead")
	require.True(t, ok)
	assert.Contains(t, isRead.Tag.Get("gorm"), "not null")
	assert.Contains(t, isRead.Tag.Get("gorm"), "default:true")
	assert.Equal(t, "is_read", isRead.Tag.Get("json"))

	lastReadAt, ok := threadType.FieldByName("LastReadAt")
	require.True(t, ok)
	assert.Contains(t, lastReadAt.Tag.Get("gorm"), "not null")
	assert.Contains(t, lastReadAt.Tag.Get("gorm"), "default:CURRENT_TIMESTAMP")
	assert.Equal(t, "-", lastReadAt.Tag.Get("json"))
}
```

- [ ] **Step 2: Write failing request and validator tests**

Create `api/pkg/requests/message_thread_update_request_test.go`:

```go
package requests

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMessageThreadUpdateToUpdateParamsPreservesOptionalFields(t *testing.T) {
	threadID := uuid.New()
	isRead := true
	input := MessageThreadUpdate{
		MessageThreadID: threadID.String(),
		IsRead:          &isRead,
	}

	params := input.ToUpdateParams(entities.UserID("user-id"))

	assert.Equal(t, threadID, params.MessageThreadID)
	assert.Equal(t, entities.UserID("user-id"), params.UserID)
	assert.Nil(t, params.IsArchived)
	assert.Same(t, &isRead, params.IsRead)
}
```

Create `api/pkg/validators/message_thread_handler_validator_test.go`:

```go
package validators

import (
	"context"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidateUpdateRequiresAtLeastOneStatusField(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	request := requests.MessageThreadUpdate{
		MessageThreadID: uuid.NewString(),
	}

	errors := validator.ValidateUpdate(context.Background(), request)

	assert.NotEmpty(t, errors.Get("payload"))
}

func TestValidateUpdateAcceptsReadOnlyUpdate(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	isRead := true
	request := requests.MessageThreadUpdate{
		MessageThreadID: uuid.NewString(),
		IsRead:          &isRead,
	}

	errors := validator.ValidateUpdate(context.Background(), request)

	assert.Empty(t, errors)
}
```

- [ ] **Step 3: Run the focused tests and confirm they fail**

Run:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts\api'
go test ./pkg/entities ./pkg/requests ./pkg/validators
```

Expected: compile failures because the read fields and optional request fields do not exist.

- [ ] **Step 4: Add the persisted fields and optional request contract**

Add to `entities.MessageThread` after `IsArchived`:

```go
IsRead     bool      `json:"is_read" gorm:"not null;default:true" example:"true"`
LastReadAt time.Time `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`
```

Change `requests.MessageThreadUpdate` and its conversion:

```go
type MessageThreadUpdate struct {
	request
	IsArchived *bool `json:"is_archived,omitempty" example:"true"`
	IsRead     *bool `json:"is_read,omitempty" example:"true"`

	MessageThreadID string `json:"messageThreadID" swaggerignore:"true"`
}

func (input *MessageThreadUpdate) ToUpdateParams(userID entities.UserID) services.MessageThreadStatusParams {
	return services.MessageThreadStatusParams{
		UserID:          userID,
		MessageThreadID: uuid.MustParse(input.MessageThreadID),
		IsArchived:      input.IsArchived,
		IsRead:          input.IsRead,
	}
}
```

Change the service parameter type:

```go
type MessageThreadStatusParams struct {
	IsArchived      *bool
	IsRead          *bool
	UserID          entities.UserID
	MessageThreadID uuid.UUID
}
```

After `v.ValidateStruct()` in `ValidateUpdate`, add an explicit payload check:

```go
errors := v.ValidateStruct()
if request.IsArchived == nil && request.IsRead == nil {
	errors.Add("payload", "at least one of is_archived or is_read is required")
}
return errors
```

Add a single-thread response beside `MessageThreadsResponse`:

```go
// MessageThreadResponse is the payload containing entities.MessageThread
type MessageThreadResponse struct {
	response
	Data entities.MessageThread `json:"data"`
}
```

- [ ] **Step 5: Run focused tests**

Run:

```powershell
go test ./pkg/entities ./pkg/requests ./pkg/validators
```

Expected: PASS.

- [ ] **Step 6: Commit the contract**

```powershell
git add api/pkg/entities/message_thread.go api/pkg/entities/message_thread_test.go api/pkg/requests/message_thread_update_request.go api/pkg/requests/message_thread_update_request_test.go api/pkg/validators/message_thread_handler_validator.go api/pkg/validators/message_thread_handler_validator_test.go api/pkg/responses/message_thead_responses.go api/pkg/services/message_thread_service.go
@'
feat(api): add thread read state contract

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
'@ | git commit -F -
```

### Task 2: Add atomic thread persistence operations

**Files:**

- Modify: `api/pkg/repositories/message_thread_repository.go`
- Modify: `api/pkg/repositories/gorm_message_thread_repository.go`
- Create: `api/pkg/repositories/gorm_message_thread_repository_test.go`

**Interfaces:**

- Produces: `repositories.MessageThreadActivityUpdate`
- Produces: `repositories.MessageThreadStatusUpdate`
- Produces: `repositories.MessageThreadDeletedUpdate`
- Produces: `MessageThreadRepository.UpdateActivity`, `UpdateStatus`, `UpdateAfterDeletedMessage`
- Removes: `MessageThreadRepository.Update`

- [ ] **Step 1: Write failing update-map ownership tests**

Create `api/pkg/repositories/gorm_message_thread_repository_test.go`:

```go
package repositories

import (
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMessageThreadActivityUpdatesOwnOnlyMessageColumns(t *testing.T) {
	messageID := uuid.New()
	updates := messageThreadActivityUpdates(MessageThreadActivityUpdate{
		Timestamp: time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC),
		MessageID: messageID,
		Content:   "hello",
		Status:    entities.MessageStatusReceived,
	})

	assert.Equal(t, map[string]any{
		"order_timestamp":     time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC),
		"last_message_id":     messageID,
		"last_message_content": "hello",
		"status":              entities.MessageStatusReceived,
	}, updates)
	assert.NotContains(t, updates, "is_read")
	assert.NotContains(t, updates, "is_archived")
	assert.NotContains(t, updates, "last_read_at")
}

func TestMessageThreadStatusUpdatesReadOnly(t *testing.T) {
	isRead := true
	readAt := time.Date(2026, 7, 18, 7, 1, 0, 0, time.UTC)

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		IsRead: &isRead,
		ReadAt: readAt,
	})

	assert.Equal(t, map[string]any{
		"is_read":      true,
		"last_read_at": readAt,
	}, updates)
	assert.NotContains(t, updates, "is_archived")
}

func TestMessageThreadStatusUpdatesArchiveOnly(t *testing.T) {
	isArchived := true

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		IsArchived: &isArchived,
	})

	assert.Equal(t, map[string]any{"is_archived": true}, updates)
	assert.NotContains(t, updates, "is_read")
	assert.NotContains(t, updates, "last_read_at")
}
```

- [ ] **Step 2: Run the repository test and confirm it fails**

Run:

```powershell
go test ./pkg/repositories -run MessageThread -v
```

Expected: compile failures because the parameter types and helper functions do not exist.

- [ ] **Step 3: Define repository update parameters and methods**

In `message_thread_repository.go`, add:

```go
type MessageThreadActivityUpdate struct {
	MessageThreadID uuid.UUID
	UserID          entities.UserID
	Timestamp       time.Time
	MessageID       uuid.UUID
	Content         string
	Status          entities.MessageStatus
	MarksUnread     bool
	EventTimestamp  time.Time
}

type MessageThreadStatusUpdate struct {
	IsArchived *bool
	IsRead     *bool
	ReadAt     time.Time
}

type MessageThreadDeletedUpdate struct {
	MessageThreadID         uuid.UUID
	UserID                  entities.UserID
	LastMessageID           *uuid.UUID
	LastMessageContent      *string
	LastMessageStatus       entities.MessageStatus
}
```

Add the `time` import. Replace `Update` and the old deleted-message method in the interface with:

```go
UpdateActivity(ctx context.Context, params MessageThreadActivityUpdate) error
UpdateStatus(ctx context.Context, userID entities.UserID, messageThreadID uuid.UUID, params MessageThreadStatusUpdate) error
UpdateAfterDeletedMessage(ctx context.Context, params MessageThreadDeletedUpdate) error
```

- [ ] **Step 4: Implement field-scoped GORM updates**

Add these helpers to `gorm_message_thread_repository.go`:

```go
func messageThreadActivityUpdates(params MessageThreadActivityUpdate) map[string]any {
	return map[string]any{
		"order_timestamp":      params.Timestamp,
		"last_message_id":      params.MessageID,
		"last_message_content": params.Content,
		"status":               params.Status,
	}
}

func messageThreadStatusUpdates(params MessageThreadStatusUpdate) map[string]any {
	updates := make(map[string]any)
	if params.IsArchived != nil {
		updates["is_archived"] = *params.IsArchived
	}
	if params.IsRead != nil {
		updates["is_read"] = *params.IsRead
		if *params.IsRead {
			updates["last_read_at"] = params.ReadAt
		}
	}
	return updates
}
```

Replace the full-row `Update` implementation with:

```go
func (repository *gormMessageThreadRepository) UpdateActivity(ctx context.Context, params MessageThreadActivityUpdate) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&entities.MessageThread{}).
			Where("user_id = ?", params.UserID).
			Where("id = ?", params.MessageThreadID)

		result := query.Updates(messageThreadActivityUpdates(params))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return stacktrace.PropagateWithCode(
				gorm.ErrRecordNotFound,
				ErrCodeNotFound,
				fmt.Sprintf("thread with id [%s] not found", params.MessageThreadID),
			)
		}

		if !params.MarksUnread {
			return nil
		}

		return tx.Model(&entities.MessageThread{}).
			Where("user_id = ?", params.UserID).
			Where("id = ?", params.MessageThreadID).
			Where("last_read_at < ?", params.EventTimestamp).
			Update("is_read", false).
			Error
	})
	if err != nil {
		msg := fmt.Sprintf("cannot update message activity for thread [%s]", params.MessageThreadID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}
	return nil
}

func (repository *gormMessageThreadRepository) UpdateStatus(
	ctx context.Context,
	userID entities.UserID,
	messageThreadID uuid.UUID,
	params MessageThreadStatusUpdate,
) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	result := repository.db.WithContext(ctx).
		Model(&entities.MessageThread{}).
		Where("user_id = ?", userID).
		Where("id = ?", messageThreadID).
		Updates(messageThreadStatusUpdates(params))
	if result.Error != nil {
		msg := fmt.Sprintf("cannot update status for thread [%s]", messageThreadID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(result.Error, msg))
	}
	if result.RowsAffected == 0 {
		msg := fmt.Sprintf("thread with id [%s] not found", messageThreadID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(gorm.ErrRecordNotFound, ErrCodeNotFound, msg))
	}
	return nil
}

func (repository *gormMessageThreadRepository) UpdateAfterDeletedMessage(ctx context.Context, params MessageThreadDeletedUpdate) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	result := repository.db.WithContext(ctx).
		Model(&entities.MessageThread{}).
		Where("user_id = ?", params.UserID).
		Where("id = ?", params.MessageThreadID).
		Updates(map[string]any{
			"last_message_id":      params.LastMessageID,
			"last_message_content": params.LastMessageContent,
			"status":               params.LastMessageStatus,
		})
	if result.Error != nil {
		msg := fmt.Sprintf("cannot update deleted-message metadata for thread [%s]", params.MessageThreadID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(result.Error, msg))
	}
	return nil
}
```

Delete the old `Save(thread)` method and the old unused `UpdateAfterDeletedMessage(userID, messageID)` implementation.

- [ ] **Step 5: Run repository tests**

Run:

```powershell
go test ./pkg/repositories -run MessageThread -v
```

Expected: PASS.

- [ ] **Step 6: Commit atomic persistence**

```powershell
git add api/pkg/repositories/message_thread_repository.go api/pkg/repositories/gorm_message_thread_repository.go api/pkg/repositories/gorm_message_thread_repository_test.go
@'
refactor(api): make thread updates atomic

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
'@ | git commit -F -
```

### Task 3: Apply read rules in services and listeners

**Files:**

- Modify: `api/pkg/services/message_thread_service.go`
- Create: `api/pkg/services/message_thread_service_test.go`
- Modify: `api/pkg/listeners/message_thread_listener.go`
- Create: `api/pkg/listeners/read_receipts_test_helpers_test.go`
- Create: `api/pkg/listeners/message_thread_listener_test.go`
- Modify: `api/pkg/listeners/websocket_listener.go`
- Create: `api/pkg/listeners/websocket_listener_test.go`

**Interfaces:**

- Consumes: repository update types and methods from Task 2.
- Produces: `MessageThreadUpdateParams.MarksUnread bool`
- Produces: `MessageThreadUpdateParams.EventTimestamp time.Time`
- Produces: missed-call thread and websocket listeners.

- [ ] **Step 1: Write failing service behavior tests**

Create `api/pkg/services/message_thread_service_test.go`. Implement a repository stub with all interface methods returning zero values by default and function hooks for the methods under test:

```go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type messageThreadRepositoryStub struct {
	loadByOwnerContact func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error)
	load               func(context.Context, entities.UserID, uuid.UUID) (*entities.MessageThread, error)
	store              func(context.Context, *entities.MessageThread) error
	updateActivity     func(context.Context, repositories.MessageThreadActivityUpdate) error
	updateStatus       func(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) error
}

func (stub *messageThreadRepositoryStub) Store(ctx context.Context, thread *entities.MessageThread) error {
	if stub.store != nil {
		return stub.store(ctx, thread)
	}
	return nil
}

func (stub *messageThreadRepositoryStub) UpdateActivity(ctx context.Context, params repositories.MessageThreadActivityUpdate) error {
	if stub.updateActivity != nil {
		return stub.updateActivity(ctx, params)
	}
	return nil
}

func (stub *messageThreadRepositoryStub) UpdateStatus(ctx context.Context, userID entities.UserID, threadID uuid.UUID, params repositories.MessageThreadStatusUpdate) error {
	if stub.updateStatus != nil {
		return stub.updateStatus(ctx, userID, threadID, params)
	}
	return nil
}

func (stub *messageThreadRepositoryStub) UpdateAfterDeletedMessage(context.Context, repositories.MessageThreadDeletedUpdate) error {
	return nil
}

func (stub *messageThreadRepositoryStub) LoadByOwnerContact(ctx context.Context, userID entities.UserID, owner string, contact string) (*entities.MessageThread, error) {
	return stub.loadByOwnerContact(ctx, userID, owner, contact)
}

func (stub *messageThreadRepositoryStub) Load(ctx context.Context, userID entities.UserID, id uuid.UUID) (*entities.MessageThread, error) {
	return stub.load(ctx, userID, id)
}

func (stub *messageThreadRepositoryStub) Index(context.Context, entities.UserID, string, bool, repositories.IndexParams) (*[]entities.MessageThread, error) {
	threads := []entities.MessageThread{}
	return &threads, nil
}

func (stub *messageThreadRepositoryStub) Delete(context.Context, entities.UserID, uuid.UUID) error {
	return nil
}

func (stub *messageThreadRepositoryStub) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}

func newMessageThreadServiceForTest(repository repositories.MessageThreadRepository) *MessageThreadService {
	logger := &noopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewMessageThreadService(logger, tracer, repository, nil)
}
```

Add these tests below the stub:

```go
func TestUpdateThreadPassesUnreadWatermarkForInboundActivity(t *testing.T) {
	threadID := uuid.New()
	eventTimestamp := time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC)
	var captured repositories.MessageThreadActivityUpdate
	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{ID: threadID}, nil
		},
		updateActivity: func(_ context.Context, params repositories.MessageThreadActivityUpdate) error {
			captured = params
			return nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
		UserID:         entities.UserID("user-id"),
		Owner:          "+18005550199",
		Contact:        "+18005550100",
		MessageID:      uuid.New(),
		Content:        "hello",
		Status:         entities.MessageStatusReceived,
		Timestamp:      eventTimestamp,
		MarksUnread:    true,
		EventTimestamp: eventTimestamp,
	})

	require.NoError(t, err)
	assert.True(t, captured.MarksUnread)
	assert.Equal(t, eventTimestamp, captured.EventTimestamp)
}

func TestUpdateThreadPreservesReadStateForOutboundActivity(t *testing.T) {
	var captured repositories.MessageThreadActivityUpdate
	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{ID: uuid.New(), IsRead: false}, nil
		},
		updateActivity: func(_ context.Context, params repositories.MessageThreadActivityUpdate) error {
			captured = params
			return nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
		UserID:    entities.UserID("user-id"),
		Owner:     "+18005550199",
		Contact:   "+18005550100",
		MessageID: uuid.New(),
		Content:   "outbound",
		Status:    entities.MessageStatusSent,
		Timestamp: time.Now().UTC(),
	})

	require.NoError(t, err)
	assert.False(t, captured.MarksUnread)
}

func TestCreateThreadSetsReadStateFromActivityDirection(t *testing.T) {
	tests := []struct {
		name        string
		marksUnread bool
		wantRead    bool
	}{
		{name: "inbound", marksUnread: true, wantRead: false},
		{name: "outbound", marksUnread: false, wantRead: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stored *entities.MessageThread
			repository := &messageThreadRepositoryStub{
				loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
					return nil, stacktrace.PropagateWithCode(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
				},
				store: func(_ context.Context, thread *entities.MessageThread) error {
					stored = thread
					return nil
				},
			}

			service := newMessageThreadServiceForTest(repository)
			err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
				UserID:      entities.UserID("user-id"),
				Owner:       "+18005550199",
				Contact:     "+18005550100",
				MessageID:   uuid.New(),
				Content:     "hello",
				Status:      entities.MessageStatusReceived,
				Timestamp:   time.Now().UTC(),
				MarksUnread: test.marksUnread,
			})

			require.NoError(t, err)
			require.NotNil(t, stored)
			assert.Equal(t, test.wantRead, stored.IsRead)
			assert.False(t, stored.LastReadAt.IsZero())
		})
	}
}

func TestUpdateStatusChangesOnlyRequestedState(t *testing.T) {
	threadID := uuid.New()
	isRead := true
	var captured repositories.MessageThreadStatusUpdate
	repository := &messageThreadRepositoryStub{
		updateStatus: func(_ context.Context, _ entities.UserID, _ uuid.UUID, params repositories.MessageThreadStatusUpdate) error {
			captured = params
			return nil
		},
		load: func(context.Context, entities.UserID, uuid.UUID) (*entities.MessageThread, error) {
			return &entities.MessageThread{ID: threadID, IsArchived: true, IsRead: true}, nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	thread, err := service.UpdateStatus(context.Background(), MessageThreadStatusParams{
		UserID:          entities.UserID("user-id"),
		MessageThreadID: threadID,
		IsRead:          &isRead,
	})

	require.NoError(t, err)
	assert.Nil(t, captured.IsArchived)
	assert.Same(t, &isRead, captured.IsRead)
	assert.False(t, captured.ReadAt.IsZero())
	assert.True(t, thread.IsArchived)
}

func TestUpdateStatusPreservesNotFoundCode(t *testing.T) {
	repository := &messageThreadRepositoryStub{
		updateStatus: func(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) error {
			return stacktrace.PropagateWithCode(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
		},
	}

	service := newMessageThreadServiceForTest(repository)
	isRead := true
	_, err := service.UpdateStatus(context.Background(), MessageThreadStatusParams{
		UserID:          entities.UserID("user-id"),
		MessageThreadID: uuid.New(),
		IsRead:          &isRead,
	})

	assert.Equal(t, repositories.ErrCodeNotFound, stacktrace.GetCode(err))
}
```

- [ ] **Step 2: Run service tests and confirm they fail**

Run:

```powershell
go test ./pkg/services -run 'Test(UpdateThread|CreateThread|UpdateStatus)' -v
```

Expected: compile failures because the new update parameters and repository calls are not wired.

- [ ] **Step 3: Implement service read rules**

Extend `MessageThreadUpdateParams`:

```go
MarksUnread    bool
EventTimestamp time.Time
```

Replace the existing full-row update in `UpdateThread` with:

```go
if err = service.repository.UpdateActivity(ctx, repositories.MessageThreadActivityUpdate{
	MessageThreadID: thread.ID,
	UserID:          params.UserID,
	Timestamp:       params.Timestamp,
	MessageID:       params.MessageID,
	Content:         params.Content,
	Status:          params.Status,
	MarksUnread:     params.MarksUnread,
	EventTimestamp:  params.EventTimestamp,
}); err != nil {
	msg := fmt.Sprintf("cannot update message thread with id [%s] after adding message [%s]", thread.ID, params.MessageID)
	return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
}
```

In `createThread`, create one `now := time.Now().UTC()` and initialize:

```go
IsRead:     !params.MarksUnread,
LastReadAt: now,
CreatedAt:  now,
UpdatedAt:  now,
```

Replace `UpdateStatus` with a partial repository update followed by a reload:

```go
func (service *MessageThreadService) UpdateStatus(ctx context.Context, params MessageThreadStatusParams) (*entities.MessageThread, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	update := repositories.MessageThreadStatusUpdate{
		IsArchived: params.IsArchived,
		IsRead:     params.IsRead,
		ReadAt:     time.Now().UTC(),
	}
	if err := service.repository.UpdateStatus(ctx, params.UserID, params.MessageThreadID, update); err != nil {
		msg := fmt.Sprintf("cannot update message thread with id [%s]", params.MessageThreadID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	thread, err := service.repository.Load(ctx, params.UserID, params.MessageThreadID)
	if err != nil {
		msg := fmt.Sprintf("cannot reload message thread with id [%s]", params.MessageThreadID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}
	return thread, nil
}
```

In `UpdateAfterDeletedMessage`, replace the full-row update with:

```go
if err = service.repository.UpdateAfterDeletedMessage(ctx, repositories.MessageThreadDeletedUpdate{
	MessageThreadID:    thread.ID,
	UserID:             thread.UserID,
	LastMessageID:      payload.PreviousMessageID,
	LastMessageContent: payload.PreviousMessageContent,
	LastMessageStatus:  *payload.PreviousMessageStatus,
}); err != nil {
	msg := fmt.Sprintf("cannot update thread with ID [%s] for user with ID [%s]", thread.ID, thread.UserID)
	return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
}
```

- [ ] **Step 4: Add inbound and missed-call listener behavior**

In `NewMessageThreadListener`, register:

```go
events.MessageCallMissed: l.OnMessageCallMissed,
```

In `OnMessagePhoneReceived`, add:

```go
MarksUnread:    true,
EventTimestamp: event.Time(),
```

Add:

```go
func (listener *MessageThreadListener) OnMessageCallMissed(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageCallMissedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	params := services.MessageThreadUpdateParams{
		Owner:          payload.Owner,
		Contact:        payload.Contact,
		UserID:         payload.UserID,
		Status:         entities.MessageStatusReceived,
		Timestamp:      payload.Timestamp,
		Content:        "Missed phone call",
		MessageID:      payload.MessageID,
		MarksUnread:    true,
		EventTimestamp: event.Time(),
	}
	if err := listener.service.UpdateThread(ctx, params); err != nil {
		msg := fmt.Sprintf("cannot update thread for missed call [%s] on event [%s]", payload.MessageID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}
```

In `NewWebsocketListener`, register:

```go
events.MessageCallMissed: l.onMessageCallMissed,
```

Add:

```go
func (listener *WebsocketListener) onMessageCallMissed(ctx context.Context, event cloudevents.Event) error {
	ctx, span, _ := listener.tracer.StartWithLogger(ctx, listener.logger)
	defer span.End()

	var payload events.MessageCallMissedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.client.Trigger(payload.UserID.String(), event.Type(), event.ID()); err != nil {
		msg := fmt.Sprintf("cannot trigger websocket [%s] event with ID [%s] for user with ID [%s]", event.Type(), event.ID(), payload.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}
```

- [ ] **Step 5: Add listener tests**

Create `api/pkg/listeners/read_receipts_test_helpers_test.go`:

```go
package listeners

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type noopListenerLogger struct{}

func (logger *noopListenerLogger) Error(error)                                 {}
func (logger *noopListenerLogger) WithService(string) telemetry.Logger         { return logger }
func (logger *noopListenerLogger) WithString(string, string) telemetry.Logger  { return logger }
func (logger *noopListenerLogger) WithSpan(trace.SpanContext) telemetry.Logger { return logger }
func (logger *noopListenerLogger) Trace(string)                                {}
func (logger *noopListenerLogger) Info(string)                                 {}
func (logger *noopListenerLogger) Warn(error)                                  {}
func (logger *noopListenerLogger) Debug(string)                                {}
func (logger *noopListenerLogger) Fatal(error)                                 {}
func (logger *noopListenerLogger) Printf(string, ...interface{})               {}

type listenerMessageThreadRepository struct {
	activity repositories.MessageThreadActivityUpdate
}

func (repository *listenerMessageThreadRepository) Store(context.Context, *entities.MessageThread) error {
	return nil
}

func (repository *listenerMessageThreadRepository) UpdateActivity(_ context.Context, params repositories.MessageThreadActivityUpdate) error {
	repository.activity = params
	return nil
}

func (repository *listenerMessageThreadRepository) UpdateStatus(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) error {
	return nil
}

func (repository *listenerMessageThreadRepository) UpdateAfterDeletedMessage(context.Context, repositories.MessageThreadDeletedUpdate) error {
	return nil
}

func (repository *listenerMessageThreadRepository) LoadByOwnerContact(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
	return &entities.MessageThread{ID: uuid.New()}, nil
}

func (repository *listenerMessageThreadRepository) Load(context.Context, entities.UserID, uuid.UUID) (*entities.MessageThread, error) {
	return &entities.MessageThread{}, nil
}

func (repository *listenerMessageThreadRepository) Index(context.Context, entities.UserID, string, bool, repositories.IndexParams) (*[]entities.MessageThread, error) {
	threads := []entities.MessageThread{}
	return &threads, nil
}

func (repository *listenerMessageThreadRepository) Delete(context.Context, entities.UserID, uuid.UUID) error {
	return nil
}

func (repository *listenerMessageThreadRepository) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}
```

Create `api/pkg/listeners/message_thread_listener_test.go`:

```go
package listeners

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageThreadListenerMarksMissedCallUnread(t *testing.T) {
	repository := &listenerMessageThreadRepository{}
	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	service := services.NewMessageThreadService(logger, tracer, repository, nil)
	_, routes := NewMessageThreadListener(logger, tracer, service)

	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource("/v1/messages/call-missed")
	event.SetType(events.MessageCallMissed)
	event.SetTime(time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC))
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.MessageCallMissedPayload{
		MessageID: uuid.New(),
		UserID:    entities.UserID("user-id"),
		Owner:     "+18005550199",
		Contact:   "+18005550100",
		Timestamp: time.Date(2026, 7, 18, 6, 59, 0, 0, time.UTC),
	}))

	err := routes[events.MessageCallMissed](context.Background(), event)

	require.NoError(t, err)
	assert.True(t, repository.activity.MarksUnread)
	assert.Equal(t, "Missed phone call", repository.activity.Content)
	assert.Equal(t, event.Time(), repository.activity.EventTimestamp)
}
```

Create `api/pkg/listeners/websocket_listener_test.go`:

```go
package listeners

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/pusher/pusher-http-go/v5"
	"github.com/stretchr/testify/assert"
)

func TestWebsocketListenerRegistersMissedCalls(t *testing.T) {
	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	_, routes := NewWebsocketListener(logger, tracer, &pusher.Client{})

	assert.Contains(t, routes, events.MessageCallMissed)
}
```

- [ ] **Step 6: Run service and listener tests**

Run:

```powershell
go test ./pkg/services ./pkg/listeners -run 'MessageThread|MissedCall|UpdateStatus|CreateThread' -v
```

Expected: PASS.

- [ ] **Step 7: Commit service and realtime behavior**

```powershell
git add api/pkg/services/message_thread_service.go api/pkg/services/message_thread_service_test.go api/pkg/listeners/message_thread_listener.go api/pkg/listeners/read_receipts_test_helpers_test.go api/pkg/listeners/message_thread_listener_test.go api/pkg/listeners/websocket_listener.go api/pkg/listeners/websocket_listener_test.go
@'
feat(api): track unread inbound threads

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
'@ | git commit -F -
```

### Task 4: Finish the HTTP behavior and regenerate Swagger

**Files:**

- Modify: `api/pkg/handlers/message_thread_handler.go`
- Modify: `api/docs/docs.go`
- Modify: `api/docs/swagger.json`
- Modify: `api/docs/swagger.yaml`

**Interfaces:**

- Consumes: `responses.MessageThreadResponse` from Task 1.
- Produces: 404 behavior for missing thread updates.
- Produces: Swagger schema fields `is_read`, optional `is_archived`, optional `is_read`.

- [ ] **Step 1: Update the handler error mapping and annotations**

Change the update annotation:

```go
// @Success      200 				{object}	responses.MessageThreadResponse
// @Failure 	 404				{object}	responses.NotFound
```

Change the error block:

```go
thread, err := h.service.UpdateStatus(ctx, request.ToUpdateParams(h.userIDFomContext(c)))
if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
	return h.responseNotFound(c, fmt.Sprintf("cannot find message thread with ID [%s]", request.MessageThreadID))
}
if err != nil {
	msg := fmt.Sprintf("cannot update message thread with params [%+#v]", request)
	ctxLogger.Error(stacktrace.Propagate(err, msg))
	return h.responseInternalServerError(c)
}
```

- [ ] **Step 2: Regenerate Swagger**

Run:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts\api'
swag init --requiredByDefault --parseDependency --parseInternal
```

Expected: `api/docs/docs.go`, `api/docs/swagger.json`, and `api/docs/swagger.yaml` update successfully.

- [ ] **Step 3: Run the full API suite**

Run:

```powershell
go test -vet=off ./pkg/entities ./pkg/repositories ./pkg/requests ./pkg/services ./pkg/validators ./pkg/listeners
go test -vet=off ./pkg/handlers -run '^$'
```

Expected: PASS. Vet is disabled because clean `origin/main` has unrelated Go
1.25 non-constant-format vet failures. Handler tests are compile-checked only
because the pre-existing suite expects an API server on `localhost:8000`; Task
7 provides the HTTP integration coverage against the Docker stack.

- [ ] **Step 4: Commit HTTP and documentation changes**

```powershell
git add api/pkg/handlers/message_thread_handler.go api/docs/docs.go api/docs/swagger.json api/docs/swagger.yaml
@'
docs(api): publish thread read state

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
'@ | git commit -F -
```

### Task 5: Add automatic read updates to the web store

**Files:**

- Modify: `web/shared/types/api.ts`
- Modify: `web/app/stores/threads.ts`

**Interfaces:**

- Consumes: API `PUT /v1/message-threads/{id}` with `{ is_read: true }`.
- Produces: `markThreadRead(threadId: string, force?: boolean): Promise<void>`.

- [ ] **Step 1: Regenerate TypeScript API models**

Run:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts\web'
pnpm api:models
```

Expected: `EntitiesMessageThread.is_read: boolean` and optional fields on `RequestsMessageThreadUpdate`.

- [ ] **Step 2: Add a typed local replacement helper**

In `web/app/stores/threads.ts`, add:

```ts
function replaceThread(updatedThread: EntitiesMessageThread) {
  const index = threads.value.findIndex(
    (thread) => thread.id === updatedThread.id,
  );
  if (index !== -1) threads.value[index] = updatedThread;
}
```

- [ ] **Step 3: Implement non-silent automatic read persistence**

Add:

```ts
async function markThreadRead(threadId: string, force = false) {
  const thread = threads.value.find((item) => item.id === threadId);
  if (!thread) throw new Error(`Cannot find thread with id ${threadId}`);
  if (!force && thread.is_read) return;

  try {
    const response = await apiFetch<{ data: EntitiesMessageThread }>(
      `/v1/message-threads/${threadId}`,
      {
        method: "PUT",
        body: { is_read: true },
      },
    );
    replaceThread(response.data);
  } catch (error) {
    notificationsStore.addNotification({
      message: "The message thread could not be marked as read",
      type: "error",
    });
    await loadThreads();
    throw error;
  }
}
```

Add `markThreadRead` to the returned store API.

Keep archive updates independent:

```ts
body: { is_archived: payload.isArchived },
```

Do not send `is_read` from `updateThread`.

- [ ] **Step 4: Run web static checks**

Run:

```powershell
pnpm lint:js
pnpm lint:prettier
```

Expected: PASS.

- [ ] **Step 5: Commit store and generated types**

```powershell
git add web/shared/types/api.ts web/app/stores/threads.ts
@'
feat(web): persist opened threads as read

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
'@ | git commit -F -
```

### Task 6: Mark opened threads read and highlight unread threads

**Files:**

- Modify: `web/app/pages/threads/[id]/index.vue`
- Modify: `web/app/components/MessageThread.vue`

**Interfaces:**

- Consumes: `threadsStore.markThreadRead(threadId, force)`.
- Produces: automatic initial and realtime read updates.
- Produces: shared unread list styling on mobile and desktop.

- [ ] **Step 1: Make read updates independent from message loading**

In `web/app/pages/threads/[id]/index.vue`, add:

```ts
async function markCurrentThreadRead(force = false) {
  const threadId = route.params.id as string;
  try {
    await threadsStore.markThreadRead(threadId, force);
  } catch (error) {
    console.error(error);
  }
}
```

At the start of `loadMessages`, after computing `threadId`, call without awaiting:

```ts
void markCurrentThreadRead();
```

This lets message loading continue even if the read update fails.

- [ ] **Step 2: Force a newer read watermark after inbound realtime events**

Change the incoming SMS binding:

```ts
webhookChannel.bind("message.phone.received", () => {
  if (!loadingMessages.value) {
    void markCurrentThreadRead(true);
    loadMessages(false);
  }
});
```

Add the missed-call binding with the same behavior:

```ts
webhookChannel.bind("message.call.missed", () => {
  if (!loadingMessages.value) {
    void markCurrentThreadRead(true);
    loadMessages(false);
  }
});
```

The forced call is required because the local thread may still say `is_read:
true` when the concurrent backend listener has not committed its unread update.

- [ ] **Step 3: Add unread presentation**

In `MessageThread.vue`, add the class binding to `v-list-item`:

```vue
:class="{ 'message-thread--unread': !thread.is_read }"
```

Change title and subtitle class bindings:

```vue
<v-list-item-title :class="{ 'font-weight-bold': !thread.is_read }">
  {{ formatPhoneNumber(thread.contact) }}
</v-list-item-title>
<v-list-item-subtitle
  class="text-truncate mt-1"
  :class="{ 'font-weight-bold': !thread.is_read }"
  style="max-width: 250px"
>
```

Add:

```vue
<style scoped>
.message-thread--unread {
  background: rgba(var(--v-theme-primary), 0.1);
}
</style>
```

- [ ] **Step 4: Run web validation**

Run:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts\web'
pnpm lint
pnpm run generate
```

Expected: both commands PASS.

- [ ] **Step 5: Commit UI behavior**

```powershell
git add web/app/pages/threads/[id]/index.vue web/app/components/MessageThread.vue
@'
feat(web): highlight unread message threads

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
'@ | git commit -F -
```

### Task 7: Add integration coverage

**Files:**

- Create: `tests/read_receipts_test.go`
- Modify: `tests/README.md`

**Interfaces:**

- Consumes: thread index and update HTTP endpoints from Tasks 1-4.
- Consumes: incoming SMS, missed-call, and outbound message flows from Task 3.
- Produces: `TestMessageThreadReadReceipts`.

- [ ] **Step 1: Write the end-to-end read-receipts test**

Create `tests/read_receipts_test.go`:

```go
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	httpsms "github.com/NdoleStudio/httpsms-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type integrationMessageThread struct {
	ID                 string  `json:"id"`
	Contact            string  `json:"contact"`
	IsRead             bool    `json:"is_read"`
	LastMessageContent *string `json:"last_message_content"`
}

func requestJSON(
	ctx context.Context,
	t *testing.T,
	method string,
	path string,
	apiKey string,
	payload any,
	expectedStatus int,
	output any,
) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, apiBaseURL+path, body)
	require.NoError(t, err)
	request.Header.Set("x-api-key", apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, response.StatusCode, "response: %s", string(responseBody))

	if output != nil {
		require.NoError(t, json.Unmarshal(responseBody, output))
	}
}

func fetchMessageThreads(ctx context.Context, t *testing.T, owner string) []integrationMessageThread {
	t.Helper()

	var response struct {
		Data []integrationMessageThread `json:"data"`
	}
	path := fmt.Sprintf(
		"/v1/message-threads?owner=%s&skip=0&limit=20&is_archived=false",
		url.QueryEscape(owner),
	)
	requestJSON(ctx, t, http.MethodGet, path, userAPIKey, nil, http.StatusOK, &response)
	return response.Data
}

func waitForMessageThread(
	ctx context.Context,
	t *testing.T,
	owner string,
	contact string,
	timeout time.Duration,
	matches func(integrationMessageThread) bool,
) integrationMessageThread {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, thread := range fetchMessageThreads(ctx, t, owner) {
			if thread.Contact == contact && matches(thread) {
				return thread
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("thread %s -> %s did not reach the expected state within %v", owner, contact, timeout)
	return integrationMessageThread{}
}

func markMessageThreadRead(ctx context.Context, t *testing.T, threadID string) integrationMessageThread {
	t.Helper()

	var response struct {
		Data integrationMessageThread `json:"data"`
	}
	requestJSON(
		ctx,
		t,
		http.MethodPut,
		"/v1/message-threads/"+threadID,
		userAPIKey,
		map[string]any{"is_read": true},
		http.StatusOK,
		&response,
	)
	return response.Data
}

func TestMessageThreadReadReceipts(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)
	contact := randomPhoneNumber()

	requestJSON(
		ctx,
		t,
		http.MethodPost,
		"/v1/messages/receive",
		phone.PhoneAPIKey,
		map[string]any{
			"from":      contact,
			"to":        phone.PhoneNumber,
			"content":   "Unread inbound message",
			"encrypted": false,
			"sim":       "SIM1",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
		http.StatusOK,
		nil,
	)

	thread := waitForMessageThread(ctx, t, phone.PhoneNumber, contact, 20*time.Second, func(thread integrationMessageThread) bool {
		return !thread.IsRead
	})
	assert.False(t, thread.IsRead)

	updated := markMessageThreadRead(ctx, t, thread.ID)
	assert.True(t, updated.IsRead)
	waitForMessageThread(ctx, t, phone.PhoneNumber, contact, 10*time.Second, func(thread integrationMessageThread) bool {
		return thread.IsRead
	})

	requestJSON(
		ctx,
		t,
		http.MethodPost,
		"/v1/messages/calls/missed",
		phone.PhoneAPIKey,
		map[string]any{
			"from":      contact,
			"to":        phone.PhoneNumber,
			"sim":       "SIM1",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
		http.StatusOK,
		nil,
	)

	thread = waitForMessageThread(ctx, t, phone.PhoneNumber, contact, 20*time.Second, func(thread integrationMessageThread) bool {
		return !thread.IsRead &&
			thread.LastMessageContent != nil &&
			*thread.LastMessageContent == "Missed phone call"
	})
	assert.False(t, thread.IsRead)

	outboundContent := "Outbound activity preserves unread"
	client := newAPIClient()
	_, response, err := client.Messages.Send(ctx, &httpsms.MessageSendParams{
		From:    phone.PhoneNumber,
		To:      contact,
		Content: outboundContent,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)

	thread = waitForMessageThread(ctx, t, phone.PhoneNumber, contact, 20*time.Second, func(thread integrationMessageThread) bool {
		return thread.LastMessageContent != nil &&
			*thread.LastMessageContent == outboundContent
	})
	assert.False(t, thread.IsRead, "outbound activity must not clear unread state")
}
```

- [ ] **Step 2: Run the test before implementation is complete**

With the Docker integration stack running, execute:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts\tests'
go test -v -timeout 120s -run TestMessageThreadReadReceipts ./...
```

Expected before the feature implementation: FAIL because thread responses do not expose/persist `is_read`.

- [ ] **Step 3: Update integration-test documentation**

Add to the Test Coverage checklist in `tests/README.md`:

```markdown
- [x] **Message thread read receipts E2E** — Incoming SMS and missed calls mark a thread unread, the existing thread update endpoint marks it read, and outbound activity preserves unread state
```

- [ ] **Step 4: Run the integration test against the completed stack**

Generate credentials, start the Docker stack, run the focused test, and tear down:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts\tests'
& 'C:\Program Files\Git\bin\bash.exe' generate-firebase-credentials.sh
$env:FIREBASE_CREDENTIALS = (Get-Content firebase-credentials.json -Raw | ConvertFrom-Json | ConvertTo-Json -Compress)
docker compose up -d --build --wait
docker compose wait seed
Start-Sleep -Seconds 2
try {
  go test -v -timeout 120s -run TestMessageThreadReadReceipts ./...
} finally {
  docker compose down -v
}
```

Expected: PASS.

- [ ] **Step 5: Commit integration coverage**

```powershell
git add tests/read_receipts_test.go tests/README.md
@'
test: cover message thread read receipts

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
'@ | git commit -F -
```

### Task 8: Verify the complete feature

**Files:**

- Review: all files changed since `origin/main`

**Interfaces:**

- Consumes: all previous tasks.
- Produces: a clean, verified feature branch.

- [ ] **Step 1: Run API formatting and tests**

Run:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts\api'
go-fumpt -w pkg/entities/message_thread.go pkg/entities/message_thread_test.go pkg/repositories/message_thread_repository.go pkg/repositories/gorm_message_thread_repository.go pkg/repositories/gorm_message_thread_repository_test.go pkg/services/message_thread_service.go pkg/services/message_thread_service_test.go pkg/requests/message_thread_update_request.go pkg/requests/message_thread_update_request_test.go pkg/validators/message_thread_handler_validator.go pkg/validators/message_thread_handler_validator_test.go pkg/responses/message_thead_responses.go pkg/handlers/message_thread_handler.go pkg/listeners/message_thread_listener.go pkg/listeners/read_receipts_test_helpers_test.go pkg/listeners/message_thread_listener_test.go pkg/listeners/websocket_listener.go pkg/listeners/websocket_listener_test.go
go test -vet=off ./pkg/entities ./pkg/repositories ./pkg/requests ./pkg/services ./pkg/validators ./pkg/listeners
go test -vet=off ./pkg/handlers -run '^$'
```

Expected: formatting completes and all API tests PASS.

- [ ] **Step 2: Run complete web validation**

Run:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts\web'
pnpm lint
pnpm run generate
```

Expected: PASS.

- [ ] **Step 3: Inspect the final diff**

Run:

```powershell
Set-Location 'C:\Users\achoa\Work\NdoleStudio\httpsms-read-receipts'
git --no-pager diff --check origin/main...HEAD
git --no-pager diff --stat origin/main...HEAD
git --no-pager status --short --branch
```

Expected: no whitespace errors and no uncommitted implementation changes.

- [ ] **Step 4: Perform manual acceptance checks against a migrated local database**

Verify:

1. Existing rows return `"is_read": true`.
2. A closed thread becomes highlighted after an incoming SMS.
3. A closed thread becomes highlighted after a missed call.
4. Opening the thread removes the highlight and refresh keeps it read.
5. Incoming activity while the thread is open does not leave it unread.
6. Sending and delivery-status changes do not clear an unread thread.
7. Archiving and unarchiving preserve read state.

- [ ] **Step 5: Commit any formatting-only corrections**

Skip this step when Step 1 produced no changes. Otherwise:

```powershell
git add api web
@'
style: format read receipts changes

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Copilot-Session: bf00a0ac-e11f-4015-b295-3cdd9b491229
'@ | git commit -F -
```
