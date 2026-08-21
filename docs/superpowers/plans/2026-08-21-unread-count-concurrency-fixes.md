# Unread Count Concurrency Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make unread-count persistence idempotent under deletion replay, read/reset races, Cockroach transaction retries, and concurrent first-message creation.

**Architecture:** Keep `message_threads.unread_count` as the cached public count and retain unread ledger rows as tombstones after individual deletion. All unread mutations lock the thread first and run through `crdbgorm.ExecuteTx`; startup migration adds a non-destructive unique conversation index only after proving no duplicate identities exist.

**Tech Stack:** Go 1.25.8, GORM 1.31.2, CockroachDB `crdbgorm` 2.4.3, PostgreSQL test fakes, Testify, Swag 1.16.6, Nuxt 2/pnpm.

## Global Constraints

- Work only in `C:\Users\achoa\Work\NdoleStudio\httpsms-unread-message-count`.
- Follow strict RED-GREEN-REFACTOR and record exact RED output.
- Do not destructively deduplicate existing message threads.
- Use GORM query builders with context propagation and wrap repository/service errors with stacktrace.
- Commit once with subject `fix(api): harden unread count concurrency` and the required Copilot trailers.

---

### Task 1: Retain deleted-item tombstones

**Files:**
- Modify: `api/pkg/entities/message_thread_unread_item.go`
- Modify: `api/pkg/entities/message_thread_test.go`
- Modify: `api/pkg/repositories/gorm_message_thread_repository.go`
- Test: `api/pkg/repositories/gorm_message_thread_repository_test.go`

**Interfaces:**
- Consumes: `MessageThreadDeletedUpdate`, `MessageThreadActivityUpdate`
- Produces: `MessageThreadUnreadItem.Counted bool`

- [ ] **Step 1: Write the failing tests**

Add an entity-tag test for `Counted` and a repository behavior test that calls deletion followed by replay. Assert deletion emits a conditional `counted=true -> false` update, decrements once, replay uses conflict-ignore, and replay does not increment.

- [ ] **Step 2: Verify RED**

Run: `cd api && go test ./pkg/entities ./pkg/repositories -run 'TestMessageThreadUnreadItemCountedState|TestMessageThreadDeletedItemReplayDoesNotIncrement'`

Expected: FAIL because `Counted` and tombstone transition do not exist.

- [ ] **Step 3: Implement minimal tombstone state**

```go
type MessageThreadUnreadItem struct {
	MessageID       uuid.UUID     `gorm:"primaryKey;type:uuid"`
	MessageThreadID uuid.UUID     `gorm:"not null;type:uuid;index"`
	Counted         bool          `gorm:"not null;default:true"`
	MessageThread   MessageThread `gorm:"constraint:OnDelete:CASCADE;"`
}
```

Replace ledger deletion with a conditional `Update("counted", false)` and decrement only when that update affects one row. Keep reset deleting all rows and insert using `ON CONFLICT DO NOTHING`.

- [ ] **Step 4: Verify GREEN**

Run the Task 1 command and all repository tests.

### Task 2: Own reset watermarks after locking

**Files:**
- Modify: `api/pkg/repositories/message_thread_repository.go`
- Modify: `api/pkg/repositories/gorm_message_thread_repository.go`
- Modify: `api/pkg/services/message_thread_service.go`
- Test: `api/pkg/repositories/gorm_message_thread_repository_test.go`
- Test: `api/pkg/services/message_thread_service_test.go`

**Interfaces:**
- Produces: `MessageThreadStatusUpdate{IsArchived *bool, UnreadCount *uint}`
- Produces: repository-private `now func() time.Time`

- [ ] **Step 1: Write the failing tests**

Add a repository test whose injected clock asserts the `FOR UPDATE` statement has already executed and returns a non-UTC fixed-zone time. Assert the persisted and returned watermark is the UTC conversion. Update the service test to require no public `ReadAt` value.

- [ ] **Step 2: Verify RED**

Run: `cd api && go test ./pkg/repositories ./pkg/services -run 'TestMessageThreadStatusResetCreatesUTCWatermarkAfterLock|TestUpdateStatusForwardsOnlyPublicState'`

Expected: FAIL because the service currently creates `ReadAt`.

- [ ] **Step 3: Implement minimal ownership change**

Remove `ReadAt` from `MessageThreadStatusUpdate`, inject `time.Now` in `gormMessageThreadRepository`, and call `repository.now().UTC()` only after `lockMessageThread` succeeds. Use the same value in SQL updates and the returned entity.

- [ ] **Step 4: Verify GREEN**

Run the Task 2 command and all repository/service tests.

### Task 3: Retry all unread transactions

**Files:**
- Modify: `api/pkg/repositories/gorm_message_thread_repository.go`
- Test: `api/pkg/repositories/gorm_message_thread_repository_test.go`

**Interfaces:**
- Consumes: `crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error)`

- [ ] **Step 1: Write/adjust behavior tests**

Teach the repository fake to accept Cockroach savepoint statements. Keep assertions on lock/mutation ordering and committed outcomes, not on implementation function names.

- [ ] **Step 2: Verify RED**

Run focused repository tests after replacing one transaction at a time; an unadapted fake must expose any retry/savepoint incompatibility.

- [ ] **Step 3: Implement retry-safe closures**

Replace the four `db.Transaction` calls in `Store`, `UpdateActivity`, `UpdateStatus`, and `UpdateAfterDeletedMessage`. Reinitialize closure-local loaded/output thread state at each attempt; assign returned state only from the successful attempt.

- [ ] **Step 4: Verify GREEN**

