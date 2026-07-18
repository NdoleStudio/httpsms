# API Rate Limiting (Phase 1 — Tracking Only) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-user, plan-based API rate limiting that tracks usage in-memory with periodic Redis sync, emitting a CloudEvent when limits are exceeded — without blocking any requests.

**Architecture:** A custom Fiber middleware increments an in-memory counter per user per request (with weighted costs for list endpoints). A background goroutine flushes dirty counters to Redis every 30 seconds via atomic `INCRBY`. On cold start, counters are lazily hydrated from Redis. When a user exceeds their plan's daily limit, a `rate.limit.exceeded` CloudEvent is dispatched (deduplicated via Redis flag).

**Tech Stack:** Go, Fiber v3, go-redis/v9, CloudEvents, OpenTelemetry tracing

## Global Constraints

- Go 1.21+ (module: `github.com/NdoleStudio/httpsms`)
- Error handling: always wrap with `github.com/palantir/stacktrace`
- Tracing: pass `telemetry.Tracer` and start spans in all public methods
- Redis key prefix: `rate_limit:`
- Rate limit window: 24 hours (sliding, based on Redis TTL)
- Rate limit budget: `SubscriptionName.Limit() * 2` requests per day
- Environment variable: `RATE_LIMIT_ENABLED` (default `"false"`)
- Excluded paths: `/v1/events`
- Phase 1: never return 429, always call `c.Next()`

---

### Task 1: Add `RateLimit()` Method to `SubscriptionName`

**Files:**
- Modify: `api/pkg/entities/user.go:38` (after `Limit()` method)
- Test: `api/pkg/entities/user_test.go`

**Interfaces:**
- Consumes: existing `SubscriptionName.Limit() uint`
- Produces: `SubscriptionName.RateLimit() uint` — returns `Limit() * 2`

- [ ] **Step 1: Write the failing test**

Add to `api/pkg/entities/user_test.go`:

```go
func TestSubscriptionName_RateLimit_Free(t *testing.T) {
	assert.Equal(t, uint(400), SubscriptionNameFree.RateLimit())
}

func TestSubscriptionName_RateLimit_Pro(t *testing.T) {
	assert.Equal(t, uint(10000), SubscriptionNameProMonthly.RateLimit())
}

func TestSubscriptionName_RateLimit_Ultra(t *testing.T) {
	assert.Equal(t, uint(20000), SubscriptionNameUltraMonthly.RateLimit())
}

func TestSubscriptionName_RateLimit_200K(t *testing.T) {
	assert.Equal(t, uint(400000), SubscriptionName200KMonthly.RateLimit())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd api && go test ./pkg/entities/ -run TestSubscriptionName_RateLimit -v`
Expected: FAIL — `RateLimit` method not found

- [ ] **Step 3: Write minimal implementation**

Add to `api/pkg/entities/user.go` after the `Limit()` method (after line 38):

```go
// RateLimit returns the daily API request rate limit for a subscription
func (subscription SubscriptionName) RateLimit() uint {
	return subscription.Limit() * 2
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd api && go test ./pkg/entities/ -run TestSubscriptionName_RateLimit -v`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Commit**

```bash
cd api && git add pkg/entities/user.go pkg/entities/user_test.go
git commit -m "feat(entities): add RateLimit() method to SubscriptionName"
```

---

### Task 2: Add `rate.limit.exceeded` CloudEvent

**Files:**
- Create: `api/pkg/events/rate_limit_exceeded_event.go`

**Interfaces:**
- Consumes: `entities.UserID`
- Produces: `events.RateLimitExceeded` constant, `events.RateLimitExceededPayload` struct

- [ ] **Step 1: Create the event file**

Create `api/pkg/events/rate_limit_exceeded_event.go`:

