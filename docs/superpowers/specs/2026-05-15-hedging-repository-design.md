# Hedging Repository for Heartbeat & Monitor

**Date:** 2026-05-15
**Status:** Approved

## Overview

Create composite "hedging" repositories for `HeartbeatRepository` and `HeartbeatMonitorRepository` that write to both GORM (primary) and Turso (secondary). Reads only hit the primary. Secondary writes are fail-open — errors are logged and a metric is emitted, but the operation succeeds from the caller's perspective.

## Motivation

Gradually migrate heartbeat data to Turso by dual-writing. The GORM/PostgreSQL backend remains the source of truth while Turso builds up a complete dataset. If Turso has issues, the system is unaffected.

## Configuration

Activated via `HEARTBEAT_DB_BACKEND=hedging`. The three modes are now:

| Value             | Behavior                                                               |
| ----------------- | ---------------------------------------------------------------------- |
| _(unset/default)_ | GORM/PostgreSQL only                                                   |
| `turso`           | Turso/libSQL only                                                      |
| `hedging`         | GORM primary (reads+writes) + Turso secondary (writes only, fail-open) |

## Architecture

### New Files

| File                                                           | Purpose                                |
| -------------------------------------------------------------- | -------------------------------------- |
| `api/pkg/repositories/hedging_heartbeat_repository.go`         | Composite `HeartbeatRepository`        |
| `api/pkg/repositories/hedging_heartbeat_monitor_repository.go` | Composite `HeartbeatMonitorRepository` |

### Modified Files

| File                      | Change                                                             |
| ------------------------- | ------------------------------------------------------------------ |
| `api/pkg/di/container.go` | Add `hedging` case to switch, add `HedgingFailureCounter()` method |

## Method Delegation

### HeartbeatRepository

| Method             | Primary (GORM) | Secondary (Turso)    |
| ------------------ | -------------- | -------------------- |
| `Store`            | ✅ write       | ✅ write (fail-open) |
| `Index`            | ✅ read        | ❌ skip              |
| `Last`             | ✅ read        | ❌ skip              |
| `DeleteAllForUser` | ✅ write       | ✅ write (fail-open) |

### HeartbeatMonitorRepository

| Method              | Primary (GORM) | Secondary (Turso)    |
| ------------------- | -------------- | -------------------- |
| `Store`             | ✅ write       | ✅ write (fail-open) |
| `Load`              | ✅ read        | ❌ skip              |
| `Exists`            | ✅ read        | ❌ skip              |
| `UpdateQueueID`     | ✅ write       | ✅ write (fail-open) |
| `Delete`            | ✅ write       | ✅ write (fail-open) |
| `UpdatePhoneOnline` | ✅ write       | ✅ write (fail-open) |
| `DeleteAllForUser`  | ✅ write       | ✅ write (fail-open) |

## Struct Design

```go
type hedgingHeartbeatRepository struct {
    logger         telemetry.Logger
    tracer         telemetry.Tracer
    primary        HeartbeatRepository
    secondary      HeartbeatRepository
    failureCounter otelMetric.Int64Counter
}

type hedgingHeartbeatMonitorRepository struct {
    logger         telemetry.Logger
    tracer         telemetry.Tracer
    primary        HeartbeatMonitorRepository
    secondary      HeartbeatMonitorRepository
    failureCounter otelMetric.Int64Counter
}
```

## Error Handling (Fail-Open Pattern)

```go
func (r *hedgingHeartbeatRepository) Store(ctx context.Context, heartbeat *entities.Heartbeat) error {
    ctx, span := r.tracer.Start(ctx)
    defer span.End()

    // Primary: must succeed
    if err := r.primary.Store(ctx, heartbeat); err != nil {
        return err
    }

    // Secondary: fail-open (log + metric)
    if err := r.secondary.Store(ctx, heartbeat); err != nil {
        r.logger.Error(stacktrace.Propagate(err, fmt.Sprintf(
            "hedging: secondary write failed for heartbeat [%s]", heartbeat.ID,
        )))
        r.failureCounter.Add(ctx, 1)
    }

    return nil
}
```

**Rules:**

- Primary error → return immediately, no secondary attempt
- Secondary error → log at ERROR level, increment `failureCounter`, return nil
- Read methods → delegate directly to primary, no secondary involvement
- Each method has its own tracing span via `tracer.Start(ctx)`

## Observability

**Metric:**

- Name: `hedging.secondary.write.failures`
- Unit: `1` (count)
- Description: `Number of failed secondary writes in hedging repositories`
- Created once in DI container, shared by both hedging repos

**Logging:** Each secondary failure logs at ERROR with the method context (entity ID, user ID, etc.)

## DI Container Changes

```go
func (container *Container) HeartbeatRepository() repositories.HeartbeatRepository {
    switch os.Getenv("HEARTBEAT_DB_BACKEND") {
    case "turso":
        return repositories.NewLibsqlHeartbeatRepository(...)
    case "hedging":
        return repositories.NewHedgingHeartbeatRepository(
            container.Logger(),
            container.Tracer(),
            repositories.NewGormHeartbeatRepository(container.Logger(), container.Tracer(), container.DedicatedDB()),
            repositories.NewLibsqlHeartbeatRepository(container.Logger(), container.Tracer(), container.TursoDB()),
            container.HedgingFailureCounter(),
        )
    default:
        return repositories.NewGormHeartbeatRepository(...)
    }
}

func (container *Container) HedgingFailureCounter() otelMetric.Int64Counter {
    meter := otel.GetMeterProvider().Meter(container.projectID)
    counter, err := meter.Int64Counter("hedging.secondary.write.failures",
        otelMetric.WithUnit("1"),
        otelMetric.WithDescription("Number of failed secondary writes in hedging repositories"),
    )
    if err != nil {
        container.logger.Fatal(...)
    }
    return counter
}
```

Same switch pattern for `HeartbeatMonitorRepository()`.

## Safety

- Default behavior (no env var) is unchanged — pure GORM/PostgreSQL
- `turso` mode remains available for pure Turso usage
- `hedging` mode never fails due to Turso issues — secondary is fully fail-open
- Service layer requires no changes — same interfaces