Run: `cd api && go test ./pkg/repositories`

### Task 4: Resolve concurrent first-message creation

**Files:**
- Modify: `api/pkg/repositories/message_thread_repository.go`
- Modify: `api/pkg/repositories/gorm_message_thread_repository.go`
- Modify: `api/pkg/services/message_thread_service.go`
- Test: `api/pkg/repositories/gorm_message_thread_repository_test.go`
- Test: `api/pkg/services/message_thread_service_test.go`

**Interfaces:**
- Produces: `MessageThreadStoreParams{Thread *entities.MessageThread, CountAsUnread bool, EventTimestamp time.Time}`

- [ ] **Step 1: Write the failing tests**

Add a Store conflict test where the conversation insert affects zero rows. Assert the winning `(user_id, owner, contact)` row is locked, losing activity is applied, and its ledger/count is applied idempotently. Add a service test asserting `EventTimestamp` and `CountAsUnread` reach Store.

- [ ] **Step 2: Verify RED**

Run: `cd api && go test ./pkg/repositories ./pkg/services -run 'TestMessageThreadStoreConflictAppliesLosingActivity|TestCreateThreadForwardsStoreUnreadIntent'`

Expected: FAIL because Store currently returns success without applying the losing event.

- [ ] **Step 3: Implement minimal conflict fallback**

Store the thread with `ON CONFLICT DO NOTHING`. On conflict, lock the winner by conversation identity, apply the losing activity, compare the event watermark, insert the ledger row with conflict-ignore, and increment only on insertion. Initialize brand-new threads with a stable pre-event watermark so concurrent first events are countable while a post-create read reset still wins by lock order.

- [ ] **Step 4: Verify GREEN**

Run the Task 4 command and all repository/service tests.

### Task 5: Propagate final-message delete failures

**Files:**
- Modify: `api/pkg/services/message_thread_service.go`
- Test: `api/pkg/services/message_thread_service_test.go`

- [ ] **Step 1: Write the failing test**

Make repository `Delete` return a sentinel error when `PreviousMessageID == nil`; assert `UpdateAfterDeletedMessage` returns an error containing both the sentinel and thread context.

- [ ] **Step 2: Verify RED**

Run: `cd api && go test ./pkg/services -run TestUpdateAfterDeletedMessagePropagatesFinalThreadDeleteError`

Expected: FAIL because the service logs the error and returns nil.

- [ ] **Step 3: Implement minimal propagation**

Return the wrapped delete error instead of logging a success-shaped result.

- [ ] **Step 4: Verify GREEN**

Run all message-thread service tests.

### Task 6: Make schema migration non-destructive and idempotent

**Files:**
- Modify: `api/pkg/migrations/message_thread_unread_count.go`
- Test: `api/pkg/migrations/message_thread_unread_count_test.go`

**Interfaces:**
- Produces: unique index `idx_message_threads_conversation` over `(user_id, owner, contact)`

- [ ] **Step 1: Write the failing migration tests**

Extend the fake schema state to cover legacy `is_read`, existing indexes, and duplicate identities. Assert backfill precedes drop, a second run skips both, counted-column migration occurs, duplicate identities return a precise error before index creation, and a clean schema creates the unique index.

- [ ] **Step 2: Verify RED**

Run: `cd api && go test ./pkg/migrations`

Expected: FAIL because current coverage cannot model state transitions and no unique migration exists.

- [ ] **Step 3: Implement safest migration**

Auto-migrate the table/ledger columns, backfill then drop `is_read`, preflight duplicate conversation identities using a GORM grouped query, and create the composite unique index through `Migrator.CreateIndex` only when absent and safe. Never delete or merge duplicates.

- [ ] **Step 4: Verify GREEN**

Run all migration tests. Record that no real database migration was run.

### Task 7: Constrain and regenerate the API contract

**Files:**
- Modify: `api/pkg/requests/message_thread_update_request.go`
- Regenerate: `api/docs/docs.go`
- Regenerate: `api/docs/swagger.json`
- Regenerate: `api/docs/swagger.yaml`
- Regenerate: `web/shared/types/api.ts`

- [ ] **Step 1: Write the contract assertion**

Add or update a request reflection/generated-contract test requiring exactly-zero Swagger metadata.

- [ ] **Step 2: Verify RED**

Run the focused request test and confirm the generated Swagger lacks the constraint.

- [ ] **Step 3: Implement and regenerate**

Use `minimum:"0" maximum:"0"` on `UnreadCount`. Run pinned `go run github.com/swaggo/swag/cmd/swag@v1.16.6 init --requiredByDefault --parseDependency --parseInternal`, then `pnpm api:models`.

- [ ] **Step 4: Verify GREEN**

Assert generated Swagger has minimum and maximum zero and the web type remains `unread_count?: number`.

### Task 8: Validate, review, commit, and report

**Files:**
- Create: `C:\Users\achoa\Work\NdoleStudio\httpsms\.git\sdd\unread-concurrency-fix-report.md`

- [ ] **Step 1: Format and run required validation**

Run gofumpt on changed Go files, `cd api && go test ./...`, `cd web && pnpm lint && pnpm run generate`, and `cd tests && go test -run '^$' ./...`.

- [ ] **Step 2: Review invariants**

Review retry closure state, lock order, tombstone lifecycle, conflict fallback, migration safety, generated contracts, and unrelated diffs.

- [ ] **Step 3: Commit**

Commit all intended changes with the exact requested subject and trailers.

- [ ] **Step 4: Write report**

Write exact RED/GREEN output, changed files, validations, migration limitations, concerns, and commit hash to the required report path.
