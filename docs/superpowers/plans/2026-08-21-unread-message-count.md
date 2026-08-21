# Message Thread Unread Count Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace binary thread read state with an idempotent unread-item count for received SMS messages and missed calls.

**Architecture:** Store `unread_count` on each message thread for cheap list reads and maintain an internal message-ID ledger so retried inbound events cannot double-count. Repository transactions lock the thread and update activity, ledger, count, and read watermark atomically; the existing update endpoint permits clients only to reset the count to zero.

**Tech Stack:** Go, Fiber, GORM, PostgreSQL/CockroachDB, CloudEvents, Testify, Nuxt 4, Vue 3, Pinia, Vuetify, TypeScript.

## Global Constraints

- `unread_count` is the sole public unread-state field; remove `is_read` from API and UI contracts.
- Received SMS messages and missed calls each increment once; outbound/status events do not change the count.
- `PUT /v1/message-threads/{id}` accepts only client value `unread_count: 0`.
- Existing `is_read=false` rows migrate to `unread_count=1`; existing read rows migrate to zero.
- Preserve `last_read_at` as an internal race-resolution watermark.
- Deleting a counted unread item decrements once without a preliminary lookup.
- Badge values are exact through 99 and display `99+` above 99.
- Use GORM with context propagation and `stacktrace.Propagate`; do not introduce raw SQL.
- Format Go with gofumpt and web code with the existing lint configuration.

---

## File Structure

### New files

- `api/pkg/entities/message_thread_unread_item.go`: internal ledger entity keyed by message ID.
- `api/pkg/migrations/message_thread_unread_count.go`: idempotent schema/backfill transition from `is_read`.
- `api/pkg/migrations/message_thread_unread_count_test.go`: migration decision/helper coverage.

### Modified API files

- `api/pkg/entities/message_thread.go`: replace `IsRead` with `UnreadCount`.
- `api/pkg/entities/message_thread_test.go`: assert count schema/public contract.
- `api/pkg/di/container.go`: run the unread-count migration and ledger auto-migration.
- `api/pkg/repositories/message_thread_repository.go`: rename count intent and define ledger-aware update inputs.
- `api/pkg/repositories/gorm_message_thread_repository.go`: atomic ledger/counter/reset/deletion transactions.
- `api/pkg/repositories/gorm_message_thread_repository_test.go`: helper SQL ownership, idempotency intent, reset, and deletion tests.
- `api/pkg/services/message_thread_service.go`: pass count intent, initialize new counts, and process non-last deletions.
- `api/pkg/services/message_thread_service_test.go`: service contract and deletion routing tests.
- `api/pkg/listeners/message_thread_listener.go`: mark received SMS and missed calls as countable.
- `api/pkg/listeners/message_thread_listener_test.go`: listener count-intent tests.
- `api/pkg/listeners/read_receipts_test_helpers_test.go`: update repository test stub signatures.
- `api/pkg/requests/message_thread_update_request.go`: replace `is_read` with optional `unread_count`.
- `api/pkg/requests/message_thread_update_request_test.go`: conversion tests.
- `api/pkg/validators/message_thread_handler_validator.go`: require archive/reset and reject nonzero counts.
- `api/pkg/validators/message_thread_handler_validator_test.go`: request validation tests.
- `api/pkg/handlers/message_thread_handler_test.go`: response and removed-contract tests.
- `api/docs/docs.go`, `api/docs/swagger.json`, `api/docs/swagger.yaml`: regenerated API contract.

### Modified web and integration files

- `web/shared/types/api.ts`: regenerated `unread_count` types.
- `web/app/stores/threads.ts`: reset count and use zero/nonzero state.
- `web/app/pages/threads/[id]/index.vue`: rename mark-read functions to count reset.
- `web/app/components/MessageThread.vue`: numeric badge and `unread_count > 0` styling.
- `tests/read_receipts_test.go`: count-based end-to-end coverage.
- `tests/README.md`: describe unread-count integration coverage.

---

### Task 1: Add unread-count schema and migration

**Files:**
- Create: `api/pkg/entities/message_thread_unread_item.go`
- Create: `api/pkg/migrations/message_thread_unread_count.go`
- Create: `api/pkg/migrations/message_thread_unread_count_test.go`
- Modify: `api/pkg/entities/message_thread.go`
- Modify: `api/pkg/entities/message_thread_test.go`
- Modify: `api/pkg/di/container.go`

