# Hedging Repository Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Status:** All tasks implemented. This plan was written retroactively to document and verify the implementation.

**Goal:** Create composite hedging repositories that dual-write to GORM (primary) and Turso (secondary) with fail-open semantics on secondary failures.

**Architecture:** Two new repository files implement the existing `HeartbeatRepository` and `HeartbeatMonitorRepository` interfaces by delegating reads to primary only and writes to both. Secondary write failures are logged and counted via an OTel metric but never propagated. Activated via `HEARTBEAT_DB_BACKEND=hedging`.

**Tech Stack:** Go, OpenTelemetry metrics (`go.opentelemetry.io/otel/metric`), existing repository interfaces

**Spec:** `docs/superpowers/specs/2026-05-15-hedging-repository-design.md`

---

### Task 1: Create hedging heartbeat repository ✅

**Files:**

- Created: `api/pkg/repositories/hedging_heartbeat_repository.go`

- [ ] **Step 1: Create the hedging heartbeat repository file**

```go
package repositories

import (
	"context"
	"fmt"

	otelMetric "go.opentelemetry.io/otel/metric"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

type hedgingHeartbeatRepository struct {
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	primary        HeartbeatRepository
	secondary      HeartbeatRepository
	failureCounter otelMetric.Int64Counter
}

func NewHedgingHeartbeatRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	primary HeartbeatRepository,
	secondary HeartbeatRepository,
	failureCounter otelMetric.Int64Counter,
) HeartbeatRepository {
	return &hedgingHeartbeatRepository{
		logger:         logger.WithService(fmt.Sprintf("%T", &hedgingHeartbeatRepository{})),
		tracer:         tracer,
		primary:        primary,
		secondary:      secondary,
		failureCounter: failureCounter,
	}
}
```

Implement 4 methods:

- `Store` — write to primary, then secondary (fail-open with log + metric)
- `Index` — delegate to primary only
- `Last` — delegate to primary only
- `DeleteAllForUser` — write to primary, then secondary (fail-open with log + metric)

Write methods follow this pattern:

```go
func (r *hedgingHeartbeatRepository) Store(ctx context.Context, heartbeat *entities.Heartbeat) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()

	if err := r.primary.Store(ctx, heartbeat); err != nil {
		return err
	}

	if err := r.secondary.Store(ctx, heartbeat); err != nil {
		r.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("hedging: secondary write failed for heartbeat [%s]", heartbeat.ID)))
		r.failureCounter.Add(ctx, 1)
	}

	return nil
}
```

Read methods simply delegate:

```go
func (r *hedgingHeartbeatRepository) Index(ctx context.Context, userID entities.UserID, owner string, params IndexParams) (*[]entities.Heartbeat, error) {
	return r.primary.Index(ctx, userID, owner, params)
}
```

- [ ] **Step 2: Verify build**

Run: `cd api && go build ./...`
Expected: exit code 0

- [ ] **Step 3: Commit**

```bash
git add api/pkg/repositories/hedging_heartbeat_repository.go
git commit -m "feat(api): add hedging heartbeat repository"
```

---

### Task 2: Create hedging heartbeat monitor repository ✅

**Files:**

- Created: `api/pkg/repositories/hedging_heartbeat_monitor_repository.go`

- [ ] **Step 1: Create the hedging heartbeat monitor repository file**

Same struct pattern as Task 1 but wrapping `HeartbeatMonitorRepository` interface.

Implement 7 methods:

| Method              | Behavior               |
| ------------------- | ---------------------- |
| `Store`             | Write both (fail-open) |
| `Load`              | Primary only           |
| `Exists`            | Primary only           |
| `UpdateQueueID`     | Write both (fail-open) |
| `Delete`            | Write both (fail-open) |
| `UpdatePhoneOnline` | Write both (fail-open) |
| `DeleteAllForUser`  | Write both (fail-open) |

