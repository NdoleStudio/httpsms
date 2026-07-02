# API Rate Limiting — Phase 1 (Tracking Only)

**Date:** 2026-06-30
**Status:** Draft
**Scope:** API backend (`api/`)

## Problem

The httpSMS API has no per-user rate limiting. Without it, a single user (free or paid) can
consume disproportionate resources. Before enforcing limits (returning 429), we need a
tracking-only phase to observe real usage patterns and validate thresholds.

## Requirements

1. **Per-user, plan-based daily limits** — each subscription tier gets a different
   requests-per-day budget equal to `2 × SubscriptionName.Limit()`.
2. **Weighted counting** — GET list endpoints with a `limit` query param count that value
   instead of 1 (e.g., `GET /v1/message-threads?limit=10` costs 10).
3. **Tracking only (Phase 1)** — the middleware never blocks requests. When a user exceeds
   their daily budget, a CloudEvent is emitted once per window.
4. **Optional** — controlled by `RATE_LIMIT_ENABLED` env var (default `false`). Self-hosted
   users are unaffected.
5. **Selective** — certain paths are excluded from rate limiting (e.g., `/v1/events` which is
   called only by the system user).
6. **Cost-efficient** — the API receives ~20M requests/month and is hosted on Google Cloud Run
   (instances restart every ~2 minutes). Per-request Redis calls are too expensive. Counters
   live in memory and are batch-synced to Redis every 30 seconds.

## Plan Tier Rate Limits

| Plan | `Limit()` (messages) | `RateLimit()` (requests/day) |
|------|---------------------|------------------------------|
| Free | 200 | 400 |
| Pro  | 5,000 | 10,000 |
| Ultra | 10,000 | 20,000 |
| 20K  | 20,000 | 40,000 |
| 50K  | 50,000 | 100,000 |
| 100K | 100,000 | 200,000 |
| 200K | 200,000 | 400,000 |

## Architecture

```
Request → Auth Middleware → Rate Limit Middleware → Handler
                                   │
                           ┌───────┴────────┐
                           │  In-Memory Map  │
                           │  (hot counter)  │
                           └───────┬────────┘
                                   │ every 30s
                           ┌───────┴────────┐
                           │     Redis       │
                           │  (persistent)   │
                           └────────────────┘
```

### Request Flow

1. **Skip check**: if `RATE_LIMIT_ENABLED != "true"` or path is in the exclude list, call
   `c.Next()` immediately.
2. **Extract identity**: read `AuthContext` from `c.Locals(ContextKeyAuthUserID)`. If not
   authenticated (noop context), skip.
3. **Compute cost**: if the request is a GET with a `limit` query param, use
   `min(parsedLimit, 200)` as the cost. Otherwise, cost = 1.
4. **Increment counter**: call `RateLimitService.Increment(ctx, userID, plan, cost)`.
5. **Check threshold**: if the returned count exceeds `plan.RateLimit()` and no event has
   been emitted for this user in this window, emit a `rate.limit.exceeded` CloudEvent.
   The "already notified" flag is stored in Redis (`rate_limit_notified:{user_id}` with 24h
   TTL) so it survives instance restarts.
6. **Continue**: always call `c.Next()` — never block in Phase 1.

### Hydration on Cold Start

Cloud Run instances restart every ~2 minutes. When a user's first request hits a new instance:

1. Read current count from Redis: `GET rate_limit:{user_id}` and `TTL rate_limit:{user_id}`.
2. Populate in-memory entry with the Redis count and remaining TTL as the window expiry.
3. Increment locally, mark delta as dirty.

This means ~2 Redis reads per active user per instance startup. With batched writes every 30s,
total Redis operations are reduced from 20M/month to roughly 200K–400K/month.

### Concurrent Instance Safety

Multiple Cloud Run instances may run simultaneously. Safety is ensured by:

