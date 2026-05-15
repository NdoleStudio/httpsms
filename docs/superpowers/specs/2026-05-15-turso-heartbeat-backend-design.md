# Turso/libSQL Backend for Heartbeat & Monitor Repositories

**Date:** 2026-05-15
**Status:** Approved

## Overview

Add a libSQL/Turso alternative implementation for `HeartbeatRepository` and `HeartbeatMonitorRepository`, switchable via the `HEARTBEAT_DB_BACKEND` environment variable. When set to `turso`, the API connects to a cloud-hosted Turso database instead of the dedicated PostgreSQL instance.

## Motivation

Move heartbeat storage to a dedicated Turso database for cost efficiency and edge performance, while keeping the existing PostgreSQL path as the default fallback.

## Configuration

| Env Var                | Purpose                                                        | Example                                               |
| ---------------------- | -------------------------------------------------------------- | ----------------------------------------------------- |
| `HEARTBEAT_DB_BACKEND` | Selects backend (`turso` = libSQL, anything else = PostgreSQL) | `turso`                                               |
| `TURSO_DATABASE_URL`   | Turso database URL                                             | `libsql://httpsms-ndolestudio.aws-us-east-1.turso.io` |
| `TURSO_AUTH_TOKEN`     | Turso auth token                                               | `eyJ...`                                              |

When `HEARTBEAT_DB_BACKEND` is not set or is any value other than `turso`, the existing GORM/PostgreSQL path is used unchanged.

## Architecture

### Approach

Direct `database/sql` with the official Turso Go driver (`github.com/tursodatabase/go-libsql`). No ORM — raw SQL queries for a simple 2-table schema.

### New Files

| File                                                          | Purpose                                                  |
| ------------------------------------------------------------- | -------------------------------------------------------- |
| `api/pkg/repositories/libsql.go`                              | Connection factory, table auto-creation, shared helpers  |
| `api/pkg/repositories/libsql_heartbeat_repository.go`         | `HeartbeatRepository` implementation using libSQL        |
| `api/pkg/repositories/libsql_heartbeat_monitor_repository.go` | `HeartbeatMonitorRepository` implementation using libSQL |

### Modified Files

| File                      | Change                                                                                                                            |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `api/pkg/di/container.go` | Add `tursoDB *sql.DB` field, `TursoDB()` method, conditional wiring in `HeartbeatRepository()` and `HeartbeatMonitorRepository()` |
| `api/go.mod`              | Add `github.com/tursodatabase/go-libsql` dependency                                                                               |

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS heartbeats (
    id TEXT PRIMARY KEY,
    owner TEXT NOT NULL,
    version TEXT NOT NULL,
    charging INTEGER NOT NULL DEFAULT 0,
    user_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_heartbeats_owner_timestamp ON heartbeats(owner, timestamp);
CREATE INDEX IF NOT EXISTS idx_heartbeats_user_id ON heartbeats(user_id);

CREATE TABLE IF NOT EXISTS heartbeat_monitors (
    id TEXT PRIMARY KEY,
    phone_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    queue_id TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL,
    phone_online INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_heartbeat_monitors_user_owner ON heartbeat_monitors(user_id, owner);
```

**Type mappings from PostgreSQL:**

- UUID → TEXT
- BOOLEAN → INTEGER (0/1)
- TIMESTAMP → DATETIME (ISO 8601 text)

## Repository Implementations

### `libsql.go` (shared)

- `NewTursoDB(url, authToken string) (*sql.DB, error)` — opens connection with libSQL driver, executes CREATE TABLE/INDEX statements
- Returns `*sql.DB` for use by both repository implementations

### `libsql_heartbeat_repository.go`

Implements `HeartbeatRepository`:

| Method             | SQL                                                                                                                      |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------ |
| `Store`            | `INSERT INTO heartbeats (id, owner, version, charging, user_id, timestamp) VALUES (?, ?, ?, ?, ?, ?)`                    |
| `Index`            | `SELECT ... WHERE user_id = ? AND owner = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?` (optional `version LIKE ?` filter) |
| `Last`             | `SELECT ... WHERE user_id = ? AND owner = ? ORDER BY timestamp DESC LIMIT 1`                                             |
| `DeleteAllForUser` | `DELETE FROM heartbeats WHERE user_id = ?`                                                                               |

### `libsql_heartbeat_monitor_repository.go`

Implements `HeartbeatMonitorRepository`:

| Method              | SQL                                                                         |
| ------------------- | --------------------------------------------------------------------------- |
| `Store`             | INSERT all fields                                                           |
| `Load`              | SELECT by user_id + owner                                                   |
| `Exists`            | `SELECT COUNT(*) FROM ... WHERE user_id = ? AND id = ?` (returns count > 0) |
| `UpdateQueueID`     | UPDATE queue_id + updated_at WHERE id = ?                                   |
| `Delete`            | DELETE WHERE user_id = ? AND owner = ?                                      |
| `UpdatePhoneOnline` | UPDATE phone_online + updated_at WHERE id = ? AND user_id = ?               |
| `DeleteAllForUser`  | DELETE WHERE user_id = ?                                                    |

### Error Handling

- `sql.ErrNoRows` → wrap with `stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg)` to match GORM behavior expected by the service layer
- All other errors → wrap with `stacktrace.Propagate(err, msg)`

### Observability

Every method follows the existing tracing pattern:

```go
ctx, span := repository.tracer.Start(ctx)
defer span.End()
```

## DI Container Changes

```go
// New field on Container struct
tursoDB *sql.DB

// New method
func (container *Container) TursoDB() *sql.DB {
    if container.tursoDB != nil {
        return container.tursoDB
    }
    db, err := repositories.NewTursoDB(os.Getenv("TURSO_DATABASE_URL"), os.Getenv("TURSO_AUTH_TOKEN"))
    if err != nil {
        container.logger.Fatal(err)
    }
    container.tursoDB = db
    return container.tursoDB
}

// Modified methods
func (container *Container) HeartbeatRepository() repositories.HeartbeatRepository {
    if os.Getenv("HEARTBEAT_DB_BACKEND") == "turso" {
        return repositories.NewLibsqlHeartbeatRepository(container.Logger(), container.Tracer(), container.TursoDB())
    }
    return repositories.NewGormHeartbeatRepository(container.Logger(), container.Tracer(), container.DedicatedDB())
}

func (container *Container) HeartbeatMonitorRepository() repositories.HeartbeatMonitorRepository {
    if os.Getenv("HEARTBEAT_DB_BACKEND") == "turso" {
        return repositories.NewLibsqlHeartbeatMonitorRepository(container.Logger(), container.Tracer(), container.TursoDB())
    }
    return repositories.NewGormHeartbeatMonitorRepository(container.Logger(), container.Tracer(), container.DedicatedDB())
}
```

## Safety

- When `HEARTBEAT_DB_BACKEND != "turso"`, `TursoDB()` is never called — no Turso connection is opened
- The existing PostgreSQL path remains the default and is completely unaffected
- Both implementations satisfy the same interface — the service layer requires no changes