**Interfaces:**
- Produces: `entities.MessageThread.UnreadCount uint`
- Produces: `entities.MessageThreadUnreadItem{MessageID, MessageThreadID}`
- Produces: `migrations.MigrateMessageThreadUnreadCount(db *gorm.DB) error`

- [ ] **Step 1: Replace the entity test with count and ledger schema assertions**

```go
func TestMessageThreadUnreadFields(t *testing.T) {
	threadType := reflect.TypeOf(MessageThread{})
	_, hasIsRead := threadType.FieldByName("IsRead")
	assert.False(t, hasIsRead)

	unreadCount, ok := threadType.FieldByName("UnreadCount")
	require.True(t, ok)
	assert.Equal(t, "unread_count", unreadCount.Tag.Get("json"))
	assert.Contains(t, unreadCount.Tag.Get("gorm"), "not null")
	assert.Contains(t, unreadCount.Tag.Get("gorm"), "default:0")

	lastReadAt, ok := threadType.FieldByName("LastReadAt")
	require.True(t, ok)
	assert.Equal(t, "-", lastReadAt.Tag.Get("json"))
}

func TestMessageThreadUnreadItemUsesMessageIDAsPrimaryKey(t *testing.T) {
	itemType := reflect.TypeOf(MessageThreadUnreadItem{})
	messageID, ok := itemType.FieldByName("MessageID")
	require.True(t, ok)
	assert.Contains(t, messageID.Tag.Get("gorm"), "primaryKey")
}
```

- [ ] **Step 2: Run the entity tests and verify failure**

Run:

```bash
cd api
go test ./pkg/entities -run 'TestMessageThreadUnread' -count=1
```

Expected: FAIL because `UnreadCount` and `MessageThreadUnreadItem` do not exist.

- [ ] **Step 3: Add the count and ledger entities**

```go
// MessageThread fields
IsArchived bool      `json:"is_archived" example:"false"`
UnreadCount uint     `json:"unread_count" gorm:"not null;default:0" example:"2"`
LastReadAt time.Time `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`
```

```go
package entities

import "github.com/google/uuid"

// MessageThreadUnreadItem records an inbound item currently counted as unread.
type MessageThreadUnreadItem struct {
	MessageID       uuid.UUID     `gorm:"primaryKey;type:uuid"`
	MessageThreadID uuid.UUID     `gorm:"not null;type:uuid;index"`
	MessageThread   MessageThread `gorm:"constraint:OnDelete:CASCADE;"`
}
```

- [ ] **Step 4: Add an idempotent GORM migration**

Implement `MigrateMessageThreadUnreadCount` so it:

```go
func MigrateMessageThreadUnreadCount(db *gorm.DB) error {
	if err := db.AutoMigrate(&entities.MessageThread{}, &entities.MessageThreadUnreadItem{}); err != nil {
		return stacktrace.Propagate(err, "cannot migrate message thread unread count schema")
	}
	if !db.Migrator().HasColumn("message_threads", "is_read") {
		return nil
	}
	if err := db.Table("message_threads").
		Where("is_read = ?", false).
		Where("unread_count = ?", 0).
		Update("unread_count", 1).Error; err != nil {
		return stacktrace.Propagate(err, "cannot backfill message thread unread counts")
	}
	if err := db.Migrator().DropColumn("message_threads", "is_read"); err != nil {
		return stacktrace.Propagate(err, "cannot drop legacy message thread is_read column")
	}
	return nil
}
```

Add helper-level tests proving the migration skips the backfill when the legacy
column is absent and propagates migration errors; use the repository's existing
GORM fake-connection pattern rather than a new test dependency.

- [ ] **Step 5: Wire the migration into the DI container**

Replace the direct `AutoMigrate(&entities.MessageThread{})` call with:

```go
if err = migrations.MigrateMessageThreadUnreadCount(db); err != nil {
	container.logger.Fatal(stacktrace.Propagate(err, "cannot migrate message thread unread counts"))
}
```

- [ ] **Step 6: Run focused tests and format**

Run:

