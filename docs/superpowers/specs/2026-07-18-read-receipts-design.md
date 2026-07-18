# Message Thread Read Receipts

- Date: 2026-07-18
- Status: Approved (design)
- Scope: `api/` Go backend and `web/` Nuxt frontend. Android is unchanged.
- Branch: `feat/read-receipts`, based on `origin/main`

## Problem

The message-thread list does not distinguish conversations with unseen inbound
activity from conversations the user has already opened. Users need unread
threads to stand out and become read automatically when opened.

The change must be backward compatible: every thread that exists when the new
schema is deployed starts as read.

## Decisions

- Unread state is stored per message thread.
- Existing threads are read after migration.
- A new incoming SMS or missed call marks its thread unread.
- Outbound messages and later delivery/status updates do not change read state.
- Opening a thread in the web UI marks it read automatically.
- A thread that receives inbound activity while already open remains read.
- There is no manual "Mark as read/unread" UI.
- Unread threads use bold contact/preview text and a subtle primary-tinted
  background.
- The existing `PUT /v1/message-threads/{messageThreadID}` endpoint handles the
  read update; no new endpoint is added.

## Design

### 1. Thread persistence

Add these fields to `api/pkg/entities/message_thread.go`:

```go
IsRead     bool      `json:"is_read" gorm:"not null;default:true" example:"true"`
LastReadAt time.Time `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`
```

`IsRead` is the public binary state used by API clients and the web UI.
`LastReadAt` is internal ordering metadata used to make concurrent event
processing deterministic.

GORM `AutoMigrate` adds both columns. The database defaults make all existing
rows read and give them a migration-time read watermark. New-thread creation
sets `IsRead` explicitly according to whether the triggering activity is
inbound.

Existing-row updates must not use the repository's current full-row `Save`
method. A stale entity loaded by one event handler could otherwise overwrite a
newer read action from the web request.

### 2. Atomic repository updates

Add field-scoped repository operations for the two update paths:

- **Message activity update:** update only `order_timestamp`,
  `last_message_id`, `last_message_content`, and `status`. For inbound activity,
  conditionally set `is_read = false` only where the stored `last_read_at` is
  older than the CloudEvent creation time.
- **User status update:** update only the supplied `is_archived` and `is_read`
  fields. Marking read writes `is_read = true` and `last_read_at = now` in the
  same database update.

The message activity update runs its metadata and conditional unread statements
inside one GORM transaction. Both operations remain scoped by `user_id` and
thread ID and use GORM's query builder with context propagation.

After a user status update, reload the thread for the API response. Do not use
`Save` in either path, because it writes unrelated columns and reintroduces lost
updates.

The stored timestamp condition prevents this race:

1. An inbound event is created.
2. The websocket listener notifies the open web page.
3. The web page marks the thread read.
4. The message-thread listener finishes later.

Because CloudEvent creation time precedes the read request, the late listener
checks the newer database watermark and does not overwrite the read action. If
the message listener commits first, the later read update wins normally.

### 3. Thread update parameters

Extend `services.MessageThreadUpdateParams` with:

```go
MarksUnread    bool
EventTimestamp time.Time
```

Inbound SMS and missed-call listeners set `MarksUnread = true` and use
`event.Time()` as `EventTimestamp`. Other message events leave `MarksUnread`
false, so sending, scheduling, delivery, failure, and expiry updates preserve
the current read state.

For an existing thread, the service delegates the field-scoped message update
and optional conditional unread transition to the repository.

For a newly created thread:

- inbound SMS or missed call: `IsRead = false`;
- outbound activity: `IsRead = true`.

### 4. Missed-call thread updates

`MessageThreadListener` currently handles received SMS events but not
`message.call.missed`. Register the missed-call event and update/create the
corresponding thread with:

- owner and contact from `MessageCallMissedPayload`;
- message ID and event timestamp;
- received status;
- a concise preview such as `Missed phone call`;
- `MarksUnread = true`.

This ensures a missed call is visible as thread activity even when no auto-reply
is configured.

### 5. Existing update endpoint

Change `requests.MessageThreadUpdate` to use optional fields:

```go
IsArchived *bool `json:"is_archived,omitempty" example:"true"`
IsRead     *bool `json:"is_read,omitempty" example:"true"`
```