```go
package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// RateLimitExceeded is raised when a user exceeds their daily API rate limit.
const RateLimitExceeded = "rate.limit.exceeded"

// RateLimitExceededPayload stores the data for the RateLimitExceeded event
type RateLimitExceededPayload struct {
	UserID    entities.UserID `json:"user_id"`
	Count     int64           `json:"count"`
	Limit     uint            `json:"limit"`
	Plan      string          `json:"plan"`
	Timestamp time.Time       `json:"timestamp"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd api && go build ./pkg/events/`
Expected: Success (exit code 0)

- [ ] **Step 3: Commit**

```bash
cd api && git add pkg/events/rate_limit_exceeded_event.go
git commit -m "feat(events): add rate.limit.exceeded CloudEvent type"
```

---

### Task 3: Implement `RateLimitService`

**Files:**
- Create: `api/pkg/services/rate_limit_service.go`
- Create: `api/pkg/services/rate_limit_service_test.go`

**Interfaces:**
- Consumes: `telemetry.Tracer`, `telemetry.Logger`, `*redis.Client`, `*EventDispatcher`, `entities.SubscriptionName.RateLimit() uint`, `events.RateLimitExceeded`, `events.RateLimitExceededPayload`
- Produces:
  - `NewRateLimitService(tracer telemetry.Tracer, logger telemetry.Logger, client *redis.Client, dispatcher *EventDispatcher) *RateLimitService`
  - `(*RateLimitService).Increment(ctx context.Context, userID entities.UserID, plan entities.SubscriptionName, cost int) (count int64, exceeded bool, err error)`
  - `(*RateLimitService).Close()`

- [ ] **Step 1: Write the failing tests**

Create `api/pkg/services/rate_limit_service_test.go`:

```go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimitService_Increment_BasicCount(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Act
	count, exceeded, err := svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.False(t, exceeded)
}

func TestRateLimitService_Increment_WeightedCost(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Act
	count, _, err := svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 10)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)
}

func TestRateLimitService_Increment_ExceedsLimit(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Free plan limit is 400. Exceed it.
	for i := 0; i < 400; i++ {
		_, _, _ = svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 1)
	}

	// Act — this pushes count to 401
	count, exceeded, err := svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(401), count)
	assert.True(t, exceeded)
}

func TestRateLimitService_Increment_MultipleUsers(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Act
	_, _, _ = svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 5)
	count, _, err := svc.Increment(ctx, "user-2", entities.SubscriptionNameProMonthly, 3)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestRateLimitService_Increment_WindowExpiry(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Simulate an existing counter with an expired window
	svc.mu.Lock()
	svc.counters["user-1"] = &userCounter{
		count:        500,
		windowExpiry: time.Now().Add(-1 * time.Hour), // expired
		dirty:        0,
	}
	svc.mu.Unlock()

	// Act — should reset because the window expired
	count, exceeded, err := svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.False(t, exceeded)
}

// newTestRateLimitService creates a RateLimitService with nil redis client (no hydration)
// suitable for unit tests that only test in-memory logic.
func newTestRateLimitService(t *testing.T) *RateLimitService {
	t.Helper()
	return NewRateLimitService(nil, nil, nil, nil)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd api && go test ./pkg/services/ -run TestRateLimitService -v`
Expected: FAIL — `RateLimitService` type not found

- [ ] **Step 3: Write the implementation**

Create `api/pkg/services/rate_limit_service.go`:

```go
package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"github.com/redis/go-redis/v9"
)

const (
	rateLimitKeyPrefix     = "rate_limit:"
	rateLimitNotifiedPrefix = "rate_limit_notified:"
	rateLimitWindow        = 24 * time.Hour
	rateLimitFlushInterval = 30 * time.Second
)

// RateLimitService tracks per-user API request counts with in-memory counters
// and periodic Redis sync.
type RateLimitService struct {
	service
	tracer     telemetry.Tracer
	logger     telemetry.Logger
	client     *redis.Client
	dispatcher *EventDispatcher

	mu       sync.Mutex
	counters map[string]*userCounter
	notified map[string]bool
	done     chan struct{}
}

type userCounter struct {
	count        int64
	windowExpiry time.Time
	dirty        int64
}

// NewRateLimitService creates a new RateLimitService and starts the background flush goroutine.
func NewRateLimitService(
	tracer telemetry.Tracer,
	logger telemetry.Logger,
	client *redis.Client,
	dispatcher *EventDispatcher,
) *RateLimitService {
	svc := &RateLimitService{
		tracer:     tracer,
		logger:     logger,
		client:     client,
		dispatcher: dispatcher,
		counters:   make(map[string]*userCounter),
		notified:   make(map[string]bool),
		done:       make(chan struct{}),
	}

	go svc.flushLoop()
	return svc
}