```bash
cd api
gofumpt -w pkg/entities/message_thread.go pkg/entities/message_thread_unread_item.go pkg/entities/message_thread_test.go pkg/migrations/message_thread_unread_count.go pkg/migrations/message_thread_unread_count_test.go pkg/di/container.go
go test ./pkg/entities ./pkg/migrations -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add api/pkg/entities api/pkg/migrations api/pkg/di/container.go
git commit -m "feat(api): add unread count schema"
```

---

### Task 2: Implement ledger-backed repository updates

**Files:**
- Modify: `api/pkg/repositories/message_thread_repository.go`
- Modify: `api/pkg/repositories/gorm_message_thread_repository.go`
- Modify: `api/pkg/repositories/gorm_message_thread_repository_test.go`

**Interfaces:**
- Produces: `MessageThreadActivityUpdate.CountAsUnread bool`
- Produces: `MessageThreadStatusUpdate.UnreadCount *uint`
- Produces: `MessageThreadDeletedUpdate.DeletedMessageID uuid.UUID`
- Consumes: `entities.MessageThreadUnreadItem`

- [ ] **Step 1: Write failing repository contract/helper tests**

Add tests that assert:

```go
func TestMessageThreadStatusUpdatesResetUnreadCount(t *testing.T) {
	zero := uint(0)
	readAt := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		UnreadCount: &zero,
		ReadAt:      readAt,
	})
	assert.Equal(t, map[string]any{
		"unread_count": 0,
		"last_read_at": readAt,
	}, updates)
}

func TestMessageThreadActivityUpdatesDoNotOwnUnreadColumns(t *testing.T) {
	updates := messageThreadActivityUpdates(MessageThreadActivityUpdate{ /* activity fields */ })
	assert.NotContains(t, updates, "unread_count")
	assert.NotContains(t, updates, "last_read_at")
}
```

Also add transaction tests using the existing fake connection to verify that:

- countable activity emits a ledger insert and count increment;
- duplicate ledger insert (`RowsAffected == 0`) does not increment;
- reset deletes ledger rows and updates `last_read_at`;
- deletion decrements only when ledger deletion affects one row;
- decrement uses `GREATEST(unread_count - 1, 0)`.

- [ ] **Step 2: Run repository tests and verify failure**

Run:

```bash
cd api
go test ./pkg/repositories -run 'TestMessageThread(Activity|Status|Unread|Deleted)' -count=1
```

Expected: FAIL on missing count fields and ledger behavior.

- [ ] **Step 3: Update repository input types**

```go
type MessageThreadActivityUpdate struct {
	MessageThreadID uuid.UUID
	UserID          entities.UserID
	Timestamp       time.Time
	MessageID       uuid.UUID
	Content         string
	Status          entities.MessageStatus
	CountAsUnread   bool
	EventTimestamp  time.Time
	Unarchive       bool
}

type MessageThreadStatusUpdate struct {
	IsArchived  *bool
	UnreadCount *uint
	ReadAt      time.Time
}

type MessageThreadDeletedUpdate struct {
	MessageThreadID    uuid.UUID
	UserID             entities.UserID
	DeletedMessageID   uuid.UUID
	UpdateLastMessage  bool
	LastMessageID      *uuid.UUID
	LastMessageContent *string
	LastMessageStatus  entities.MessageStatus
}
```

- [ ] **Step 4: Add shared lock and ledger helpers**

Use `clause.Locking{Strength: "UPDATE"}` with `WithContext(ctx)` and always
scope the thread by `user_id` and ID. Add private helpers that accept the
transaction:

```go
func lockMessageThread(tx *gorm.DB, userID entities.UserID, threadID uuid.UUID) (*entities.MessageThread, error)
func insertUnreadItem(tx *gorm.DB, item entities.MessageThreadUnreadItem) (bool, error)
func deleteUnreadItem(tx *gorm.DB, messageID uuid.UUID, threadID uuid.UUID) (bool, error)
```

`insertUnreadItem` uses `clause.OnConflict{DoNothing: true}` and returns
`RowsAffected == 1`.

- [ ] **Step 5: Implement atomic activity counting**

Inside `UpdateActivity`:

