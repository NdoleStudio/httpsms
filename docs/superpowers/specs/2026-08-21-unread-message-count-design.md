# Message Thread Unread Count

- Date: 2026-08-21
- Status: Approved (design)
- Scope: `api/` Go backend and `web/` Nuxt frontend. Android is unchanged.
- Branch: `feat/unread-message-count`, based on `origin/main`

## Problem

Message threads currently expose only a binary `is_read` state. Users can tell
that a thread contains unread activity, but not how many inbound items they have
not opened.

Replace the binary state with an exact, server-owned unread count. Received SMS
messages and missed calls each contribute one unread item. Opening a thread
resets its count to zero.

## Decisions

- `unread_count` is the sole public unread-state field.
- Remove `is_read` from the Go entity, API requests, API responses, generated
  web types, and UI logic.
- A received SMS increments the count once.
- A missed call increments the count once.
- Duplicate or retried events do not increment the count twice.
- Outbound messages and later delivery/status updates preserve the count.
- Opening a thread resets the count to zero.
- The existing thread update endpoint accepts `unread_count: 0`; clients cannot
  assign a nonzero count.
- Deleting a still-unread inbound item decrements the count without an extra
  preliminary database read.
- Existing `is_read=false` threads migrate to `unread_count=1`; exact counting
  starts for new activity after deployment.
- The thread list displays a numeric badge through `99`, then displays `99+`.
- The existing internal `last_read_at` watermark remains to resolve races.

## Architecture

Use a ledger-backed cached counter:

1. `message_threads.unread_count` is the value returned by the API and rendered
   by the web UI.
2. An internal unread-item ledger stores the message ID for every post-deploy
   inbound item that currently contributes to the count.
3. Thread updates, ledger changes, and count changes occur in one transaction.

The ledger provides idempotency. Its message ID is unique, so replaying the same
received-SMS or missed-call event cannot increment the count again. The cached
counter keeps thread-list reads as cheap as they are today and avoids correlated
message-count queries on every page load.

## Persistence

### Message thread

Replace `IsRead` with:

```go
UnreadCount uint `json:"unread_count" gorm:"not null;default:0" example:"2"`
```

Keep:

```go
LastReadAt time.Time `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`
```

`LastReadAt` is not exposed to clients. It remains the ordering watermark used
to stop a delayed inbound listener from restoring unread state after the user
has opened the thread.

### Unread-item ledger

Add an internal entity with:

- `MessageID` as its UUID primary key;
- `MessageThreadID` as an indexed UUID foreign key;
- cascading deletion when the thread is deleted.

The ledger does not need a public API. A globally unique message ID identifies
both received SMS messages and stored missed-call messages.

### Schema transition

The startup migration performs these steps before normal service traffic:

1. Add `message_threads.unread_count` with a non-null default of zero.
2. Create the unread-item ledger table and indexes.
3. Where the legacy column exists, set `unread_count=1` for rows whose
   `is_read=false`; leave previously read rows at zero.
4. Drop the legacy `is_read` column after the backfill succeeds.

The transition is idempotent: it checks schema state before each one-time step.
Migration errors remain fatal. The application must not serve a mixed contract
or silently skip a failed backfill.

Existing unread rows intentionally have no synthetic ledger record. Their
preserved count of one remains until the thread is opened. All inbound activity
processed after deployment is tracked exactly in the ledger.

## API Components

### Listener inputs

Rename the service/repository intent from `MarkAsUnread` to `CountAsUnread`.
`MessageThreadUpdateParams` continues to carry:

- the message ID;
- the activity timestamp used for thread ordering;
- the CloudEvent timestamp used as the unread watermark;
- whether this event represents a countable inbound item.

Received-SMS and missed-call listeners set `CountAsUnread=true`. Outbound,
sending, delivery, failure, scheduling, and expiry listeners leave it false.

### Service

`MessageThreadService` continues to coordinate:

- loading or creating the thread;
- last-message metadata;
- optional unarchiving for inbound activity;
- repository calls.

The service does not implement ledger or counter arithmetic. Those details stay
inside the repository transaction.

New threads start with:

- `unread_count=1` and one ledger row when created from countable inbound
  activity;
- `unread_count=0` and no ledger row for outbound activity.

### Repository

For a countable existing-thread update, the repository transaction:

1. locks the thread row;
2. updates the normal last-message activity fields;
3. compares the CloudEvent timestamp with `last_read_at`;
4. inserts the message ID into the ledger with conflict-ignore when the event
   is newer than the read watermark;
5. increments `unread_count` only when the insert affected one row.

For a non-countable event, the repository updates only the existing activity
and optional unarchive fields. It does not touch the ledger, count, or read
watermark.

For a read reset, the repository transaction:

1. locks and updates the authenticated user's thread;
2. sets `unread_count=0` and `last_read_at` to the same UTC timestamp;
3. deletes all ledger rows for the thread;
4. returns the updated thread.

For deletion of an individual message, the service must process unread-ledger
cleanup even when the deleted item was not the thread's last message. If the
deleted item was the last message, the repository also applies the existing
last-message replacement fields. In the same transaction it:

1. removes the matching ledger row;
2. decrements `unread_count` with a floor of zero only if a ledger row was
   removed;
