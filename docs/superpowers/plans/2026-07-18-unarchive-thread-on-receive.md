# Unarchive Thread on Receive Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an archived message thread automatically move back to the inbox when a new inbound message is received, configurable per phone number.

**Architecture:** Add a per-phone boolean `UnarchiveThread` (default false). It is copied onto the `message.phone.received` event payload when a message is received, carried through the thread listener into `MessageThreadUpdateParams`, and applied by `MessageThreadService.UpdateThread`, which clears `IsArchived` only for inbound (received) messages when the flag is true. A per-phone toggle is exposed in the web settings page.

**Tech Stack:** Go 1.x (Fiber v3, GORM, stacktrace, testify), Nuxt 4 / Vue 3 (Vuetify 4, Pinia), swaggo/swag, swagger-typescript-api.

## Global Constraints

- Field name is exactly `UnarchiveThread` (Go) / `unarchive_thread` (JSON) everywhere.
- Default value is `false` (opt-in). GORM tag: `gorm:"default:false"`.
- Only inbound messages (`entities.MessageStatusReceived`) may unarchive a thread.
- Wrap all Go errors with `github.com/palantir/stacktrace` (never return bare errors).
- Go formatting via `go-fumpt` (pre-commit). Run `go test ./...` from `api/`.
- After changing Swagger annotations run, in `api/`: `swag init --requiredByDefault --parseDependency --parseInternal`.
- After Swagger regen, regenerate web models in `web/`: `pnpm api:models`.
- Web: no semicolons, single quotes, 2-space indent (Prettier/ESLint). Lint via `pnpm lint`.
- The `docs/` directory is gitignored in this repo; this plan and its spec were committed with `git add -f`. Do NOT `-f` add source files — only the plan/spec docs need force-add.

## File Structure

- `api/pkg/entities/phone.go` — add `UnarchiveThread` field to `Phone`.
- `api/pkg/events/message_phone_received_event.go` — add `UnarchiveThread` to `MessagePhoneReceivedPayload`.
- `api/pkg/services/message_service.go` — populate `UnarchiveThread` on the received-event payload from the loaded phone.
- `api/pkg/services/message_thread_service.go` — add `UnarchiveThread` to `MessageThreadUpdateParams`; add pure helper `shouldUnarchive`; apply it in `UpdateThread`.
- `api/pkg/services/message_thread_service_test.go` — NEW unit tests for `shouldUnarchive`.
- `api/pkg/listeners/message_thread_listener.go` — pass `payload.UnarchiveThread` into params in `OnMessagePhoneReceived`.
- `api/pkg/requests/phone_update_request.go` — add `UnarchiveThread *bool` request field + partial-update mapping.
- `api/pkg/services/phone_service.go` — add `UnarchiveThread *bool` to `PhoneUpsertParams`; apply in `update`.
- `web/app/stores/phones.ts` — send `unarchive_thread` in `updatePhone` PUT body.
- `web/app/pages/settings/index.vue` — add a per-phone toggle bound to `activePhone.unarchive_thread`.
- `web/shared/types/api.ts` — regenerated (not hand-edited).
- `tests/unarchive_thread_integration_test.go` — NEW end-to-end integration test (package `tests`, runs against the live Docker stack).

---

### Task 1: Add `UnarchiveThread` field to the Phone entity

**Files:**
- Modify: `api/pkg/entities/phone.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `entities.Phone.UnarchiveThread bool` (JSON `unarchive_thread`).

- [ ] **Step 1: Add the field**

In `api/pkg/entities/phone.go`, inside the `Phone` struct, add the field just below `MissedCallAutoReply` (keep the existing blank-line grouping style):

```go
	MissedCallAutoReply *string `json:"missed_call_auto_reply" example:"This phone cannot receive calls. Please send an SMS instead." validate:"optional"`

	// UnarchiveThread moves an archived message thread back to the inbox when a new message is received on this phone.
	UnarchiveThread bool `json:"unarchive_thread" gorm:"default:false" example:"false"`