// Increment adds cost to the user's counter and returns the current count.
// If the count exceeds the plan's rate limit, exceeded is true.
func (svc *RateLimitService) Increment(ctx context.Context, userID entities.UserID, plan entities.SubscriptionName, cost int) (count int64, exceeded bool, err error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	key := string(userID)
	counter, exists := svc.counters[key]

	if !exists {
		counter, err = svc.hydrate(ctx, key)
		if err != nil {
			counter = &userCounter{
				count:        0,
				windowExpiry: time.Now().Add(rateLimitWindow),
				dirty:        0,
			}
		}
		svc.counters[key] = counter
	}

	// Reset if window has expired
	if time.Now().After(counter.windowExpiry) {
		counter.count = 0
		counter.dirty = 0
		counter.windowExpiry = time.Now().Add(rateLimitWindow)
		svc.notified[key] = false
	}

	counter.count += int64(cost)
	counter.dirty += int64(cost)

	limit := plan.RateLimit()
	exceeded = counter.count > int64(limit)

	if exceeded && !svc.notified[key] {
		svc.notified[key] = true
		go svc.emitExceededEvent(ctx, userID, counter.count, limit, plan)
	}

	return counter.count, exceeded, nil
}

// Close flushes remaining dirty counters and stops the background goroutine.
func (svc *RateLimitService) Close() {
	close(svc.done)
	if svc.client != nil {
		svc.flush(context.Background())
	}
}

func (svc *RateLimitService) flushLoop() {
	ticker := time.NewTicker(rateLimitFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if svc.client != nil {
				svc.flush(context.Background())
			}
		case <-svc.done:
			return
		}
	}
}

func (svc *RateLimitService) flush(ctx context.Context) {
	svc.mu.Lock()
	// Collect dirty entries
	type flushEntry struct {
		key   string
		delta int64
		ttl   time.Duration
	}
	var entries []flushEntry
	now := time.Now()

	for key, counter := range svc.counters {
		// Clean up expired windows
		if now.After(counter.windowExpiry) {
			delete(svc.counters, key)
			delete(svc.notified, key)
			continue
		}
		if counter.dirty > 0 {
			entries = append(entries, flushEntry{
				key:   rateLimitKeyPrefix + key,
				delta: counter.dirty,
				ttl:   time.Until(counter.windowExpiry),
			})
			counter.dirty = 0
		}
	}
	svc.mu.Unlock()

	// Flush to Redis outside the lock
	for _, entry := range entries {
		pipe := svc.client.Pipeline()
		pipe.IncrBy(ctx, entry.key, entry.delta)
		pipe.ExpireNX(ctx, entry.key, entry.ttl)
		if _, err := pipe.Exec(ctx); err != nil {
			if svc.logger != nil {
				svc.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot flush rate limit for key [%s]", entry.key)))
			}
		}
	}
}

func (svc *RateLimitService) hydrate(ctx context.Context, userID string) (*userCounter, error) {
	if svc.client == nil {
		return &userCounter{
			count:        0,
			windowExpiry: time.Now().Add(rateLimitWindow),
			dirty:        0,
		}, nil
	}

	key := rateLimitKeyPrefix + userID
	countStr, err := svc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return &userCounter{
			count:        0,
			windowExpiry: time.Now().Add(rateLimitWindow),
			dirty:        0,
		}, nil
	}
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot hydrate rate limit for user [%s]", userID))
	}

	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot parse rate limit count [%s] for user [%s]", countStr, userID))
	}

	ttl, err := svc.client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = rateLimitWindow
	}

	// Check if already notified (in Redis)
	notifiedKey := rateLimitNotifiedPrefix + userID
	if exists, _ := svc.client.Exists(ctx, notifiedKey).Result(); exists > 0 {
		svc.notified[userID] = true
	}

	return &userCounter{
		count:        count,
		windowExpiry: time.Now().Add(ttl),
		dirty:        0,
	}, nil
}