```go
return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	thread, err := lockMessageThread(tx, params.UserID, params.MessageThreadID)
	if err != nil { return err }

	if err := tx.Model(thread).Updates(messageThreadActivityUpdates(params)).Error; err != nil {
		return err
	}
	if !params.CountAsUnread || !params.EventTimestamp.After(thread.LastReadAt) {
		return nil
	}
	inserted, err := insertUnreadItem(tx, entities.MessageThreadUnreadItem{
		MessageID: params.MessageID, MessageThreadID: params.MessageThreadID,
	})
	if err != nil || !inserted { return err }
	return tx.Model(thread).UpdateColumn(
		"unread_count", gorm.Expr("unread_count + ?", 1),
	).Error
})
```

Wrap all returned errors with the existing tracer/stacktrace pattern and map a
missing locked thread to `ErrCodeNotFound`.

- [ ] **Step 6: Implement reset and deletion transactions**

`UpdateStatus` locks the thread. When `UnreadCount != nil`, update count and
watermark, delete all ledger rows for the thread, and return the updated entity.
The validator guarantees the value is zero, but the repository must reject or
return an error for a nonzero value rather than silently writing it.

`UpdateAfterDeletedMessage` locks the thread, deletes the matching ledger row,
conditionally decrements, and applies last-message fields only when
`UpdateLastMessage` is true.

- [ ] **Step 7: Update new-thread storage**

Change `Store` to accept an optional unread item ID:

```go
Store(ctx context.Context, thread *entities.MessageThread, unreadMessageID *uuid.UUID) error
```

Create the thread and initial ledger row in one transaction. Outbound threads
pass nil. Inbound threads pass their message ID and store `UnreadCount=1`.

- [ ] **Step 8: Run and commit**

Run:

```bash
cd api
gofumpt -w pkg/repositories/message_thread_repository.go pkg/repositories/gorm_message_thread_repository.go pkg/repositories/gorm_message_thread_repository_test.go
go test ./pkg/repositories -count=1
```

Expected: PASS.

```bash
git add api/pkg/repositories
git commit -m "feat(api): count unread thread items"
```

---

### Task 3: Update service and listener flows

**Files:**
- Modify: `api/pkg/services/message_thread_service.go`
- Modify: `api/pkg/services/message_thread_service_test.go`
- Modify: `api/pkg/listeners/message_thread_listener.go`
- Modify: `api/pkg/listeners/message_thread_listener_test.go`
- Modify: `api/pkg/listeners/read_receipts_test_helpers_test.go`

**Interfaces:**
- Consumes: repository contracts from Task 2.
- Produces: `MessageThreadUpdateParams.CountAsUnread bool`
- Produces: `MessageThreadStatusParams.UnreadCount *uint`

- [ ] **Step 1: Write failing service/listener tests**

Cover:

```go
assert.True(t, captured.CountAsUnread) // received SMS
assert.True(t, captured.CountAsUnread) // missed call
assert.Equal(t, uint(1), stored.UnreadCount) // new inbound
assert.Equal(t, uint(0), stored.UnreadCount) // new outbound
```

Add deletion tests proving a non-last deleted message still calls
`UpdateAfterDeletedMessage` with `UpdateLastMessage=false`, while deleting the
last message sets it true.

- [ ] **Step 2: Run focused tests and verify failure**

```bash
cd api
go test ./pkg/services ./pkg/listeners -run 'Test(MessageThread|UpdateThread|CreateThread|UpdateAfterDeleted)' -count=1
```

- [ ] **Step 3: Rename count intent and initialize new threads**

Replace `MarkAsUnread` with `CountAsUnread` throughout service/listener inputs.
For new threads:

```go
thread.UnreadCount = 0
var unreadMessageID *uuid.UUID
if params.CountAsUnread {
	thread.UnreadCount = 1
	unreadMessageID = &params.MessageID
}
err := service.repository.Store(ctx, thread, unreadMessageID)
```

- [ ] **Step 4: Make deletion cleanup unconditional**

Keep the existing whole-thread deletion when no previous message remains.
Otherwise always call the repository:

```go
updateLastMessage := thread.LastMessageID != nil && *thread.LastMessageID == payload.MessageID
err = service.repository.UpdateAfterDeletedMessage(ctx, repositories.MessageThreadDeletedUpdate{
	MessageThreadID: thread.ID,
	UserID: thread.UserID,
	DeletedMessageID: payload.MessageID,
	UpdateLastMessage: updateLastMessage,
	LastMessageID: payload.PreviousMessageID,
	LastMessageContent: payload.PreviousMessageContent,
	LastMessageStatus: *payload.PreviousMessageStatus,
})
```