```

- [ ] **Step 2: Verify it compiles**

Run (from `api/`): `go build ./...`
Expected: no output, exit code 0.

- [ ] **Step 3: Commit**

```bash
git add api/pkg/entities/phone.go
git commit -m "feat(api): add UnarchiveThread setting to Phone entity"
```

---

### Task 2: Add `UnarchiveThread` to the received-message event payload

**Files:**
- Modify: `api/pkg/events/message_phone_received_event.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `events.MessagePhoneReceivedPayload.UnarchiveThread bool` (JSON `unarchive_thread`).

- [ ] **Step 1: Add the field**

In `api/pkg/events/message_phone_received_event.go`, add the field to the `MessagePhoneReceivedPayload` struct (after `Attachments`):

```go
type MessagePhoneReceivedPayload struct {
	MessageID       uuid.UUID       `json:"message_id"`
	UserID          entities.UserID `json:"user_id"`
	Owner           string          `json:"owner"`
	Encrypted       bool            `json:"encrypted"`
	Contact         string          `json:"contact"`
	Timestamp       time.Time       `json:"timestamp"`
	Content         string          `json:"content"`
	SIM             entities.SIM    `json:"sim"`
	Attachments     []string        `json:"attachments"`
	UnarchiveThread bool            `json:"unarchive_thread"`
}
```

- [ ] **Step 2: Verify it compiles**

Run (from `api/`): `go build ./...`
Expected: exit code 0.

- [ ] **Step 3: Commit**

```bash
git add api/pkg/events/message_phone_received_event.go
git commit -m "feat(api): add UnarchiveThread to MessagePhoneReceivedPayload"
```

---

### Task 3: Populate `UnarchiveThread` on the payload in ReceiveMessage

**Files:**
- Modify: `api/pkg/services/message_service.go` (function `ReceiveMessage`, around lines 333-362)

**Interfaces:**
- Consumes: `entities.Phone.UnarchiveThread` (Task 1); `events.MessagePhoneReceivedPayload.UnarchiveThread` (Task 2); existing `service.phoneService.Load(ctx, userID, ownerE164 string) (*entities.Phone, error)`.
- Produces: received events now carry the phone's `UnarchiveThread` value.

- [ ] **Step 1: Load the phone and set the flag**

In `ReceiveMessage`, the owner E.164 string is `phonenumbers.Format(&params.Owner, phonenumbers.E164)`. Replace the `eventPayload := events.MessagePhoneReceivedPayload{...}` block so the owner string is computed once and the phone setting is looked up. Insert BEFORE the `eventPayload :=` assignment:

```go
	owner := phonenumbers.Format(&params.Owner, phonenumbers.E164)

	unarchiveThread := false
	phone, err := service.phoneService.Load(ctx, params.UserID, owner)
	if err != nil {
		ctxLogger.Warn(stacktrace.Propagate(err, fmt.Sprintf("cannot load phone [%s] for user [%s] to resolve UnarchiveThread; defaulting to false", owner, params.UserID)))
	} else {
		unarchiveThread = phone.UnarchiveThread
	}
```

Then update the payload literal to reuse `owner` and set the flag:

```go
	eventPayload := events.MessagePhoneReceivedPayload{
		MessageID:       messageID,
		UserID:          params.UserID,
		Encrypted:       params.Encrypted,
		Owner:           owner,
		Contact:         params.Contact,
		Timestamp:       params.Timestamp,
		Content:         params.Content,
		SIM:             params.SIM,
		Attachments:     attachmentURLs,
		UnarchiveThread: unarchiveThread,
	}
```

Note: `err` is already declared earlier in `ReceiveMessage` (from the attachments upload), so use `=` not `:=` for the phone load, as shown. Confirm `stacktrace` is already imported in this file (it is).

- [ ] **Step 2: Verify it compiles**

Run (from `api/`): `go build ./...`
Expected: exit code 0. If you get "err redeclared" or "err not used", ensure the phone load uses `phone, err = ...` on its own line (not `:=`) and that `phone` is newly declared with `:=` — since `phone` is new and `err` exists, `phone, err := ...` is correct Go (at least one new var on the left). Prefer `phone, err := service.phoneService.Load(...)`.