All write methods follow the same fail-open pattern: primary must succeed, secondary logs + increments counter on failure.

- [ ] **Step 2: Verify build**

Run: `cd api && go build ./...`
Expected: exit code 0

- [ ] **Step 3: Commit**

```bash
git add api/pkg/repositories/hedging_heartbeat_monitor_repository.go
git commit -m "feat(api): add hedging heartbeat monitor repository"
```

---

### Task 3: Add HedgingFailureCounter to DI container ✅

**Files:**

- Modified: `api/pkg/di/container.go` (added `HedgingFailureCounter()` method after `TursoDB()`, ~line 320)

- [ ] **Step 1: Add the HedgingFailureCounter method**

```go
func (container *Container) HedgingFailureCounter() otelMetric.Int64Counter {
	meter := otel.GetMeterProvider().Meter(
		container.projectID,
		otelMetric.WithInstrumentationVersion(otel.Version()),
	)
	counter, err := meter.Int64Counter(
		"hedging.secondary.write.failures",
		otelMetric.WithUnit("1"),
		otelMetric.WithDescription("Number of failed secondary writes in hedging repositories"),
	)
	if err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, "cannot create hedging failure counter"))
	}
	return counter
}
```

- [ ] **Step 2: Verify build**

Run: `cd api && go build ./...`
Expected: exit code 0

---

### Task 4: Wire hedging mode in DI container ✅

**Files:**

- Modified: `api/pkg/di/container.go`

  - `HeartbeatRepository()` (~line 1768)
  - `HeartbeatMonitorRepository()` (~line 930)

- [ ] **Step 1: Change both methods from if/else to switch**

Replace the existing `if os.Getenv("HEARTBEAT_DB_BACKEND") == "turso"` with a switch:

```go
func (container *Container) HeartbeatRepository() repositories.HeartbeatRepository {
	switch os.Getenv("HEARTBEAT_DB_BACKEND") {
	case "turso":
		// existing libSQL path
	case "hedging":
		return repositories.NewHedgingHeartbeatRepository(
			container.Logger(),
			container.Tracer(),
			repositories.NewGormHeartbeatRepository(container.Logger(), container.Tracer(), container.DedicatedDB()),
			repositories.NewLibsqlHeartbeatRepository(container.Logger(), container.Tracer(), container.TursoDB()),
			container.HedgingFailureCounter(),
		)
	default:
		// existing GORM path
	}
}
```

Same pattern for `HeartbeatMonitorRepository()`.

- [ ] **Step 2: Verify build**

Run: `cd api && go build ./...`
Expected: exit code 0

- [ ] **Step 3: Run pre-commit hooks**

Run: `cd api && gofumpt -w pkg/repositories/hedging_heartbeat_repository.go pkg/repositories/hedging_heartbeat_monitor_repository.go pkg/di/container.go`
Expected: exit code 0

- [ ] **Step 4: Final commit**

```bash
git add api/pkg/di/container.go
git commit -m "feat(api): wire hedging mode in DI container (HEARTBEAT_DB_BACKEND=hedging)"
```

---

### Task 5: Final verification

- [ ] **Step 1: Full build**

Run: `cd api && go build ./...`
Expected: exit code 0

- [ ] **Step 2: Run tests**

Run: `cd api && go test ./...`
Expected: all tests pass

- [ ] **Step 3: go vet (no new warnings)**

Run: `cd api && go vet ./... 2>&1 | Select-String "hedging"`
Expected: only pre-existing `non-constant format string` warnings (same category as all other repos)

- [ ] **Step 4: Pre-commit hooks pass**

Run: `git add -A && git commit --dry-run`
Expected: all hooks pass (go-fumpt, go-lint, go-imports, go-mod-tidy)

- [ ] **Step 5: Verify three modes work (code review)**

Check that the DI container correctly handles all three values:

- Default (unset) → GORM only
- `turso` → libSQL only
- `hedging` → GORM primary + libSQL secondary