- [ ] **Step 5: Run, format, and commit**

```bash
cd api
gofumpt -w pkg/services/message_thread_service.go pkg/services/message_thread_service_test.go pkg/listeners/message_thread_listener.go pkg/listeners/message_thread_listener_test.go pkg/listeners/read_receipts_test_helpers_test.go
go test ./pkg/services ./pkg/listeners -count=1
git add pkg/services pkg/listeners
git commit -m "feat(api): route unread count activity"
```

---

### Task 4: Replace the update API contract

**Files:**
- Modify: `api/pkg/requests/message_thread_update_request.go`
- Modify: `api/pkg/requests/message_thread_update_request_test.go`
- Modify: `api/pkg/validators/message_thread_handler_validator.go`
- Modify: `api/pkg/validators/message_thread_handler_validator_test.go`
- Modify: `api/pkg/handlers/message_thread_handler_test.go`

**Interfaces:**
- Produces: request field `UnreadCount *uint`
- Consumes: `services.MessageThreadStatusParams.UnreadCount *uint`

- [ ] **Step 1: Write failing request and validator tests**

Add cases for:

```go
zero := uint(0)
request := requests.MessageThreadUpdate{MessageThreadID: uuid.NewString(), UnreadCount: &zero}
assert.Empty(t, validator.ValidateUpdate(context.Background(), request))
```

```go
one := uint(1)
errors := validator.ValidateUpdate(context.Background(), requests.MessageThreadUpdate{
	MessageThreadID: uuid.NewString(), UnreadCount: &one,
})
assert.Contains(t, errors, "unread_count")
```

Also verify an `is_read`-only JSON body returns 422 because it contains no
supported update field.

- [ ] **Step 2: Run focused tests and verify failure**

```bash
cd api
go test ./pkg/requests ./pkg/validators ./pkg/handlers -run 'Test(MessageThreadUpdate|ValidateUpdate|MessageThreadHandler)' -count=1
```

- [ ] **Step 3: Replace the request and validation fields**

```go
type MessageThreadUpdate struct {
	request
	IsArchived  *bool `json:"is_archived,omitempty" example:"true"`
	UnreadCount *uint `json:"unread_count,omitempty" example:"0"`
	MessageThreadID string `json:"messageThreadID" swaggerignore:"true"`
}
```

Validation requires at least one supported pointer. If `UnreadCount != nil &&
*UnreadCount != 0`, add `"unread_count": "must be 0"`.

- [ ] **Step 4: Run, format, and commit**

```bash
cd api
gofumpt -w pkg/requests/message_thread_update_request.go pkg/requests/message_thread_update_request_test.go pkg/validators/message_thread_handler_validator.go pkg/validators/message_thread_handler_validator_test.go pkg/handlers/message_thread_handler_test.go
go test ./pkg/requests ./pkg/validators ./pkg/handlers -count=1
git add pkg/requests pkg/validators pkg/handlers
git commit -m "feat(api): expose unread count reset"
```

---

### Task 5: Regenerate Swagger and web API types

**Files:**
- Modify: `api/docs/docs.go`
- Modify: `api/docs/swagger.json`
- Modify: `api/docs/swagger.yaml`
- Modify: `web/shared/types/api.ts`

**Interfaces:**
- Produces: `EntitiesMessageThread.unread_count: number`
- Produces: `RequestsMessageThreadUpdate.unread_count?: number`
- Removes: both generated `is_read` properties.

- [ ] **Step 1: Regenerate Swagger**

```bash
cd api
swag init --requiredByDefault --parseDependency --parseInternal
```

Expected: generated docs contain `unread_count` and no message-thread
`is_read`.

- [ ] **Step 2: Regenerate web types**

```bash
cd web
pnpm api:models
```

- [ ] **Step 3: Verify generated contracts**

```bash
rg -n '"?unread_count"?|"?is_read"?' api/docs web/shared/types/api.ts
```

Expected: message-thread schemas contain `unread_count`; `is_read` has no
message-thread contract matches.

- [ ] **Step 4: Commit**

```bash
git add api/docs web/shared/types/api.ts
git commit -m "docs(api): publish unread counts"
```