Correction to Step 1: use `phone, err := service.phoneService.Load(ctx, params.UserID, owner)` (mixed assignment is valid because `phone` is new). Keep the rest as written.

- [ ] **Step 3: Run existing message service tests**

Run (from `api/`): `go test ./pkg/services/ -run TestMessageService -v`
Expected: PASS (these are pure helper tests unaffected by this change). Also run `go build ./...` again to be safe.

- [ ] **Step 4: Commit**

```bash
git add api/pkg/services/message_service.go
git commit -m "feat(api): populate UnarchiveThread from phone on received event"
```

---

### Task 4: Thread unarchive decision helper + wiring (TDD)

**Files:**
- Modify: `api/pkg/services/message_thread_service.go`
- Create: `api/pkg/services/message_thread_service_test.go`

**Interfaces:**
- Consumes: `entities.MessageThread.IsArchived`; `entities.MessageStatusReceived`; `MessageThreadUpdateParams`.
- Produces:
  - `MessageThreadUpdateParams.UnarchiveThread bool`
  - `func (service *MessageThreadService) shouldUnarchive(thread *entities.MessageThread, params MessageThreadUpdateParams) bool`

- [ ] **Step 1: Write the failing test**

Create `api/pkg/services/message_thread_service_test.go`:

```go
package services

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func TestShouldUnarchive(t *testing.T) {
	service := &MessageThreadService{}

	archived := &entities.MessageThread{IsArchived: true}
	notArchived := &entities.MessageThread{IsArchived: false}

	received := MessageThreadUpdateParams{Status: entities.MessageStatusReceived, UnarchiveThread: true}
	receivedFlagOff := MessageThreadUpdateParams{Status: entities.MessageStatusReceived, UnarchiveThread: false}
	sentFlagOn := MessageThreadUpdateParams{Status: entities.MessageStatusSent, UnarchiveThread: true}

	assert.True(t, service.shouldUnarchive(archived, received), "archived + inbound + flag on -> unarchive")
	assert.False(t, service.shouldUnarchive(archived, receivedFlagOff), "flag off -> no unarchive")
	assert.False(t, service.shouldUnarchive(archived, sentFlagOn), "outbound status -> no unarchive")
	assert.False(t, service.shouldUnarchive(notArchived, received), "already unarchived -> no change")
}
```

Note: confirm `entities.MessageStatusSent` exists (it does — used in listeners). If not, substitute any non-received status constant such as `entities.MessageStatusPending`.

- [ ] **Step 2: Run the test to verify it fails**

Run (from `api/`): `go test ./pkg/services/ -run TestShouldUnarchive -v`
Expected: FAIL — compile error `service.shouldUnarchive undefined` and `params.UnarchiveThread` unknown field.

- [ ] **Step 3: Add the param field and the helper**

In `api/pkg/services/message_thread_service.go`, add the field to `MessageThreadUpdateParams`:

```go
type MessageThreadUpdateParams struct {
	Owner           string
	Status          entities.MessageStatus
	Contact         string
	Content         string
	UserID          entities.UserID
	MessageID       uuid.UUID
	Timestamp       time.Time
	UnarchiveThread bool
}
```

Then add the pure helper (place it directly above `UpdateThread`):

```go
// shouldUnarchive reports whether an archived thread should be moved back to
// the inbox because a new inbound message was received and the phone has the
// UnarchiveThread setting enabled.
func (service *MessageThreadService) shouldUnarchive(thread *entities.MessageThread, params MessageThreadUpdateParams) bool {
	return thread.IsArchived && params.UnarchiveThread && params.Status == entities.MessageStatusReceived
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run (from `api/`): `go test ./pkg/services/ -run TestShouldUnarchive -v`
Expected: PASS.

- [ ] **Step 5: Apply the decision inside UpdateThread**

In `UpdateThread`, the thread is updated via `thread.Update(params.Timestamp, params.MessageID, params.Content, params.Status)`. Immediately BEFORE the `if err = service.repository.Update(ctx, thread.Update(...))` call, add:

```go
	if service.shouldUnarchive(thread, params) {
		thread.UpdateArchive(false)
		ctxLogger.Info(fmt.Sprintf("unarchiving thread [%s] after inbound message [%s]", thread.ID, params.MessageID))
	}