- **Lazy load via `GET`**: each instance reads the latest Redis value on first access per user.
- **Atomic flush via `INCRBY`**: dirty deltas are flushed with `INCRBY`, not `SET`. This is
  atomic and additive — concurrent flushes from different instances all contribute correctly.
- **TTL management**: the first instance to create a key sets a 24h TTL. Subsequent `INCRBY`
  calls do not reset the TTL (Redis preserves TTL on INCRBY).

## Components

### New Files

#### `pkg/services/rate_limit_service.go`

Core rate limiting logic with in-memory counters and periodic Redis sync.

```go
type RateLimitService struct {
    tracer     telemetry.Tracer
    logger     telemetry.Logger
    client     *redis.Client
    dispatcher *EventDispatcher

    mu       sync.Mutex
    counters map[string]*userCounter
    notified map[string]bool          // in-memory cache; authoritative flag is in Redis
    done     chan struct{}
}

type userCounter struct {
    count       int64
    windowExpiry time.Time
    dirty       int64
}
```

**Methods:**

- `NewRateLimitService(tracer, logger, client, dispatcher) *RateLimitService` — starts
  background flush goroutine (every 30s).
- `Increment(ctx, userID string, plan SubscriptionName, cost int) (count int64, exceeded bool, err error)` —
  increments counter, returns current count and whether the limit is exceeded.
- `Close()` — flushes remaining dirty counters and stops the background goroutine. Called on
  graceful shutdown.
- `flush(ctx)` — iterates all dirty counters, calls `INCRBY` for each, resets dirty to 0.
  Expired windows (past 24h) are cleaned up during flush.
- `hydrate(ctx, userID string) (*userCounter, error)` — reads `GET` + `TTL` from Redis,
  returns populated counter. If key doesn't exist, returns a fresh counter with
  `windowExpiry = now + 24h`.

#### `pkg/middlewares/rate_limit_middleware.go`

Fiber middleware that wraps `RateLimitService`.

```go
func RateLimit(
    tracer telemetry.Tracer,
    service *services.RateLimitService,
    excludePaths []string,
) fiber.Handler
```

- Reads `RATE_LIMIT_ENABLED` env var once at creation (not per-request).
- Uses prefix matching for exclude paths.
- Extracts cost from `limit` query param on GET requests, capped at 200.

#### `pkg/events/rate_limit_exceeded_event.go`

```go
const RateLimitExceeded = "rate.limit.exceeded"

type RateLimitExceededPayload struct {
    UserID    string `json:"user_id"`
    Count     int64  `json:"count"`
    Limit     uint   `json:"limit"`
    Plan      string `json:"plan"`
}
```

### Modified Files

#### `pkg/entities/user.go`

Add method:

```go
func (subscription SubscriptionName) RateLimit() uint {
    return subscription.Limit() * 2
}
```

#### `pkg/di/container.go`

- Add `RedisClient() *redis.Client` method — extracts Redis client creation from `Cache()`
  so both `Cache` and `RateLimitService` share the same connection.
- Add `RateLimitService() *services.RateLimitService` method.
- Register the rate limit middleware after API key auth in `App()`:
  ```go
  app.Use(middlewares.RateLimit(
      container.Tracer(),
      container.RateLimitService(),
      []string{"/v1/events"},
  ))
  ```

#### `.env.docker` / `.env.example`

Add:
```
RATE_LIMIT_ENABLED=false
```

## Excluded Paths

The following paths are excluded from rate limiting:

- `/v1/events` — called only by the system user
- `/health` — registered before all middleware, naturally excluded

## Testing Strategy

- Unit tests for `RateLimitService`: increment, window expiry, hydration, flush, weighted
  cost, concurrent access.
- Unit test for `SubscriptionName.RateLimit()`.
- Integration test for the middleware: verify it skips excluded paths, skips when disabled,
  computes correct cost, and emits events on threshold breach.

## Future Work (Phase 2)

- Return `429 Too Many Requests` when the limit is exceeded.
- Add `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` response headers.
- Dashboard/UI for users to see their current usage.
- Admin override to adjust individual user limits.