3. updates last-message metadata only when the deleted item was the thread's
   current last message.

This deletion path needs no extra lookup and no extension to the deletion event:
the payload already includes the deleted message ID.

Deleting a thread or user cascades or explicitly deletes its ledger rows as part
of the existing deletion operation.

## Update Endpoint

Keep:

```text
PUT /v1/message-threads/{messageThreadID}
```

Replace the optional `is_read` request field with:

```go
UnreadCount *uint `json:"unread_count,omitempty" example:"0"`
```

Validation rules:

- at least one of `is_archived` or `unread_count` is present;
- if `unread_count` is present, its only valid value is zero;
- archive-only updates preserve unread count and `last_read_at`;
- unread-reset-only updates preserve archive state;
- combined archive/reset updates apply atomically;
- invalid IDs and unsupported or empty payloads return the existing bad-request
  response;
- a thread outside the authenticated user scope returns the existing not-found
  response.

API responses contain `unread_count` and no `is_read`.

## Concurrency and Idempotency

All operations that mutate unread state lock the thread first and use the same
lock order. This prevents receive, read, and delete transactions from producing
a counter/ledger mismatch.

The required race behavior is:

- duplicate inbound event: the ledger conflict prevents a second increment;
- inbound event committed before read: the later read clears its ledger row and
  resets the count;
- old inbound event processed after read: its CloudEvent timestamp is not newer
  than `last_read_at`, so it does not insert or increment;
- genuinely new inbound event after read: it inserts and increments;
- deletion after read: the cleared ledger has no matching row, so the count
  remains zero;
- repeated deletion: only the first successful ledger deletion can decrement;
- every decrement uses a floor of zero.

Transaction failures roll back activity metadata, ledger mutations, and counter
changes together.

## Web Components

### Store

Replace binary read checks with `thread.unread_count > 0`.

The current mark-read action becomes an unread-count reset:

```json
{ "unread_count": 0 }
```

The action:

- skips the request when the local count is already zero unless a realtime
  refresh explicitly forces reconciliation;
- replaces the matching local thread from the successful API response;
- does not optimistically clear the count;
- preserves the existing notification and reload behavior when the request
  fails.

Opening a thread invokes the reset as part of the existing message-loading flow.
Inbound realtime activity for the currently open thread forces the idempotent
reset so the thread does not remain unread while visible.

### Thread list

`MessageThread.vue` uses `unread_count > 0` for:

- bold contact text;
- bold message preview text;
- displaying the primary-color avatar badge.

The badge displays:

- no badge for zero;
- the exact count from 1 through 99;
- `99+` for counts above 99.

The same shared component continues to cover mobile, desktop, inbox, and
archived thread lists.

### Generated contracts

After changing API annotations:

1. Run `swag init --requiredByDefault --parseDependency --parseInternal` in
   `api/`.
2. Run `pnpm api:models` in `web/`.

Commit generated Swagger files and `web/shared/types/api.ts`.

## Error Handling

- Repository and service errors continue to use `stacktrace.Propagate`.
- Repository transactions return errors instead of falling back to
  success-shaped state.
- Missing user-owned threads preserve the repository not-found code.
- Migration and backfill failures stop startup.
- The web UI clears a badge only from a successful response or subsequent
  reload.
- Failed automatic resets remain visible through the existing notification
  path and do not block message display.

## Testing

### API unit and repository tests

Cover:

- entity schema defaults and removal of the public `is_read` field;
- migration of legacy read rows to zero and unread rows to one;
- new inbound thread initialization with count one and a ledger row;
- new outbound thread initialization with count zero;
- received-SMS increments;
- missed-call increments;
- duplicate inbound-event idempotency;
- outbound and delivery/status preservation;
- read reset clearing both counter and ledger;
- archive-only and reset-only field isolation;
- combined archive/reset atomicity;
- old delayed inbound event losing to a newer read watermark;
- genuinely new inbound event incrementing after a read;
- unread-item deletion decrementing once for both last and non-last messages;
- deletion after read and repeated deletion preserving zero;
- counter underflow protection;
- request validation accepting only `unread_count=0`;
- handler responses exposing `unread_count` and preserving not-found errors.

Run:

```bash
cd api
go test ./...
```

### Web validation

Cover the store and component behavior with existing frontend test facilities
where available, including zero/nonzero logic, exact badge values, and `99+`.
Then run:

```bash
cd web
pnpm lint
pnpm run generate
```

### Integration

Extend the read-receipts integration coverage to exercise:

1. a received SMS increments a thread to one;
2. replaying that event does not increment again;
3. another received SMS increments to two;
4. opening/resetting the thread returns the count to zero;
5. a missed call increments to one;
6. deleting that unread missed-call message returns the count to zero;
7. outbound/status activity does not change the count.

Run:

```bash
cd tests
go test -v -timeout 120s -run TestMessageThreadReadReceipts ./...
```

## Out of Scope

- Android unread-count UI.
- Per-device or per-user-within-a-shared-account read positions.
- Manual "mark unread" behavior.
- Client-assigned nonzero counts.
- Unread-count badges outside the thread list.
- Reconstructing an exact historical count for threads that were already
  unread before deployment.
