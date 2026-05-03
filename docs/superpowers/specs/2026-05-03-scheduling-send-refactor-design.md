# Scheduling Send Refactor Design

## Problem Statement

The current SMS scheduling logic has two issues:

1. **No way to send at an exact time without scheduling interference.** When a user specifies a `SendTime`/`SendAt`, the system still applies rate-limiting and schedule window logic, which may shift the actual send time.

2. **Bulk message contention.** When bulk messages (API or CSV) are sent, all events arrive at the Cloud Tasks queue near-simultaneously, causing DB serialization conflicts in `PhoneNotificationRepository.Schedule()` (which uses `SELECT ... ORDER BY scheduled_at DESC` in a transaction). The current workaround is a hardcoded 1-second spacing hack.

## Proposed Solution

### Core Principle

- **Explicit `SendTime`** = send at exactly that time, bypass all scheduling logic.
- **No `SendTime`** = apply full scheduling logic (rate-limit + schedule windows), with rate-based Cloud Task dispatch delay to prevent DB contention.

### Design

#### 1. ExactSendTime Flag (Transient — not persisted)

A boolean `ExactSendTime` flows through the event system:

```
Request → MessageSendParams → MessageAPISentPayload → PhoneNotificationScheduleParams
```

When `true`, the notification scheduling layer sets `ScheduledAt` to the exact time and skips rate-limit + window logic.

#### 2. Rate-Based Dispatch Delay

For bulk messages without an explicit `SendTime`, instead of the `index * 1s` hack, the service computes:

```go
interval := time.Minute / time.Duration(messagesPerMinute)
delay := time.Duration(index) * interval
```

Where `index` is **per-phone** (not global across the batch). This spreads Cloud Task deliveries at the phone's actual send rate, eliminating DB contention naturally. Duration math avoids integer truncation issues for rates > 60/min or non-divisors of 60.

#### 3. Per-Endpoint Behavior

| Endpoint                                | `SendAt` provided                                    | `SendAt` absent                                           |
| --------------------------------------- | ---------------------------------------------------- | --------------------------------------------------------- |
| Single SMS API (`/v1/messages/send`)    | `ExactSendTime=true`, delay = `time.Until(SendAt)`   | `ExactSendTime=false`, delay = 0                          |
| Bulk SMS API (`/v1/messages/bulk-send`) | N/A (no SendAt field)                                | `ExactSendTime=false`, delay = `perPhoneIndex * interval` |
| CSV Upload                              | `ExactSendTime=true`, delay = `time.Until(SendTime)` | `ExactSendTime=false`, delay = `perPhoneIndex * interval` |

**Index is per-phone**: In a CSV with messages to multiple phones, each phone maintains its own index counter. Messages to Phone A get indices 0, 1, 2... and messages to Phone B get separate indices 0, 1, 2... This ensures correct rate-limiting per phone without over-throttling unrelated phones.

#### 4. Notification Scheduling Bypass

In `PhoneNotificationService.Schedule()`:

```go
if params.ExactSendTime && params.ScheduledSendTime != nil {
    notification.ScheduledAt = *params.ScheduledSendTime
    // Skip rate-limit and schedule window logic
    // Insert directly
} else {
    // Existing logic: rate-limit + schedule window
}
```

### Changes by File

| File                                                     | Change                                                                                                                                                                                                                                         |
| -------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pkg/events/message_api_sent_event.go`                   | Add `ExactSendTime bool` field to `MessageAPISentPayload`                                                                                                                                                                                      |
| `pkg/services/message_service.go`                        | Add `Index int` to `MessageSendParams`; update `getSendDelay()` to compute rate-based delay when `Index > 0` and `SendAt == nil`; set `ExactSendTime` on event payload when `SendAt != nil`                                                    |
| `pkg/services/phone_notification_service.go`             | Add `ExactSendTime bool` + `ScheduledSendTime *time.Time` to `PhoneNotificationScheduleParams`; add bypass path in `Schedule()` when `ExactSendTime && ScheduledSendTime != nil` — insert notification directly without transaction/rate logic |
| `pkg/repositories/gorm_phone_notification_repository.go` | Add `ScheduleExact(ctx, notification)` method that inserts with a fixed `ScheduledAt` (no transaction, no rate query). Add unique constraint or dedupe check on `(message_id)` for pending notifications to ensure idempotency.                |
| `pkg/repositories/phone_notification_repository.go`      | Add `ScheduleExact` to the repository interface                                                                                                                                                                                                |
| `pkg/listeners/phone_notification_listener.go`           | Pass `ExactSendTime` + `ScheduledSendTime` from event payload to service params                                                                                                                                                                |
| `pkg/requests/message_bulk_send_request.go`              | Remove per-index `SendAt` computation; add `Index` to each `MessageSendParams`                                                                                                                                                                 |
| `pkg/requests/bulk_message_request.go`                   | Propagate `Index` into params for CSV rows                                                                                                                                                                                                     |
| `pkg/handlers/message_handler.go`                        | Remove `index * 1s` hack in `BulkSend` handler                                                                                                                                                                                                 |
| `pkg/handlers/bulk_message_handler.go`                   | Compute per-phone index for CSV rows; remove any concurrent scheduling; ensure `Index` is passed to `MessageSendParams`                                                                                                                        |

### Data Flow

```
User sends request
  → Handler creates MessageSendParams (with Index for bulk, ExactSendTime derived from SendAt presence)
    → MessageService.SendMessage()
      → Computes dispatch delay:
        - ExactSendTime: time.Until(SendAt)
        - Bulk without SendAt: Index * (60/MessagesPerMinute)s
        - Single without SendAt: 0
      → Sets ExactSendTime on MessageAPISentPayload
      → DispatchWithTimeout(event, delay) → Cloud Tasks
        → [delay elapses] → PhoneNotificationListener.onMessageAPISent()
          → PhoneNotificationService.Schedule(params with ExactSendTime)
            → If ExactSendTime: insert with exact ScheduledAt
            → Else: apply rate-limit + schedule window logic
```

### Edge Cases

- **SendAt in the past**: Send immediately (existing behavior preserved).
- **MessagesPerMinute = 0**: No rate limiting; bulk messages dispatch immediately (existing behavior — `Schedule()` already handles this). Rate-based delay uses 0 when rate is 0.
- **No schedule attached to phone**: Window logic returns current time unchanged (existing behavior).
- **CSV with mixed rows**: Some rows have `SendTime`, others don't. Each row is processed independently — those with `SendTime` get exact dispatch, those without get rate-based delay.
- **Cloud Task duplicate delivery**: `ScheduleExact` and `Schedule` use a dedupe check (unique active notification per `message_id`) to prevent duplicate notification creation on at-least-once delivery.
- **Retries for exact-send messages**: When an exact-send message expires and triggers a retry, the retry does NOT preserve exact-send semantics — it falls through to standard scheduling. The explicit time was a one-shot intent.

### Terminology Note

"Send at exactly that time" means the system will not apply additional rate-limit or schedule-window adjustments. It does NOT guarantee precise handset delivery timing (which depends on Cloud Tasks delivery, FCM push, and device state).

### What Does NOT Change

- The `MessageSendSchedule` entity and its `ResolveScheduledAt()` logic
- The `SendScheduleService` CRUD operations
- The phone notification entity schema (no new DB columns)
- The Android app behavior
- The web frontend (models auto-generated from Swagger)
