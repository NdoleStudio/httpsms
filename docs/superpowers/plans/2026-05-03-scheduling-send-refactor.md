# Scheduling Send Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow users to send SMS at an exact time (bypassing scheduling) when `SendAt` is specified, and replace the 1-second bulk hack with rate-based dispatch delays.

**Related docs:**

- [Scheduling SMS Messages](https://docs.httpsms.com/features/scheduling-sms-messages) — the existing `SendAt`/`SendTime` feature
- [Control SMS Send Rate](https://docs.httpsms.com/features/control-sms-send-rate) — the existing `MessagesPerMinute` rate-limiting feature

**Architecture:** Add a transient `ExactSendTime` flag flowing through the event system. When true, bypass [rate-limit](https://docs.httpsms.com/features/control-sms-send-rate) and schedule window logic in notification scheduling. For bulk sends without explicit time, compute dispatch delay from `MessagesPerMinute` per-phone instead of hardcoded 1s.

**Tech Stack:** Go, Fiber, GORM, CockroachDB, Google Cloud Tasks (CloudEvents)

**Spec:** `docs/superpowers/specs/2026-05-03-scheduling-send-refactor-design.md`

**Build/Test commands:**

```bash
cd api && go build ./...
cd api && go test -vet=off ./...
```

---

## Task 1: Add ExactSendTime to Event Payload

**Files:**

- Modify: `api/pkg/events/message_api_sent_event.go`

- [ ] **Step 1: Add `ExactSendTime` field to `MessageAPISentPayload`**

In `api/pkg/events/message_api_sent_event.go`, add to the struct:

```go
ExactSendTime     bool            `json:"exact_send_time"`
```

Add it after line 22 (`ScheduledSendTime *time.Time`).

- [ ] **Step 2: Build to verify no compile errors**

Run: `cd api && go build ./...`
Expected: success

- [ ] **Step 3: Commit**

```bash
cd api && git add -A && git commit -m "feat(events): add ExactSendTime field to MessageAPISentPayload"
```

---

## Task 2: Add Index and ExactSendTime to MessageSendParams + Update getSendDelay

**Files:**

- Modify: `api/pkg/services/message_service.go`

- [ ] **Step 1: Add `Index` field to `MessageSendParams`**

In `api/pkg/services/message_service.go` at line ~453, add `Index int` to the struct:

```go
type MessageSendParams struct {
	Owner             *phonenumbers.PhoneNumber
	Contact           string
	Encrypted         bool
	Content           string
	Attachments       []string
	Source            string
	SendAt            *time.Time
	RequestID         *string
	UserID            entities.UserID
	RequestReceivedAt time.Time
	Index             int
}
```

- [ ] **Step 2: Update `phoneSettings` to also return `MessagesPerMinute`**

Change the `phoneSettings` method signature and body at line ~1014:

```go
func (service *MessageService) phoneSettings(ctx context.Context, userID entities.UserID, owner string) (uint, entities.SIM, uint) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phone, err := service.phoneService.Load(ctx, userID, owner)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone for userID [%s] and owner [%s]. using default max send attempt of 2", userID, owner)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return 2, entities.SIM1, 0
	}

	return phone.MaxSendAttemptsSanitized(), phone.SIM, phone.MessagesPerMinute
}
```

- [ ] **Step 3: Update `SendMessage` to use new `phoneSettings` return value and set `ExactSendTime`**

Update `SendMessage` at line ~467. Key changes: get `messagesPerMinute` from `phoneSettings`, derive `ExactSendTime` from `SendAt != nil`, pass `messagesPerMinute` to `getSendDelay`:

```go
func (service *MessageService) SendMessage(ctx context.Context, params MessageSendParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	sendAttempts, sim, messagesPerMinute := service.phoneSettings(ctx, params.UserID, phonenumbers.Format(params.Owner, phonenumbers.E164))

	eventPayload := events.MessageAPISentPayload{
		MessageID:         uuid.New(),
		UserID:            params.UserID,
		Encrypted:         params.Encrypted,
		MaxSendAttempts:   sendAttempts,
		RequestID:         params.RequestID,
		Owner:             phonenumbers.Format(params.Owner, phonenumbers.E164),
		Contact:           params.Contact,
		RequestReceivedAt: params.RequestReceivedAt,
		Content:           params.Content,
		Attachments:       params.Attachments,
		ScheduledSendTime: params.SendAt,
		ExactSendTime:     params.SendAt != nil,
		SIM:               sim,
	}

	event, err := service.createMessageAPISentEvent(params.Source, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot create %T from payload with message id [%s]", event, eventPayload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] and message id [%s] and user [%s]", event.Type(), event.ID(), eventPayload.MessageID, eventPayload.UserID))

	message, err := service.storeSentMessage(ctx, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot store message with id [%s]", eventPayload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	timeout := service.getSendDelay(ctxLogger, eventPayload, params, messagesPerMinute)
	if _, err = service.eventDispatcher.DispatchWithTimeout(ctx, event, timeout); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s]", event.Type(), event.ID())
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("[%s] event with ID [%s] dispatched succesfully for message [%s] with user [%s] and delay [%s]", event.Type(), event.ID(), eventPayload.MessageID, eventPayload.UserID, timeout))
	return message, err
}
```

- [ ] **Step 4: Rewrite `getSendDelay` to handle rate-based delay**

Replace the existing `getSendDelay` method. New signature takes `messagesPerMinute` as a separate arg:

```go
func (service *MessageService) getSendDelay(ctxLogger telemetry.Logger, eventPayload events.MessageAPISentPayload, params MessageSendParams, messagesPerMinute uint) time.Duration {
	// Exact send time: delay until that time (clamped to 0 if in the past)
	if params.SendAt != nil {
		delay := params.SendAt.Sub(time.Now().UTC())
		if delay < 0 {
			ctxLogger.Info(fmt.Sprintf("message [%s] has send time [%s] in the past. sending immediately", eventPayload.MessageID, params.SendAt.String()))
			return time.Duration(0)
		}
		return delay
	}

	// Rate-based delay for bulk messages (Index > 0)
	if params.Index > 0 && messagesPerMinute > 0 {
		interval := time.Minute / time.Duration(messagesPerMinute)
		delay := time.Duration(params.Index) * interval
		ctxLogger.Info(fmt.Sprintf("message [%s] bulk index [%d] rate-based delay [%s]", eventPayload.MessageID, params.Index, delay))
		return delay
	}

	return time.Duration(0)
}
```

- [ ] **Step 5: Build to verify no compile errors**

Run: `cd api && go build ./...`
Expected: success

- [ ] **Step 6: Run tests**

Run: `cd api && go test -vet=off ./...`
Expected: all pass

- [ ] **Step 7: Commit**

```bash
cd api && git add -A && git commit -m "feat(services): add rate-based dispatch delay and ExactSendTime to SendMessage"
```

---

## Task 3: Add ScheduleExact to Repository Interface and Implementation

**Files:**

- Modify: `api/pkg/repositories/phone_notification_repository.go`
- Modify: `api/pkg/repositories/gorm_phone_notification_repository.go`

- [ ] **Step 1: Add `ScheduleExact` to the repository interface**

In `api/pkg/repositories/phone_notification_repository.go`:

```go
// PhoneNotificationRepository loads and persists an entities.PhoneNotification
type PhoneNotificationRepository interface {
	// Schedule a new entities.PhoneNotification
	Schedule(ctx context.Context, messagesPerMinute uint, schedule *entities.MessageSendSchedule, notification *entities.PhoneNotification) error

	// ScheduleExact stores a phone notification with a fixed ScheduledAt time,
	// bypassing rate-limit and schedule window logic.
	ScheduleExact(ctx context.Context, notification *entities.PhoneNotification) error

	// UpdateStatus of a notification
	UpdateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) error

	// DeleteAllForUser deletes all entities.PhoneNotification for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
```

- [ ] **Step 2: Implement `ScheduleExact` on `gormPhoneNotificationRepository`**

In `api/pkg/repositories/gorm_phone_notification_repository.go`, add after the `Schedule` method:

```go
// ScheduleExact stores a phone notification with an exact ScheduledAt time.
// It performs a dedupe check — if a pending notification for the same message already exists, it's a no-op.
func (repository *gormPhoneNotificationRepository) ScheduleExact(
	ctx context.Context,
	notification *entities.PhoneNotification,
) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	// Dedupe: check if a pending notification for this message already exists
	var count int64
	if err := repository.db.WithContext(ctx).
		Model(&entities.PhoneNotification{}).
		Where("message_id = ? AND status = ?", notification.MessageID, entities.PhoneNotificationStatusPending).
		Count(&count).Error; err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(err, "cannot check for existing notification for message [%s]", notification.MessageID),
		)
	}

	if count > 0 {
		return nil
	}

	if err := repository.db.WithContext(ctx).Create(notification).Error; err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(err, "cannot create exact-time notification with id [%s]", notification.ID),
		)
	}

	return nil
}
```

- [ ] **Step 3: Build to verify no compile errors**

Run: `cd api && go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
cd api && git add -A && git commit -m "feat(repositories): add ScheduleExact method for exact-time notifications"
```

---

## Task 4: Update PhoneNotificationService to Support ExactSendTime

**Files:**

- Modify: `api/pkg/services/phone_notification_service.go`

- [ ] **Step 1: Add fields to `PhoneNotificationScheduleParams`**

Update the struct at line ~162:

```go
// PhoneNotificationScheduleParams are parameters for sending a notification
type PhoneNotificationScheduleParams struct {
	UserID            entities.UserID
	Owner             string
	Source            string
	Encrypted         bool
	Contact           string
	Content           string
	SIM               entities.SIM
	MessageID         uuid.UUID
	ExactSendTime     bool
	ScheduledSendTime *time.Time
}
```

- [ ] **Step 2: Add bypass logic at the start of `Schedule` method**

Update `Schedule` method at line ~175. Add the bypass path after loading the phone:

```go
// Schedule a notification to be sent to a phone
func (service *PhoneNotificationService) Schedule(ctx context.Context, params *PhoneNotificationScheduleParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phone, err := service.phoneRepository.Load(ctx, params.UserID, params.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone with userID [%s] and phone [%s]", params.UserID, params.Owner)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	notification := &entities.PhoneNotification{
		ID:          uuid.New(),
		MessageID:   params.MessageID,
		UserID:      params.UserID,
		PhoneID:     phone.ID,
		Status:      entities.PhoneNotificationStatusPending,
		ScheduledAt: time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Bypass rate-limit and schedule window logic for exact send time
	if params.ExactSendTime && params.ScheduledSendTime != nil {
		scheduledAt := *params.ScheduledSendTime
		// Clamp past times to now (send immediately)
		if scheduledAt.Before(time.Now().UTC()) {
			scheduledAt = time.Now().UTC()
		}
		notification.ScheduledAt = scheduledAt
		if err = service.phoneNotificationRepository.ScheduleExact(ctx, notification); err != nil {
			msg := fmt.Sprintf("cannot schedule exact notification for message [%s] to phone [%s]", params.MessageID, phone.ID)
			return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}

		if err = service.dispatchMessageNotificationScheduled(ctx, params, notification); err != nil {
			ctxLogger.Error(err)
		}

		if err = service.dispatchMessageNotificationSend(ctx, params.Source, notification); err != nil {
			return service.tracer.WrapErrorSpan(span, err)
		}

		ctxLogger.Info(fmt.Sprintf(
			"message with id [%s] exact notification scheduled for [%s] with id [%s]",
			params.MessageID,
			notification.ScheduledAt,
			notification.ID,
		))
		return nil
	}

	// Standard path: apply rate-limit + schedule window logic
	var schedule *entities.MessageSendSchedule
	if phone.ScheduleID != nil {
		schedule, err = service.sendScheduleRepository.Load(ctx, params.UserID, *phone.ScheduleID)
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			schedule = nil
			err = nil
		}
		if err != nil {
			msg := fmt.Sprintf("cannot load send schedule [%s] for phone [%s]", *phone.ScheduleID, phone.ID)
			return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}
	}

	if err = service.phoneNotificationRepository.Schedule(ctx, phone.MessagesPerMinute, schedule, notification); err != nil {
		msg := fmt.Sprintf("cannot schedule notification for message [%s] to phone [%s]", params.MessageID, phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatchMessageNotificationScheduled(ctx, params, notification); err != nil {
		ctxLogger.Error(err)
	}

	if err = service.dispatchMessageNotificationSend(ctx, params.Source, notification); err != nil {
		return service.tracer.WrapErrorSpan(span, err)
	}

	ctxLogger.Info(fmt.Sprintf(
		"message with id [%s] notification scheduled for [%s] with id [%s]",
		params.MessageID,
		notification.ScheduledAt,
		notification.ID,
	))
	return nil
}
```

- [ ] **Step 3: Build to verify no compile errors**

Run: `cd api && go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
cd api && git add -A && git commit -m "feat(services): add ExactSendTime bypass in PhoneNotificationService.Schedule"
```

---

## Task 5: Update Phone Notification Listener to Pass ExactSendTime

**Files:**

- Modify: `api/pkg/listeners/phone_notification_listener.go`

- [ ] **Step 1: Pass ExactSendTime and ScheduledSendTime from event payload to service params**

Update the `onMessageAPISent` method at line ~44:

```go
func (listener *PhoneNotificationListener) onMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageAPISentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	sendParams := &services.PhoneNotificationScheduleParams{
		UserID:            payload.UserID,
		Owner:             payload.Owner,
		Contact:           payload.Contact,
		Content:           payload.Content,
		SIM:               payload.SIM,
		Encrypted:         payload.Encrypted,
		Source:            event.Source(),
		MessageID:         payload.MessageID,
		ExactSendTime:     payload.ExactSendTime,
		ScheduledSendTime: payload.ScheduledSendTime,
	}

	if err := listener.service.Schedule(ctx, sendParams); err != nil {
		msg := fmt.Sprintf("cannot send notification with params [%s] for event with ID [%s]", spew.Sdump(sendParams), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
```

- [ ] **Step 2: Build to verify no compile errors**

Run: `cd api && go build ./...`
Expected: success

- [ ] **Step 3: Commit**

```bash
cd api && git add -A && git commit -m "feat(listeners): pass ExactSendTime to PhoneNotificationService from event"
```

---

## Task 6: Update Bulk Send Request + Handler

**Files:**

- Modify: `api/pkg/requests/message_bulk_send_request.go`
- Modify: `api/pkg/handlers/message_handler.go`

- [ ] **Step 1: Remove per-index SendAt from `MessageBulkSend.ToMessageSendParams()`**

In `api/pkg/requests/message_bulk_send_request.go`, update `ToMessageSendParams`:

```go
// ToMessageSendParams converts MessageSend to services.MessageSendParams
func (input *MessageBulkSend) ToMessageSendParams(userID entities.UserID, source string) []services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.From, phonenumbers.UNKNOWN_REGION)

	var result []services.MessageSendParams
	for index, to := range input.To {
		result = append(result, services.MessageSendParams{
			Source:            source,
			Owner:             from,
			Encrypted:         input.Encrypted,
			RequestID:         input.sanitizeStringPointer(input.RequestID),
			UserID:            userID,
			RequestReceivedAt: time.Now().UTC(),
			Contact:           to,
			Content:           input.Content,
			Attachments:       input.Attachments,
			Index:             index,
		})
	}

	return result
}
```

Key changes: removed `SendAt` assignment and added `Index: index`.

- [ ] **Step 2: Remove the `index * 1s` hack from `BulkSend` handler**

In `api/pkg/handlers/message_handler.go`, update the `BulkSend` handler goroutine (around line 160-175). Remove the `if message.SendAt == nil` block:

Replace:

```go
for index, message := range params {
    wg.Add(1)
    go func(message services.MessageSendParams, index int) {
        count.Add(1)
        if message.SendAt == nil {
            sentAt := time.Now().UTC().Add(time.Duration(index) * time.Second)
            message.SendAt = &sentAt
        }

        response, err := h.service.SendMessage(ctx, message)
```

With:

```go
for index, message := range params {
    wg.Add(1)
    go func(message services.MessageSendParams, index int) {
        count.Add(1)
        response, err := h.service.SendMessage(ctx, message)
```

- [ ] **Step 3: Remove unused `time` import if needed**

Check if `time` is still used in `message_handler.go`. It likely is (used elsewhere), so skip this step if so.

- [ ] **Step 4: Build to verify no compile errors**

Run: `cd api && go build ./...`
Expected: success

- [ ] **Step 5: Commit**

```bash
cd api && git add -A && git commit -m "feat(handlers): replace 1s hack with rate-based delay for bulk send"
```

---

## Task 7: Update CSV Bulk Message Request + Handler

**Files:**

- Modify: `api/pkg/requests/bulk_message_request.go`
- Modify: `api/pkg/handlers/bulk_message_handler.go`

- [ ] **Step 1: Add `Index` parameter to `BulkMessage.ToMessageSendParams()`**

In `api/pkg/requests/bulk_message_request.go`, change the method signature to accept index:

```go
// ToMessageSendParams converts BulkMessage to services.MessageSendParams
func (input *BulkMessage) ToMessageSendParams(userID entities.UserID, requestID uuid.UUID, source string, index int) services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.FromPhoneNumber, phonenumbers.UNKNOWN_REGION)

	return services.MessageSendParams{
		Source:            source,
		Owner:             from,
		RequestID:         input.sanitizeStringPointer(fmt.Sprintf("bulk-%s", requestID.String())),
		UserID:            userID,
		SendAt:            input.SendTime,
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.ToPhoneNumber),
		Content:           input.Content,
		Attachments:       input.removeEmptyStrings(strings.Split(input.AttachmentURLs, ",")),
		Index:             index,
	}
}
```

- [ ] **Step 2: Update `BulkMessageHandler.Store()` to compute per-phone index**

In `api/pkg/handlers/bulk_message_handler.go`, update the Store method to compute per-phone indices:

```go
func (h *BulkMessageHandler) Store(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	file, err := c.FormFile("document")
	if err != nil {
		msg := fmt.Sprintf("cannot fetch file with name [%s] from request", "document")
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	messages, validationErrors := h.validator.ValidateStore(ctx, h.userIDFomContext(c), file)
	if len(validationErrors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while sending bulk sms from CSV file [%s] for [%s]", spew.Sdump(validationErrors), file.Filename, h.userIDFomContext(c))
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, validationErrors, "validation errors while sending bulk SMS")
	}

	if msg := h.billingService.IsEntitledWithCount(ctx, h.userIDFomContext(c), uint(len(messages))); msg != nil {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] is not entitled to send [%d] messages", h.userIDFomContext(c), len(messages))))
		return h.responsePaymentRequired(c, *msg)
	}

	requestID := uuid.New()
	wg := sync.WaitGroup{}
	count := atomic.Int64{}

	// Compute per-phone index for rate-based dispatch delay
	phoneIndexMap := make(map[string]int)
	for _, message := range messages {
		if message.SendTime != nil {
			continue // Exact-time messages don't need indexing
		}
		phone := message.FromPhoneNumber
		phoneIndexMap[phone]++ // Pre-count not needed, we'll compute inline
	}

	// Reset for actual iteration
	phoneIndexCounter := make(map[string]int)

	for _, message := range messages {
		wg.Add(1)
		var perPhoneIndex int
		if message.SendTime == nil {
			perPhoneIndex = phoneIndexCounter[message.FromPhoneNumber]
			phoneIndexCounter[message.FromPhoneNumber]++
		}

		go func(message *requests.BulkMessage, index int) {
			count.Add(1)
			_, err = h.messageService.SendMessage(
				ctx,
				message.ToMessageSendParams(h.userIDFomContext(c), requestID, c.OriginalURL(), index),
			)
			if err != nil {
				count.Add(-1)
				msg := fmt.Sprintf("cannot send message with paylod [%s] at index [%d]", spew.Sdump(message), index)
				ctxLogger.Error(stacktrace.Propagate(err, msg))
			}
			wg.Done()
		}(message, perPhoneIndex)
	}

	wg.Wait()
	return h.responseAccepted(c, fmt.Sprintf("Added %d out of %d messages to the queue", count.Load(), len(messages)))
}
```

- [ ] **Step 3: Clean up unused `phoneIndexMap` variable**

The `phoneIndexMap` is computed but unused. Remove it — we only need `phoneIndexCounter`:

```go
// Compute per-phone index for rate-based dispatch delay
phoneIndexCounter := make(map[string]int)

for _, message := range messages {
    wg.Add(1)
    var perPhoneIndex int
    if message.SendTime == nil {
        perPhoneIndex = phoneIndexCounter[message.FromPhoneNumber]
        phoneIndexCounter[message.FromPhoneNumber]++
    }

    go func(message *requests.BulkMessage, index int) {
        // ... same as above
    }(message, perPhoneIndex)
}
```

- [ ] **Step 4: Build to verify no compile errors**

Run: `cd api && go build ./...`
Expected: success

- [ ] **Step 5: Run tests**

Run: `cd api && go test -vet=off ./...`
Expected: all pass

- [ ] **Step 6: Commit**

```bash
cd api && git add -A && git commit -m "feat(handlers): add per-phone index for CSV bulk messages"
```

---

## Task 8: Add Unit Tests for getSendDelay

**Files:**

- Create: `api/pkg/services/message_service_test.go`

- [ ] **Step 1: Write tests for the new `getSendDelay` logic**

Create `api/pkg/services/message_service_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestGetSendDelay_WithSendAt_ReturnsTimeUntil(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	sendAt := time.Now().UTC().Add(5 * time.Minute)
	params := MessageSendParams{SendAt: &sendAt}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 10)

	// Should be approximately 5 minutes (within 2 seconds tolerance)
	assert.InDelta(t, float64(5*time.Minute), float64(delay), float64(2*time.Second))
}

func TestGetSendDelay_WithSendAtInPast_ReturnsZero(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	sendAt := time.Now().UTC().Add(-5 * time.Minute)
	params := MessageSendParams{SendAt: &sendAt}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 10)

	assert.Equal(t, time.Duration(0), delay)
}

func TestGetSendDelay_BulkIndex_RateBasedDelay(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	params := MessageSendParams{Index: 3}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	// 10 messages per minute = 6 seconds interval
	delay := service.getSendDelay(logger, payload, params, 10)

	expected := time.Duration(3) * (time.Minute / time.Duration(10))
	assert.Equal(t, expected, delay)
}

func TestGetSendDelay_BulkIndex_ZeroRate_ReturnsZero(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	params := MessageSendParams{Index: 5}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 0)

	assert.Equal(t, time.Duration(0), delay)
}

func TestGetSendDelay_IndexZero_ReturnsZero(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	params := MessageSendParams{Index: 0}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 10)

	assert.Equal(t, time.Duration(0), delay)
}

func TestGetSendDelay_NoSendAtNoIndex_ReturnsZero(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	params := MessageSendParams{}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 10)

	assert.Equal(t, time.Duration(0), delay)
}

// noopLogger implements telemetry.Logger for testing
type noopLogger struct{}

var _ telemetry.Logger = (*noopLogger)(nil)

func (l *noopLogger) Error(_ error)                      {}
func (l *noopLogger) WithService(_ string) telemetry.Logger { return l }
func (l *noopLogger) WithString(_, _ string) telemetry.Logger { return l }
func (l *noopLogger) WithSpan(_ trace.SpanContext) telemetry.Logger { return l }
func (l *noopLogger) Trace(_ string)                     {}
func (l *noopLogger) Info(_ string)                      {}
func (l *noopLogger) Warn(_ error)                       {}
func (l *noopLogger) Debug(_ string)                     {}
func (l *noopLogger) Fatal(_ error)                      {}
func (l *noopLogger) Printf(_ string, _ ...interface{})  {}
```

- [ ] **Step 2: Run the tests**

Run: `cd api && go test -vet=off ./pkg/services/ -run TestGetSendDelay -v`
Expected: all pass

- [ ] **Step 3: Commit**

```bash
cd api && git add -A && git commit -m "test(services): add unit tests for getSendDelay rate-based logic"
```

---

## Task 9: Add Unit Test for ResolveScheduledAt (Existing, Verify No Regression)

**Files:**

- Create: `api/pkg/entities/send_schedule_test.go`

- [ ] **Step 1: Write tests to lock existing ResolveScheduledAt behavior**

Create `api/pkg/entities/send_schedule_test.go`:

```go
package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveScheduledAt_NilSchedule_ReturnsCurrentUTC(t *testing.T) {
	now := time.Now()
	var schedule *MessageSendSchedule
	result := schedule.ResolveScheduledAt(now)
	assert.Equal(t, now.UTC(), result)
}

func TestResolveScheduledAt_InactiveSchedule_ReturnsCurrentUTC(t *testing.T) {
	now := time.Now()
	schedule := &MessageSendSchedule{IsActive: false}
	result := schedule.ResolveScheduledAt(now)
	assert.Equal(t, now.UTC(), result)
}

func TestResolveScheduledAt_NoWindows_ReturnsCurrentUTC(t *testing.T) {
	now := time.Now()
	schedule := &MessageSendSchedule{
		IsActive: true,
		Timezone: "UTC",
		Windows:  []MessageSendScheduleWindow{},
	}
	result := schedule.ResolveScheduledAt(now)
	assert.Equal(t, now.UTC(), result)
}

func TestResolveScheduledAt_WithinWindow_ReturnsCurrentUTC(t *testing.T) {
	// Wednesday at 10:00 UTC, window is Wed 9:00-17:00 (540-1020 minutes)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC) // Wednesday
	schedule := &MessageSendSchedule{
		IsActive: true,
		Timezone: "UTC",
		Windows: []MessageSendScheduleWindow{
			{DayOfWeek: int(now.Weekday()), StartMinute: 540, EndMinute: 1020},
		},
	}
	result := schedule.ResolveScheduledAt(now)
	assert.Equal(t, now.UTC(), result)
}

func TestResolveScheduledAt_BeforeWindow_ReturnsWindowStart(t *testing.T) {
	// Wednesday at 7:00 UTC, window is Wed 9:00-17:00
	now := time.Date(2025, 1, 1, 7, 0, 0, 0, time.UTC) // Wednesday
	schedule := &MessageSendSchedule{
		IsActive: true,
		Timezone: "UTC",
		Windows: []MessageSendScheduleWindow{
			{DayOfWeek: int(now.Weekday()), StartMinute: 540, EndMinute: 1020},
		},
	}
	result := schedule.ResolveScheduledAt(now)
	expected := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}
```

- [ ] **Step 2: Run the tests**

Run: `cd api && go test -vet=off ./pkg/entities/ -run TestResolveScheduledAt -v`
Expected: all pass

- [ ] **Step 3: Commit**

```bash
cd api && git add -A && git commit -m "test(entities): add regression tests for ResolveScheduledAt"
```

---

## Task 10: Final Build + Integration Verification

**Files:** None (verification only)

- [ ] **Step 1: Full build**

Run: `cd api && go build ./...`
Expected: success

- [ ] **Step 2: Full test suite**

Run: `cd api && go test -vet=off ./...`
Expected: all pass

- [ ] **Step 3: Generate Swagger docs (if API annotations changed)**

The API request structs' annotations haven't changed for swagger (no new endpoints, `SendAt` already documented). Skip swagger regen unless compile errors appear.

- [ ] **Step 4: Verify git status is clean**

Run: `cd api && git status`
Expected: clean working tree

---

## Notes

- The `noopLogger` in tests implements the full `telemetry.Logger` interface (Error, WithService, WithString, WithSpan, Trace, Info, Warn, Debug, Fatal, Printf).
- The `ExactSendTime` field is transient — no database migrations needed.
- **Dedupe strategy**: `ScheduleExact` uses a `SELECT COUNT` check before insert. This is not fully race-proof but acceptable given: (a) Cloud Tasks at-least-once duplicates are rare, and (b) the existing `Schedule` path also has this same theoretical gap. Adding a DB unique constraint on `(message_id, status='pending')` would require a partial index migration — this is deferred as a future improvement if duplicates become a problem in practice.
- The existing `Schedule` method already handles concurrency via CockroachDB's serializable transactions (`crdbgorm.ExecuteTx`), which retries automatically on conflicts. No additional dedupe is added there.
- All existing behavior for single messages without `SendAt` is preserved (delay = 0, standard scheduling path).
- Past `SendAt` times are handled at both layers: `getSendDelay` returns 0 (immediate dispatch), and `Schedule` clamps `ScheduledAt` to `now` (no past timestamps persisted).
