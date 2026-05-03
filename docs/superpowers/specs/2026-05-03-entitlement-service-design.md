# Entitlement Service Design

## Problem

The [MessageSendSchedule](./2026-05-03-scheduling-send-refactor-design.md#messagesendschedule-send-windows--new-feature) feature (and future features) need usage limits based on the user's subscription plan. Free users should be limited to 1 send schedule; paid users get unlimited. The system must be:

- **Scalable**: Easy to add new entity limits without architectural changes
- **Configurable**: Disabled by default for self-hosted deployments, enabled via env var for cloud
- **Non-invasive**: Enforced at the handler layer, before business logic executes

## Approach

Create a dedicated `EntitlementService` in `pkg/services/` that:

1. Reads `ENTITLEMENT_ENABLED` from environment (defaults to `false`)
2. Defines a code-based map of entity limits per subscription plan
3. Exposes a single `Check()` method that handlers call before creating resources
4. Returns 402 Payment Required when a free user exceeds their limit

## Configuration

### Environment Variable

```env
# Set to "true" on cloud deployment; self-hosted defaults to false (no limits)
ENTITLEMENT_ENABLED=false
```

### Entity Limits (code-based)

```go
// entityLimits maps entity name → subscription plan → max count
// A limit of 0 means unlimited. If a plan is not listed, it defaults to unlimited.
var entityLimits = map[string]map[entities.SubscriptionName]int{
    "MessageSendSchedule": {
        entities.SubscriptionNameFree: 1,
    },
    // Future: add more entities here
    // "Webhook": {
    //     entities.SubscriptionNameFree: 3,
    // },
}
```

## Service Interface

```go
// EntitlementService checks whether a user can create more of a given entity.
type EntitlementService struct {
    logger         telemetry.Logger
    tracer         telemetry.Tracer
    enabled        bool
    userRepository repositories.UserRepository
}

// NewEntitlementService creates the service. `enabled` comes from ENTITLEMENT_ENABLED env var.
func NewEntitlementService(
    logger telemetry.Logger,
    tracer telemetry.Tracer,
    enabled bool,
    userRepository repositories.UserRepository,
) *EntitlementService

// CheckResult holds the outcome of an entitlement check.
type CheckResult struct {
    Allowed bool
    Message string
}

// Check verifies if the user can create another instance of the given entity.
// - If entitlements are disabled (self-hosted), always returns Allowed: true.
// - Loads the user's subscription plan.
// - Looks up the limit for the entity + plan combination.
// - Compares currentCount against the limit.
func (s *EntitlementService) Check(
    ctx context.Context,
    userID entities.UserID,
    entityName string,
    currentCount int,
) (*CheckResult, error)
```

## Handler Integration

In `SendScheduleHandler.Store()`:

```go
func (h *SendScheduleHandler) Store(c *fiber.Ctx) error {
    // 1. Validate request (existing logic)
    // 2. Get current count (efficient COUNT query)
    count, err := h.service.CountByUser(ctx, userID)
    if err != nil { ... }
    // 3. Check entitlement
    result, err := h.entitlementService.Check(ctx, userID, "MessageSendSchedule", count)
    if err != nil {
        return h.responseInternalServerError(c)
    }
    if !result.Allowed {
        return h.responsePaymentRequired(c, result.Message)
    }
    // 4. Proceed with creating schedule (existing logic)
}
```

## Repository Addition

Add to `SendScheduleRepository` interface and GORM implementation:

```go
// CountByUser returns the number of schedules owned by a user.
CountByUser(ctx context.Context, userID entities.UserID) (int, error)
```

````

## Error Response

HTTP 402 Payment Required:

```json
{
    "message": "Upgrade to a paid plan to create more than 1 send schedule. Visit https://httpsms.com/pricing for details.",
    "status": "payment_required"
}
````

## Files to Create/Modify

| Action | File                                                | Change                                                       |
| ------ | --------------------------------------------------- | ------------------------------------------------------------ |
| Create | `pkg/services/entitlement_service.go`               | New service with limits map, `Check()`, `CheckResult`        |
| Modify | `pkg/handlers/handler.go`                           | Add `responsePaymentRequired()` helper method                |
| Modify | `pkg/handlers/send_schedule_handler.go`             | Inject `EntitlementService`, add check in `Store()`          |
| Modify | `pkg/di/container.go`                               | Wire `EntitlementService`, read env var, inject into handler |
| Modify | `pkg/repositories/send_schedule_repository.go`      | Add `CountByUser()` to interface                             |
| Modify | `pkg/repositories/gorm_send_schedule_repository.go` | Implement `CountByUser()` with SQL COUNT                     |
| Modify | `pkg/services/send_schedule_service.go`             | Add `CountByUser()` pass-through method                      |
| Modify | `.env.example` or `.env`                            | Add `ENTITLEMENT_ENABLED=false`                              |

## Concurrency & Race Conditions

The handler-level check (`count → check → create`) is not atomic. Two concurrent requests could both see `count=0` and both proceed. Mitigations:

1. **Repository count method**: Use `CountByUser(ctx, userID)` instead of loading all records (efficient SQL `SELECT COUNT(*)`).
2. **Acceptable race window**: For a limit of 1, the worst case is 2 schedules created. This is acceptable because:
   - The window is extremely small (single user, same millisecond)
   - The consequence is minor (user has 2 schedules instead of 1)
   - A DB-level unique constraint is impractical here (limit is per-user count, not per-row uniqueness)
3. **Future hardening**: If stricter enforcement is needed, add an advisory lock or transaction-based count+insert.

## Counting Semantics

All schedules owned by the user count toward the limit, regardless of `is_active` status. A user must delete a schedule to free up their quota.

## Error Handling When Enabled

- **Entitlements disabled** (`ENTITLEMENT_ENABLED=false`): Always returns `Allowed: true`, zero DB calls.
- **Entitlements enabled, DB error loading user**: Return error (surfaces as 500). Do NOT fail-open — this is a monetized feature gate.
- **Entitlements enabled, entity not in limits map**: Returns `Allowed: true` (entity has no restrictions).

## Design Decisions

1. **Handler-layer enforcement**: The handler gets the count and calls `Check()`. This keeps the entitlement service free of domain-specific repository dependencies.
2. **Entity name as key**: Using the entity struct name (e.g., `"MessageSendSchedule"`) makes it self-documenting and matches the user's preference for entity-based naming.
3. **Fail-open when disabled**: Self-hosted users never hit limits. The `enabled` flag short-circuits all checks.
4. **Fail-closed on error when enabled**: If the user can't be loaded and entitlements are enabled, the request fails with 500.
5. **Separate from BillingService**: BillingService handles SMS message counting/billing. EntitlementService handles feature-level access gating. Different concerns.
6. **No caching**: User plan data is already fast to load. Caching can be added later if needed.

## Swagger & Handler Updates

- Add `@Failure 402 {object} responses.PaymentRequired` annotation to `Store` route
- Add `responsePaymentRequired` helper to base handler struct
- Update handler constructor to accept `*services.EntitlementService`

## Testing Strategy

- Unit test `EntitlementService.Check()` with:
  - Disabled mode → always allowed
  - Free user at limit → denied
  - Free user under limit → allowed
  - Paid user → always allowed
  - Unknown entity → allowed (no restrictions defined)
  - User load error when enabled → returns error
- Handler test for `Store`:
  - Free user with 0 schedules → 201 Created
  - Free user with 1 schedule → 402 Payment Required
  - Paid user with N schedules → 201 Created