---

### Task 6: Update the web store, detail page, and badge

**Files:**
- Modify: `web/app/stores/threads.ts`
- Modify: `web/app/pages/threads/[id]/index.vue`
- Modify: `web/app/components/MessageThread.vue`

**Interfaces:**
- Consumes: generated `EntitiesMessageThread.unread_count`.
- Produces: `resetThreadUnreadCount(threadId: string, force?: boolean): Promise<void>`.

- [ ] **Step 1: Replace the store action**

```ts
async function resetThreadUnreadCount(threadId: string, force = false) {
  const thread = threads.value.find((item) => item.id === threadId)
  if (!thread) throw new Error(`Cannot find thread with id ${threadId}`)
  if (!force && thread.unread_count === 0) return

  const response = await apiFetch<{ data: EntitiesMessageThread }>(
    `/v1/message-threads/${threadId}`,
    { method: 'PUT', body: { unread_count: 0 } },
  )
  replaceThread(response.data)
}
```

Preserve the existing `try/catch`, notification, reload, and `AggregateError`
behavior around the request. Export the renamed action.

- [ ] **Step 2: Rename detail-page read calls**

Rename `markCurrentThreadRead` to `resetCurrentThreadUnreadCount` and call
`threadsStore.resetThreadUnreadCount`. Preserve forced realtime resets for
received SMS and missed-call events.

- [ ] **Step 3: Render the numeric badge**

Add:

```ts
function unreadBadge(count: number): false | { color: string; content: string } {
  if (count === 0) return false
  return { color: 'primary', content: count > 99 ? '99+' : String(count) }
}
```

Use `thread.unread_count > 0` for bold classes and bind the avatar badge to
`unreadBadge(thread.unread_count)`.

- [ ] **Step 4: Run web validation**

```bash
cd web
pnpm lint
pnpm run generate
```

Expected: both commands pass.

- [ ] **Step 5: Commit**

```bash
git add web/app/stores/threads.ts web/app/pages/threads/[id]/index.vue web/app/components/MessageThread.vue
git commit -m "feat(web): show unread message counts"
```

---

### Task 7: Update integration coverage and run full validation

**Files:**
- Modify: `tests/read_receipts_test.go`
- Modify: `tests/README.md`

**Interfaces:**
- Consumes: public `unread_count` response and reset request.

- [ ] **Step 1: Convert the integration model and reset helper**

```go
type integrationMessageThread struct {
	ID                 string  `json:"id"`
	Contact            string  `json:"contact"`
	UnreadCount        uint    `json:"unread_count"`
	LastMessageContent *string `json:"last_message_content"`
}
```

Reset with:

```go
map[string]any{"unread_count": 0}
```

- [ ] **Step 2: Extend count assertions**

Exercise:

- first received SMS reaches count 1;
- second received SMS reaches count 2;
- reset returns and persists zero;
- missed call reaches count 1;
- outbound activity preserves count 1;
- deleting the unread missed-call item returns count to zero when the existing
  integration API provides the created message ID.

Do not attempt to replay internal CloudEvents through a public endpoint. Keep
duplicate-event idempotency in repository/listener tests.

- [ ] **Step 3: Update integration coverage documentation**

Change the read-receipts entry in `tests/README.md` to state that the test
covers unread SMS/missed-call counts, reset, and outbound preservation.

- [ ] **Step 4: Run API tests**

```bash
cd api
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Run web checks**

```bash
cd web
pnpm lint
pnpm run generate
```

Expected: PASS.

- [ ] **Step 6: Run targeted integration tests when the Docker stack is available**

```bash
cd tests
go test -v -timeout 120s -run TestMessageThreadReadReceipts ./...
```

Expected: PASS. If the stack is unavailable, record the connection failure
without treating it as product behavior.

- [ ] **Step 7: Scan the active contract and inspect the final diff**

```bash
rg -n 'IsRead|is_read|MarkAsUnread|markThreadRead' api/pkg web/app web/shared/types/api.ts tests
git diff --check
git status --short
```

Expected: no stale active-contract matches; historical specs/plans may still
mention `is_read`.

- [ ] **Step 8: Commit**

```bash
git add tests/read_receipts_test.go tests/README.md
git commit -m "test: cover unread message counts"
```