```

`thread.UpdateArchive(false)` mutates the same `thread` pointer that `thread.Update(...)` mutates and persists, so the single `repository.Update` call saves both the new last message and the cleared archive flag. Do not add a second Update call.

- [ ] **Step 6: Run the full services test package**

Run (from `api/`): `go test ./pkg/services/ -v`
Expected: PASS (including `TestShouldUnarchive` and existing `TestMessageService*`).

- [ ] **Step 7: Commit**

```bash
git add api/pkg/services/message_thread_service.go api/pkg/services/message_thread_service_test.go
git commit -m "feat(api): unarchive thread on inbound message when enabled"
```

---

### Task 5: Pass the flag through the received-message listener

**Files:**
- Modify: `api/pkg/listeners/message_thread_listener.go` (function `OnMessagePhoneReceived`)

**Interfaces:**
- Consumes: `events.MessagePhoneReceivedPayload.UnarchiveThread` (Task 2); `MessageThreadUpdateParams.UnarchiveThread` (Task 4).
- Produces: inbound received events now request unarchiving.

- [ ] **Step 1: Set the field on the params**

In `OnMessagePhoneReceived`, update the `updateParams := services.MessageThreadUpdateParams{...}` literal to include the flag:

```go
	updateParams := services.MessageThreadUpdateParams{
		Owner:           payload.Owner,
		Contact:         payload.Contact,
		Timestamp:       payload.Timestamp,
		UserID:          payload.UserID,
		Status:          entities.MessageStatusReceived,
		Content:         payload.Content,
		MessageID:       payload.MessageID,
		UnarchiveThread: payload.UnarchiveThread,
	}
```

Do NOT set `UnarchiveThread` on any other handler in this file (only inbound received messages unarchive).

- [ ] **Step 2: Verify it compiles and tests pass**

Run (from `api/`): `go build ./...` then `go test ./pkg/services/... ./pkg/listeners/...`
Expected: exit code 0 / PASS.

- [ ] **Step 3: Commit**

```bash
git add api/pkg/listeners/message_thread_listener.go
git commit -m "feat(api): forward UnarchiveThread flag from received event to thread update"
```

---

### Task 6: Accept `unarchive_thread` in the phone upsert request/service

**Files:**
- Modify: `api/pkg/requests/phone_update_request.go`
- Modify: `api/pkg/services/phone_service.go` (`PhoneUpsertParams` struct and `update` method)

**Interfaces:**
- Consumes: `entities.Phone.UnarchiveThread` (Task 1).
- Produces: `PhoneUpsertParams.UnarchiveThread *bool`; PUT `/v1/phones` persists the setting.

- [ ] **Step 1: Add the request field**

In `api/pkg/requests/phone_update_request.go`, add to the `PhoneUpsert` struct (after `MissedCallAutoReply`):

```go
	MissedCallAutoReply *string `json:"missed_call_auto_reply" example:"e.g. This phone cannot receive calls. Please send an SMS instead."`

	// UnarchiveThread moves an archived thread back to the inbox when a new message is received on this phone.
	UnarchiveThread bool `json:"unarchive_thread" example:"false"`
```

- [ ] **Step 2: Map it as a partial-update field in ToUpsertParams**

Still in `phone_update_request.go`, inside `ToUpsertParams`, after the `maxSendAttempts` block, add the presence-detected mapping (mirrors the existing pattern):

```go
	var unarchiveThread *bool
	if _, exists := fields["unarchive_thread"]; exists {
		unarchiveThread = &input.UnarchiveThread
	}