func (svc *RateLimitService) emitExceededEvent(ctx context.Context, userID entities.UserID, count int64, limit uint, plan entities.SubscriptionName) {
	if svc.dispatcher == nil {
		return
	}

	// Set notified flag in Redis (24h TTL) to survive restarts
	if svc.client != nil {
		notifiedKey := rateLimitNotifiedPrefix + string(userID)
		svc.client.Set(ctx, notifiedKey, "1", rateLimitWindow)
	}

	payload := events.RateLimitExceededPayload{
		UserID:    userID,
		Count:     count,
		Limit:     limit,
		Plan:      string(plan),
		Timestamp: time.Now().UTC(),
	}

	event, err := svc.createEvent(events.RateLimitExceeded, string(userID), payload)
	if err != nil {
		if svc.logger != nil {
			svc.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot create rate limit exceeded event for user [%s]", userID)))
		}
		return
	}

	if err = svc.dispatcher.Dispatch(ctx, event); err != nil {
		if svc.logger != nil {
			svc.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch rate limit exceeded event for user [%s]", userID)))
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd api && go test ./pkg/services/ -run TestRateLimitService -v`
Expected: PASS (all 5 tests)

- [ ] **Step 5: Commit**

```bash
cd api && git add pkg/services/rate_limit_service.go pkg/services/rate_limit_service_test.go
git commit -m "feat(services): implement RateLimitService with in-memory counters and Redis sync"
```

---

### Task 4: Implement Rate Limit Middleware

**Files:**
- Create: `api/pkg/middlewares/rate_limit_middleware.go`

**Interfaces:**
- Consumes: `telemetry.Tracer`, `*services.RateLimitService`, `entities.AuthContext` (from `c.Locals`), `entities.SubscriptionName`
- Produces: `middlewares.RateLimit(tracer telemetry.Tracer, logger telemetry.Logger, service *services.RateLimitService, userRepository repositories.UserRepository, excludePaths []string) fiber.Handler`

- [ ] **Step 1: Create the middleware**

Create `api/pkg/middlewares/rate_limit_middleware.go`:

```go
package middlewares

import (
	"os"
	"strconv"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v3"
)

const rateLimitCostCap = 200

// RateLimit tracks per-user API request counts without blocking requests.
func RateLimit(
	tracer telemetry.Tracer,
	logger telemetry.Logger,
	service *services.RateLimitService,
	userRepository repositories.UserRepository,
	excludePaths []string,
) fiber.Handler {
	enabled := os.Getenv("RATE_LIMIT_ENABLED") == "true"
	logger = logger.WithService("middlewares.RateLimit")

	return func(c fiber.Ctx) error {
		if !enabled {
			return c.Next()
		}

		// Check excluded paths
		path := c.Path()
		for _, excluded := range excludePaths {
			if strings.HasPrefix(path, excluded) {
				return c.Next()
			}
		}

		ctx, span := tracer.StartFromFiberCtx(c, "middlewares.RateLimit")
		defer span.End()

		// Extract authenticated user
		authUser, ok := c.Locals(ContextKeyAuthUserID).(entities.AuthContext)
		if !ok || authUser.IsNoop() {
			return c.Next()
		}

		// Compute cost
		cost := 1
		if c.Method() == fiber.MethodGet {
			if limitParam := c.Query("limit"); limitParam != "" {
				if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
					cost = parsed
					if cost > rateLimitCostCap {
						cost = rateLimitCostCap
					}
				}
			}
		}

		// Load user's subscription plan
		user, err := userRepository.Load(ctx, authUser.ID)
		if err != nil {
			ctxLogger := tracer.CtxLogger(logger, span)
			ctxLogger.Error(err)
			return c.Next()
		}

		// Increment rate limit counter
		_, _, _ = service.Increment(ctx, authUser.ID, user.SubscriptionName, cost)

		return c.Next()
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd api && go build ./pkg/middlewares/`
Expected: Success (exit code 0)

- [ ] **Step 3: Commit**

```bash
cd api && git add pkg/middlewares/rate_limit_middleware.go
git commit -m "feat(middlewares): add rate limit middleware for tracking API usage"
```

---

### Task 5: Wire Everything in the DI Container and Add Env Config

**Files:**
- Modify: `api/pkg/di/container.go:85-99` (add `rateLimitService` and `redisClient` fields to Container struct)
- Modify: `api/pkg/di/container.go:443-469` (extract Redis client creation)
- Modify: `api/pkg/di/container.go:203-206` (add rate limit middleware to the chain)
- Modify: `api/.env.docker`

**Interfaces:**
- Consumes: `services.NewRateLimitService(...)`, `middlewares.RateLimit(...)`, all existing container methods
- Produces: `Container.RedisClient() *redis.Client`, `Container.RateLimitService() *services.RateLimitService`

- [ ] **Step 1: Add fields to the Container struct**

In `api/pkg/di/container.go`, add to the `Container` struct (after line 98, `inMemoryCache cache.Cache`):

```go
	rateLimitService *services.RateLimitService
	redisClient      *redis.Client
```

- [ ] **Step 2: Add `RedisClient()` method**

Add after the existing `Cache()` method (after line 469):

```go
// RedisClient creates or returns the shared *redis.Client
func (container *Container) RedisClient() *redis.Client {
	if container.redisClient != nil {
		return container.redisClient
	}

	container.logger.Debug("creating *redis.Client")
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot parse redis url [%s]", os.Getenv("REDIS_URL"))))
	}
	if strings.HasPrefix(os.Getenv("REDIS_URL"), "rediss://") {
		opt.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	container.redisClient = redis.NewClient(opt)

	if err = redisotel.InstrumentTracing(container.redisClient); err != nil {
		container.logger.Error(stacktrace.Propagate(err, "cannot instrument redis tracing"))
	}
	if err = redisotel.InstrumentMetrics(container.redisClient); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, "cannot instrument redis metrics"))
	}

	return container.redisClient
}
```

- [ ] **Step 3: Refactor `Cache()` to use `RedisClient()`**

Replace the body of `Cache()` to reuse the shared client:

```go
// Cache creates a new instance of cache.Cache
func (container *Container) Cache() cache.Cache {
	container.logger.Debug("creating cache.Cache")
	return cache.NewRedisCache(container.Tracer(), container.RedisClient())
}
```

- [ ] **Step 4: Add `RateLimitService()` method**

Add after `RedisClient()`:

```go
// RateLimitService creates or returns the shared *services.RateLimitService
func (container *Container) RateLimitService() *services.RateLimitService {
	if container.rateLimitService != nil {
		return container.rateLimitService
	}

	container.logger.Debug("creating services.RateLimitService")
	container.rateLimitService = services.NewRateLimitService(
		container.Tracer(),
		container.Logger(),
		container.RedisClient(),
		container.EventDispatcher(),
	)
	return container.rateLimitService
}
```

- [ ] **Step 5: Register middleware in `App()`**

In the `App()` method, add the rate limit middleware after the API key auth line (`app.Use(middlewares.APIKeyAuth(...))`). Insert after line 205:

```go
	app.Use(middlewares.RateLimit(
		container.Tracer(),
		container.Logger(),
		container.RateLimitService(),
		container.UserRepository(),
		[]string{"/v1/events"},
	))
```

- [ ] **Step 6: Add `RATE_LIMIT_ENABLED` to `.env.docker`**

Add after the `REDIS_URL` line in `api/.env.docker`:

```
# Rate limiting (set to "true" to enable per-user API rate tracking)
RATE_LIMIT_ENABLED=false
```

- [ ] **Step 7: Verify it compiles and tests pass**

Run: `cd api && go build . && go test ./...`
Expected: Build succeeds, all tests pass

- [ ] **Step 8: Commit**

```bash
cd api && git add pkg/di/container.go .env.docker
git commit -m "feat(di): wire rate limit service and middleware into DI container"
```

---

### Task 6: End-to-End Verification

**Files:**
- No new files — verification only

**Interfaces:**
- Consumes: all previous tasks

- [ ] **Step 1: Run the full test suite**

Run: `cd api && go test ./... -count=1`
Expected: All tests pass

- [ ] **Step 2: Build the binary**

Run: `cd api && go build -o ./tmp/main.exe .`
Expected: Build succeeds

- [ ] **Step 3: Verify Docker Compose build (optional, if Docker available)**

Run: `docker compose build api`
Expected: Build succeeds

- [ ] **Step 4: Final commit (if any formatting fixes needed)**

Run: `cd api && go fmt ./... && go vet ./...`
If changes: commit with `style: format rate limiting code`
