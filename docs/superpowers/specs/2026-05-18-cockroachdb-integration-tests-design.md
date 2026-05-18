# Replace PostgreSQL with CockroachDB in Integration Tests

**Date:** 2026-05-18
**Status:** Approved
**Goal:** Align the integration test database with production (CockroachDB).

## Context

Production httpSMS uses CockroachDB. The integration test suite (`tests/docker-compose.yml`) currently uses `postgres:alpine`. This mismatch means tests don't exercise CockroachDB-specific behavior (e.g., serializable isolation by default, distributed SQL quirks). The API's GORM `postgres` driver is already wire-compatible with CockroachDB, so no application code changes are needed.

## Design

### Docker Compose Changes (`tests/docker-compose.yml`)

1. **Replace `postgres` service with `cockroachdb`:**

   - Image: `cockroachdb/cockroach:latest`
   - Command: `start-single-node --insecure --store=type=mem,size=256MiB`
   - Ports: `26257:26257` (SQL), `8081:8080` (admin UI — remapped to avoid wiremock conflict)
   - Healthcheck: `curl -f http://localhost:8080/health?ready=1`

2. **Add `cockroachdb-init` service** (runs once after cockroachdb is healthy):

   - Uses same CockroachDB image
   - Creates the `httpsms` database: `cockroach sql --insecure --host=cockroachdb --execute="CREATE DATABASE IF NOT EXISTS httpsms;"`

3. **Update `seed` service:**

   - Image: `cockroachdb/cockroach:latest` (instead of `postgres:alpine`)
   - Entrypoint: `cockroach sql --insecure --host=cockroachdb --database=httpsms --file=/seed.sql`
   - Depends on `api` (healthy) so GORM migrations run first

4. **Update `api` service:**
   - `depends_on`: replace `postgres` with `cockroachdb-init`

### Environment Changes (`tests/.env.test`)

```
DATABASE_URL=postgresql://root@cockroachdb:26257/httpsms?sslmode=disable
DATABASE_URL_DEDICATED=postgresql://root@cockroachdb:26257/httpsms?sslmode=disable
```

### Seed SQL (`tests/seed.sql`)

No changes required. `INSERT ... ON CONFLICT ... DO NOTHING` and `NOW()` are supported by CockroachDB.

### What Does NOT Change

- API application code (GORM postgres driver works with CockroachDB)
- Integration test Go code
- Redis, MongoDB, Wiremock services
- API Dockerfile

## Key Decisions

| Decision                            | Rationale                                                                                     |
| ----------------------------------- | --------------------------------------------------------------------------------------------- |
| Single-node insecure mode           | Simplest for tests; no TLS cert management                                                    |
| In-memory store (`type=mem`)        | Faster startup, no persistent volume needed for tests                                         |
| Separate `cockroachdb-init` service | CockroachDB doesn't support `POSTGRES_DB`-style env vars; database must be created explicitly |
| Remap admin UI to 8081              | Avoids port conflict with wiremock on 8080                                                    |
| Use `cockroach sql` for seeding     | Native client; avoids pulling a separate postgres image                                       |

## Risks

- **GORM migration compatibility:** CockroachDB may handle some DDL differently (e.g., `ALTER TABLE` constraints). The existing `DATABASE_MIGRATION_CONSTRAINT_FIX` env var handles known issues — we'll enable it in `.env.test`.
- **Startup time:** CockroachDB takes slightly longer to start than PostgreSQL (~5-10s). The healthcheck handles this.