```

Then add `UnarchiveThread: unarchiveThread,` to the returned `&services.PhoneUpsertParams{...}` literal.

- [ ] **Step 3: Add the field to PhoneUpsertParams and apply it**

In `api/pkg/services/phone_service.go`, add to `PhoneUpsertParams`:

```go
	UnarchiveThread *bool
```

In the `update` method, before `phone.SIM = params.SIM`, add:

```go
	if params.UnarchiveThread != nil {
		phone.UnarchiveThread = *params.UnarchiveThread
	}
```

- [ ] **Step 4: Verify build and tests**

Run (from `api/`): `go build ./...` then `go test ./pkg/...`
Expected: exit code 0 / PASS.

- [ ] **Step 5: Regenerate Swagger docs**

Run (from `api/`): `swag init --requiredByDefault --parseDependency --parseInternal`
Expected: regenerates `api/docs/*` including the new `unarchive_thread` fields. If `swag` is not installed: `go install github.com/swaggo/swag/cmd/swag@latest` then re-run.

- [ ] **Step 6: Commit**

```bash
git add api/pkg/requests/phone_update_request.go api/pkg/services/phone_service.go api/docs
git commit -m "feat(api): accept unarchive_thread in phone upsert request"
```

---

### Task 7: Regenerate web API models

**Files:**
- Modify (generated): `web/shared/types/api.ts`

**Interfaces:**
- Consumes: regenerated Swagger from Task 6.
- Produces: `EntitiesPhone.unarchive_thread?: boolean` available to the web app.

- [ ] **Step 1: Install web deps (if not already installed)**

Run (from `web/`): `pnpm install`
Expected: completes without `ERR_PNPM_IGNORED_BUILDS`.

- [ ] **Step 2: Regenerate models**

Run (from `web/`): `pnpm api:models`
Expected: `web/shared/types/api.ts` now includes `unarchive_thread` on the phone type (search the file for `unarchive_thread`).

- [ ] **Step 3: Verify the field exists**

Run (from `web/`): `Select-String -Path shared/types/api.ts -Pattern unarchive_thread`
Expected: at least one match.

- [ ] **Step 4: Commit**

```bash
git add web/shared/types/api.ts
git commit -m "chore(web): regenerate api models with unarchive_thread"
```

---

### Task 8: Send `unarchive_thread` in the phones store update

**Files:**
- Modify: `web/app/stores/phones.ts` (function `updatePhone`, around lines 48-62)

**Interfaces:**
- Consumes: `EntitiesPhone.unarchive_thread` (Task 7).
- Produces: PUT `/v1/phones` body includes `unarchive_thread`.

- [ ] **Step 1: Add the field to the PUT body**

In `updatePhone`, inside the `body` object passed to `apiFetch('/v1/phones', { method: 'PUT', ... })`, add after `message_send_schedule_id`:

```ts
          message_send_schedule_id: phone.message_send_schedule_id ?? null,
          unarchive_thread: phone.unarchive_thread ?? false,
```

- [ ] **Step 2: Lint**

Run (from `web/`): `pnpm lint`
Expected: no new errors for `web/app/stores/phones.ts`.

- [ ] **Step 3: Commit**

```bash
git add web/app/stores/phones.ts
git commit -m "feat(web): send unarchive_thread in updatePhone"
```

---

### Task 9: Add the per-phone toggle to the settings UI

**Files:**
- Modify: `web/app/pages/settings/index.vue` (phone settings card, near the `missed_call_auto_reply` VTextarea around lines 1680-1690)

**Interfaces:**
- Consumes: `activePhone.unarchive_thread` (Task 7); `updatePhone` store action (Task 8).
- Produces: user-visible toggle that persists via the existing "Update Phone" button.

- [ ] **Step 1: Add a VSwitch bound to activePhone.unarchive_thread**

In `web/app/pages/settings/index.vue`, immediately AFTER the closing `/>` of the `missed_call_auto_reply` `<VTextarea>` (line ~1690) and before the `</VCol>`, add:

```html
                <VSwitch
                  v-model="activePhone.unarchive_thread"
                  class="mt-4"
                  color="primary"
                  density="compact"
                  hide-details
                  label="Unarchive thread on new message"
                  hint="When a new message is received, move an archived conversation back to the inbox"
                  persistent-hint
                />
```

Note: match the surrounding indentation exactly (this block sits inside the same `<VCol>` as the textarea). If `activePhone` is typed and `unarchive_thread` is optional, `v-model` still works; the store update coerces with `?? false`.

- [ ] **Step 2: Lint**

Run (from `web/`): `pnpm lint`
Expected: no new errors for `settings/index.vue`. Auto-fix formatting if needed with `pnpm lintfix`.

- [ ] **Step 3: Build the site to confirm the template compiles**

Run (from `web/`): `pnpm run generate`
Expected: build completes without template/compile errors. (If `generate` is heavy, `pnpm dev` briefly and confirm the settings page renders is an acceptable alternative.)

- [ ] **Step 4: Commit**

```bash
git add web/app/pages/settings/index.vue
git commit -m "feat(web): add per-phone unarchive-thread toggle to settings"
```

---

### Task 10: Full validation

**Files:** none (verification only).

- [ ] **Step 1: API full test + build**

Run (from `api/`): `go test ./...` then `go build ./...`
Expected: all PASS, exit code 0.

- [ ] **Step 2: Web lint + test**

Run (from `web/`): `pnpm lint` then `pnpm test`
Expected: PASS.

- [ ] **Step 3: Manual smoke (optional, if a local stack is available)**

Start the stack (`docker compose up --build`), open the web settings for a phone, enable "Unarchive thread on new message", save. Archive a thread, then simulate/receive an inbound message for that owner and confirm the thread returns to the inbox. Toggle off and confirm an inbound message leaves the thread archived.

- [ ] **Step 4: Final commit (only if any fixups were needed)**

```bash
git add -A
git commit -m "chore: finalize unarchive-thread-on-receive feature"
```

---

### Task 11: End-to-end integration test

**Files:**
- Create: `tests/unarchive_thread_integration_test.go`

**Context:** The `tests/` module is a black-box integration suite (package `tests`)
that runs against the full Docker stack (API on `localhost:8000`, CockroachDB,
Redis, phone emulator). It uses the external `github.com/NdoleStudio/httpsms-go`
client for convenience, but that client does NOT know about the new
`unarchive_thread` field — so this test sets the phone flag with a raw HTTP PUT
to `/v1/phones` and reads threads with raw HTTP GETs. It reuses existing helpers
from `helpers_test.go`: `apiBaseURL`, `userAPIKey`, `newAPIClient`, `setupPhone`,
`randomPhoneNumber`, `randomEncryptionKey`, `pollMessageStatus`.

**Interfaces:**
- Consumes: the running API with all prior tasks deployed; endpoints
  `PUT /v1/phones`, `POST /v1/messages/receive`, `GET /v1/message-threads`,
  `PUT /v1/message-threads/{id}`.
- Produces: `TestUnarchiveThreadOnReceive_Enabled` and
  `TestUnarchiveThreadOnReceive_Disabled`.

- [ ] **Step 1: Write the integration test file**

Create `tests/unarchive_thread_integration_test.go`:

```go
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	httpsms "github.com/NdoleStudio/httpsms-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type integrationThread struct {
	ID                 string  `json:"id"`
	Contact            string  `json:"contact"`
	Owner              string  `json:"owner"`
	IsArchived         bool    `json:"is_archived"`
	LastMessageContent *string `json:"last_message_content"`
}

// setUnarchiveThread flips the per-phone unarchive_thread flag via a raw PUT
// (the httpsms-go client has no field for it).
func setUnarchiveThread(ctx context.Context, t *testing.T, phoneNumber string, enabled bool) {
	t.Helper()

	payload := map[string]interface{}{
		"phone_number":     phoneNumber,
		"sim":              "SIM1",
		"unarchive_thread": enabled,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiBaseURL+"/v1/phones", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "set unarchive_thread failed: %s", string(respBody))
}

// receiveInbound submits an inbound message as the phone and returns the message ID.
func receiveInbound(ctx context.Context, t *testing.T, phoneAPIKey, from, to, content string, ts time.Time) string {
	t.Helper()

	payload := map[string]interface{}{
		"from":      from,
		"to":        to,
		"content":   content,
		"sim":       "SIM1",
		"timestamp": ts.UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"/v1/messages/receive", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", phoneAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "receive failed: %s", string(respBody))

	var result httpsms.MessageResponse
	require.NoError(t, json.Unmarshal(respBody, &result))
	id := result.Data.ID.String()
	require.NotEmpty(t, id)
	return id
}

// fetchThreads returns threads for an owner filtered by archived state.
func fetchThreads(ctx context.Context, t *testing.T, owner string, archived bool) []integrationThread {
	t.Helper()

	url := fmt.Sprintf("%s/v1/message-threads?owner=%s&is_archived=%t&limit=20", apiBaseURL, owner, archived)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "fetch threads failed: %s", string(respBody))

	var result struct {
		Data []integrationThread `json:"data"`
	}
	require.NoError(t, json.Unmarshal(respBody, &result))
	return result.Data
}

func findThreadByContact(threads []integrationThread, contact string) *integrationThread {
	for i := range threads {
		if threads[i].Contact == contact {
			return &threads[i]
		}
	}
	return nil
}

// waitForThread polls the archived/unarchived thread list until a thread for the
// contact appears (optionally matching a last-message content), then returns it.
func waitForThread(ctx context.Context, t *testing.T, owner, contact string, archived bool, wantContent string, timeout time.Duration) *integrationThread {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		thread := findThreadByContact(fetchThreads(ctx, t, owner, archived), contact)
		if thread != nil && (wantContent == "" || (thread.LastMessageContent != nil && *thread.LastMessageContent == wantContent)) {
			return thread
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// archiveThread archives a thread by ID.
func archiveThread(ctx context.Context, t *testing.T, threadID string) {
	t.Helper()

	body, err := json.Marshal(map[string]interface{}{"is_archived": true})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiBaseURL+"/v1/message-threads/"+threadID, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "archive thread failed: %s", string(respBody))
}

func TestUnarchiveThreadOnReceive_Enabled(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)
	setUnarchiveThread(ctx, t, phone.PhoneNumber, true)

	contact := randomPhoneNumber()
	content1 := "first inbound " + randomEncryptionKey()
	content2 := "second inbound " + randomEncryptionKey()

	// First inbound message creates the thread.
	msgID1 := receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, content1, time.Now().Add(-1*time.Minute))
	pollMessageStatus(ctx, t, msgID1, "received", 15*time.Second)

	thread := waitForThread(ctx, t, phone.PhoneNumber, contact, false, "", 15*time.Second)
	require.NotNil(t, thread, "thread not created for contact %s", contact)

	// Archive it.
	archiveThread(ctx, t, thread.ID)
	archived := waitForThread(ctx, t, phone.PhoneNumber, contact, true, "", 10*time.Second)
	require.NotNil(t, archived, "thread was not archived")

	// Second inbound message should unarchive it.
	msgID2 := receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, content2, time.Now())
	pollMessageStatus(ctx, t, msgID2, "received", 15*time.Second)

	unarchived := waitForThread(ctx, t, phone.PhoneNumber, contact, false, content2, 20*time.Second)
	require.NotNil(t, unarchived, "thread was not unarchived after inbound message")
	assert.False(t, unarchived.IsArchived)
	require.NotNil(t, unarchived.LastMessageContent)
	assert.Equal(t, content2, *unarchived.LastMessageContent)
}

func TestUnarchiveThreadOnReceive_Disabled(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60) // unarchive_thread defaults to false; do not enable it

	contact := randomPhoneNumber()
	content1 := "first inbound " + randomEncryptionKey()
	content2 := "second inbound " + randomEncryptionKey()

	msgID1 := receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, content1, time.Now().Add(-1*time.Minute))
	pollMessageStatus(ctx, t, msgID1, "received", 15*time.Second)

	thread := waitForThread(ctx, t, phone.PhoneNumber, contact, false, "", 15*time.Second)
	require.NotNil(t, thread, "thread not created for contact %s", contact)

	archiveThread(ctx, t, thread.ID)
	archived := waitForThread(ctx, t, phone.PhoneNumber, contact, true, "", 10*time.Second)
	require.NotNil(t, archived, "thread was not archived")

	// Second inbound message must NOT unarchive it. Sync on the archived thread's
	// last_message_content updating to content2 (proves the listener processed it),
	// then assert it is still archived.
	msgID2 := receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, content2, time.Now())
	pollMessageStatus(ctx, t, msgID2, "received", 15*time.Second)

	stillArchived := waitForThread(ctx, t, phone.PhoneNumber, contact, true, content2, 20*time.Second)
	require.NotNil(t, stillArchived, "archived thread did not reflect the second inbound message")
	assert.True(t, stillArchived.IsArchived, "thread should remain archived when unarchive_thread is disabled")

	// And it must not have leaked into the unarchived list.
	assert.Nil(t, findThreadByContact(fetchThreads(ctx, t, phone.PhoneNumber, false), contact),
		"thread should not appear in the unarchived list")
}
```

- [ ] **Step 2: Build the test module (compile check without a running stack)**

Run (from `tests/`): `go vet ./...`
Expected: exit code 0 (compiles). If `httpsms.MessageResponse` field access differs, adjust to match the version pinned in `tests/go.mod` (the existing `integration_test.go` already uses `httpsms.MessageResponse` and `result.Data.ID`).

- [ ] **Step 3: Run the full integration suite against the Docker stack**

Run (from `tests/`), following the README one-liner:

```bash
bash generate-firebase-credentials.sh
export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json)
docker compose up -d --build --wait
docker compose wait seed
sleep 2
go test -v -timeout 180s -run TestUnarchiveThreadOnReceive ./...
docker compose down -v
```

On Windows without bash/jq available, run the integration suite via the CI
workflow (`.github/workflows/integration-test.yml`) or a Linux/WSL shell; the
Step 2 `go vet` compile check is the local gate.

Expected: both `TestUnarchiveThreadOnReceive_Enabled` and
`TestUnarchiveThreadOnReceive_Disabled` PASS.

- [ ] **Step 4: Update the test coverage checklist in the README**

In `tests/README.md`, under "## Test Coverage", add:

```markdown
- [x] **Unarchive Thread on Receive E2E** — Archived thread returns to the inbox on inbound message when the phone's `unarchive_thread` setting is enabled, and stays archived when disabled
```

- [ ] **Step 5: Commit**

```bash
git add tests/unarchive_thread_integration_test.go tests/README.md
git commit -m "test(integration): verify unarchive thread on receive"
```

---

## Self-Review

**Spec coverage:**
- Data model field -> Task 1. ✓
- Event payload field -> Task 2. ✓
- Populate flag in ReceiveMessage via phoneService.Load -> Task 3 (with fallback to false on load error). ✓
- Thread trigger (inbound-only, archived-only, same Update call) -> Task 4. ✓
- Listener wiring, inbound-only -> Task 5. ✓
- Phone upsert request + params partial update -> Task 6. ✓
- Swagger regen -> Task 6 Step 5. ✓
- Web: store PUT body -> Task 8; settings toggle -> Task 9; model regen -> Task 7. ✓
- Testing (pure helper unit test, go test, web lint/test) -> Tasks 4 & 10. ✓
- Integration test (live-stack E2E, enabled + disabled) -> Task 11. ✓
- Out of scope (Android, global/per-thread config, auto-archive) -> not implemented. ✓

**Placeholder scan:** No TBD/TODO; all code steps include concrete code and exact commands.

**Type consistency:** `UnarchiveThread` (Go) / `unarchive_thread` (JSON) used identically across entity, event, params, request, store, and template. Helper `shouldUnarchive(thread, params)` signature matches its test and its call site. `phoneService.Load(ctx, userID, ownerE164)` matches existing usage in `RespondToMissedCall`.
