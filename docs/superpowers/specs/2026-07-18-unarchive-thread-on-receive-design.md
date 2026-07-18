# Auto-unarchive Message Threads on Inbound Message (per-phone)

- Date: 2026-07-18
- Status: Approved (design)
- Scope: `api/` (Go backend) and `web/` (Nuxt frontend). Android app not touched.

## Problem

When a message thread is archived it stays archived even if the contact sends a
new inbound message. Users want an archived thread to move back to the
unarchived (inbox) state when a new message is received. This behaviour must be
configurable.

## Decisions

- **Granularity:** configurable per **phone number** (per `Phone`), consistent
  with other per-phone settings such as `MessagesPerMinute` and
  `MissedCallAutoReply`.
- **Trigger:** only **inbound** messages received from the contact
  (`entities.MessageStatusReceived`). Outbound activity (sent/sending/delivered/
  scheduled/expired) does not unarchive a thread.
- **Default:** **disabled** (opt-in). Existing and new phones default to `false`.
- **Field name:** `UnarchiveThread` (JSON `unarchive_thread`).
- **Data flow:** the flag travels on the received-message event payload
  (Option B), so `MessageThreadService` gains **no** new repository dependency.

## Design

### 1. Data model — `api/pkg/entities/phone.go`

Add a boolean to the `Phone` entity:

```go
UnarchiveThread bool `json:"unarchive_thread" gorm:"default:false" example:"false"`
```

GORM auto-migrates the column. Existing rows default to `false`.

### 2. Event payload — `api/pkg/events/message_phone_received_event.go`

Add the flag to `MessagePhoneReceivedPayload`:

```go
UnarchiveThread bool `json:"unarchive_thread"`
```

### 3. Populate the flag — `api/pkg/services/message_service.go`

In `ReceiveMessage`, before building `eventPayload`, load the owner phone using
the already-injected `phoneService` (same pattern as `RespondToMissedCall`,
which calls `service.phoneService.Load(ctx, payload.UserID, payload.Owner)`) and
copy `phone.UnarchiveThread` onto the payload.

- The owner E.164 string is `phonenumbers.Format(&params.Owner, phonenumbers.E164)`.
- If the phone lookup fails, log and default `UnarchiveThread` to `false`; do not
  fail message reception because of a missing/failed phone lookup.

### 4. Thread service trigger — `api/pkg/services/message_thread_service.go`

- Add `UnarchiveThread bool` to `MessageThreadUpdateParams`.
- In `UpdateThread`, after the thread is loaded, if the thread `IsArchived` is
  true **and** `params.Status == entities.MessageStatusReceived` **and**
  `params.UnarchiveThread` is true, set the thread's `IsArchived = false` as part
  of the same update that persists the new last message. Reuse
  `thread.UpdateArchive(false)` or set the field directly before
  `repository.Update`.
- The existing early-return guards (out-of-order timestamp, already-delivered)
  must not skip the unarchive when a genuinely new inbound message arrives; the
  unarchive is applied on the same path as the normal thread update.

### 5. Listener — `api/pkg/listeners/message_thread_listener.go`

In `OnMessagePhoneReceived`, set `UnarchiveThread: payload.UnarchiveThread` on
the `MessageThreadUpdateParams`. No other listener passes this flag (only inbound
received messages should unarchive).

### 6. Request/params — phone upsert

- `api/pkg/requests/phone_update_request.go`: add `UnarchiveThread *bool`
  (pointer, applied only when present in the JSON body — same partial-update
  pattern as `MessagesPerMinute`, using the `fields` map check on key
  `unarchive_thread`).
- `api/pkg/services/phone_service.go` `PhoneUpsertParams`: add
  `UnarchiveThread *bool` and apply it in the upsert when non-nil.

### 7. Web frontend — `web/`

- `web/app/stores/phones.ts` `updatePhone`: include
  `unarchive_thread: phone.unarchive_thread` in the PUT body.
- `web/app/pages/settings/index.vue`: add a toggle (switch/checkbox) for the
  per-phone "Unarchive thread when a new message is received" setting, alongside
  the existing per-phone settings.
- Regenerate TypeScript API models with `pnpm api:models` after the Swagger spec
  is regenerated so `EntitiesPhone` in `web/shared/types/api.ts` includes
  `unarchive_thread`.

### 8. Swagger

Run `swag init --requiredByDefault --parseDependency --parseInternal` in `api/`
after adding the annotation-affecting struct field so the generated docs and the
web model regeneration stay in sync.

## Testing

- **API unit test** (`message_thread_service` test): archived thread + inbound
  received message + `UnarchiveThread=true` -> thread becomes unarchived;
  `UnarchiveThread=false` -> stays archived; outbound status with the flag true
  -> stays archived.
- **API**: `MessageService.ReceiveMessage` populates `UnarchiveThread` from the
  loaded phone; a failed phone lookup yields `false` and does not error.
- Run `go test ./...` in `api/`.
- **Web**: `pnpm lint` and `pnpm test` in `web/`.

## Out of scope

- Android app settings UI.
- Global (user-level) or per-thread configuration.
- Auto-archiving behaviour (this design only unarchives).