The validator requires at least one supported field. Pointer fields distinguish
"not supplied" from `false`, preventing a read-only update from unarchiving a
thread and an archive-only update from changing read state.

`MessageThreadStatusParams` receives the optional fields. `UpdateStatus`:

- loads the authenticated user's thread;
- builds an update map containing only supplied fields;
- writes `last_read_at = time.Now().UTC()` alongside `is_read = true`;
- supports `IsRead = false` at the API level without adding a web control;
- persists one combined partial update and reloads the updated thread.

The handler preserves repository error codes so a missing thread returns 404.
Invalid IDs or empty/unsupported update bodies return validation errors.

### 6. Web store behavior

Regenerate the API model so `EntitiesMessageThread` includes `is_read` and
`RequestsMessageThreadUpdate` has optional `is_archived` and `is_read`.

In `web/app/stores/threads.ts`:

- keep archive updates on the existing endpoint with
  `{ is_archived: value }`;
- add `markThreadRead(threadId)` using `{ is_read: true }`;
- replace the matching thread in local state with the successful API response;
- make the operation idempotent and skip the request when the local thread is
  already read unless it is called after an inbound realtime event.

Opening a thread invokes `markThreadRead` automatically as part of the thread
loading flow. Message fetching remains independent: if the read update fails,
messages still display, the thread remains/reloads as unread, and the existing
notification system reports the failure.

### 7. Realtime behavior

The message detail page already reloads messages for incoming SMS websocket
events. That reload also marks the currently selected thread read.

Add `message.call.missed` to:

- `WebsocketListener` registrations in the API;
- the Pusher bindings on the thread detail page.

The websocket payload can remain the event ID. The selected thread is marked
read idempotently after any inbound realtime refresh; a different thread's
backend state remains unread.

### 8. Thread-list presentation

In `web/app/components/MessageThread.vue`, unread list items receive:

- bold contact title;
- bold message preview;
- a subtle background using the Vuetify primary theme color at low opacity.

Read threads retain the current appearance. Active-route styling remains
visible and takes precedence where Vuetify applies it.

The highlight is applied in the shared thread-list component, so it works in
both the mobile thread page and desktop navigation drawer, including archived
threads.

### 9. Swagger and generated types

After changing the entity and request annotations:

1. Run `swag init --requiredByDefault --parseDependency --parseInternal` in
   `api/`.
2. Run `pnpm api:models` in `web/`.

Generated Swagger files and `web/shared/types/api.ts` are committed with the
implementation.

## Error Handling

- Repository and service errors continue to use `stacktrace.Propagate`.
- The update handler returns 404 for a thread that does not belong to the
  authenticated user.
- A failed automatic read update is not swallowed and does not block message
  display.
- Local UI state changes only from a successful response or a subsequent thread
  reload; the UI does not claim a persisted read state after an API failure.
- Realtime callbacks report failures through the existing notification/error
  path rather than using empty catches.

## Testing

### API

Add targeted tests for:

- entity schema defaults and new-thread read-state initialization;
- an inbound SMS making an existing thread unread;
- a missed call creating or updating an unread thread;
- outbound and delivery/status events preserving read state;
- a read action winning when its timestamp is newer than an inbound event;
- concurrent message/read updates changing only their owned columns;
- archive-only updates preserving read state;
- read-only updates preserving archive state;
- missing threads preserving the not-found error code;
- existing-row migration defaults (`is_read = true`).

Run:

```bash
cd api
go test ./...
```

### Web

No frontend unit-test runner is configured. Validate the generated types,
Pinia/Vue changes, styles, and production output with:

```bash
cd web
pnpm lint
pnpm run generate
```

Manual acceptance checks:

1. Existing threads appear read immediately after migration.
2. A new incoming SMS highlights its closed thread.
3. A missed call highlights its closed thread.
4. Opening an unread thread removes the highlight and persists across refresh.
5. Incoming activity in the currently open thread does not leave it unread.
6. Sending or receiving delivery updates does not clear another unread state.
7. Archiving/unarchiving does not change read state.

## Out of Scope

- Per-message read receipts.
- Android read/unread UI.
- Unread counts or badges outside the thread list.
- Manual read/unread controls.
- Cross-user/group-conversation receipt tracking.
