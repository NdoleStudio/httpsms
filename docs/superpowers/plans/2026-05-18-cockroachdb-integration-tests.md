# CockroachDB Integration Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace PostgreSQL with CockroachDB in the integration test Docker Compose so the test environment matches production.

**Architecture:** Swap the `postgres:alpine` container for `cockroachdb/cockroach:latest` in single-node insecure mode with in-memory storage. Add a one-shot init container to create the database (CockroachDB has no `POSTGRES_DB` env var equivalent). Update connection strings and seed service.

**Tech Stack:** Docker Compose, CockroachDB (single-node), GORM (postgres driver — wire-compatible)

---

## File Map

| File                       | Action | Responsibility                                                            |
| -------------------------- | ------ | ------------------------------------------------------------------------- |
| `tests/docker-compose.yml` | Modify | Replace postgres service, add cockroachdb + init, update seed and api     |
| `tests/.env.test`          | Modify | Update DATABASE_URL connection strings, add migration fix flag            |
| `tests/README.md`          | Modify | Update architecture diagram and references from PostgreSQL to CockroachDB |

---

### Task 1: Replace PostgreSQL with CockroachDB in Docker Compose

**Files:**

- Modify: `tests/docker-compose.yml`

- [ ] **Step 1: Replace the `postgres` service with `cockroachdb`**

Replace lines 1–15 of `tests/docker-compose.yml` (the `postgres` service) with:

```yaml
services:
  cockroachdb:
    image: cockroachdb/cockroach:latest
    command: start-single-node --insecure --store=type=mem,size=256MiB
    ports:
      - "26257:26257"
      - "8081:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health?ready=1"]
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 10s
```

- [ ] **Step 2: Add `cockroachdb-init` service after `cockroachdb`**

Add this service immediately after the `cockroachdb` service:

```yaml
cockroachdb-init:
  image: cockroachdb/cockroach:latest
  depends_on:
    cockroachdb:
      condition: service_healthy
  entrypoint:
    [
      "cockroach",
      "sql",
      "--insecure",
      "--host=cockroachdb",
      "--execute=CREATE DATABASE IF NOT EXISTS httpsms;",
    ]
  restart: "no"
```

- [ ] **Step 3: Update `api` service `depends_on`**

Replace the `depends_on` block of the `api` service. Change:

```yaml
depends_on:
  postgres:
    condition: service_healthy
  redis:
    condition: service_healthy
  wiremock:
    condition: service_healthy
  mongodb:
    condition: service_healthy
```

To:

```yaml
depends_on:
  cockroachdb-init:
    condition: service_completed_successfully
  redis:
    condition: service_healthy
  wiremock:
    condition: service_healthy
  mongodb:
    condition: service_healthy
```

- [ ] **Step 4: Update `seed` service**

Replace the entire `seed` service. Change:

```yaml
seed:
  image: postgres:alpine
  depends_on:
    api:
      condition: service_healthy
  environment:
    PGPASSWORD: dbpassword
  volumes:
    - ./seed.sql:/seed.sql:ro
  entrypoint:
    [
      "psql",
      "-h",
      "postgres",
      "-U",
      "dbusername",
      "-d",
      "httpsms",
      "-f",
      "/seed.sql",
    ]
  restart: "no"
```

To:

```yaml
seed:
  image: cockroachdb/cockroach:latest
  depends_on:
    api:
      condition: service_healthy
  volumes:
    - ./seed.sql:/seed.sql:ro
  entrypoint:
    [
      "cockroach",
      "sql",
      "--insecure",
      "--host=cockroachdb",
      "--database=httpsms",
      "--file=/seed.sql",
    ]
  restart: "no"
```

- [ ] **Step 5: Verify the final `docker-compose.yml` structure**

The final file should contain these services in order:

1. `cockroachdb` — database server
2. `cockroachdb-init` — creates the database (one-shot)
3. `redis` — unchanged
4. `mongodb` — unchanged
5. `wiremock` — unchanged
6. `api` — depends on `cockroachdb-init` instead of `postgres`
7. `seed` — uses `cockroach sql` instead of `psql`

- [ ] **Step 6: Commit**

```bash
git add tests/docker-compose.yml
git commit -m "test: replace postgres with cockroachdb in integration tests

Use cockroachdb/cockroach:latest in single-node insecure mode with
in-memory storage. Add cockroachdb-init service to create the database
and update the seed service to use cockroach sql CLI."
```

---

### Task 2: Update Environment Variables

**Files:**

- Modify: `tests/.env.test`

- [ ] **Step 1: Update DATABASE_URL**

Change line 11:

```
DATABASE_URL=postgresql://dbusername:dbpassword@postgres:5432/httpsms
```

To:

```
DATABASE_URL=postgresql://root@cockroachdb:26257/httpsms?sslmode=disable
```

- [ ] **Step 2: Update DATABASE_URL_DEDICATED**

Change line 12:

```
DATABASE_URL_DEDICATED=postgresql://dbusername:dbpassword@postgres:5432/httpsms
```

To:

```
DATABASE_URL_DEDICATED=postgresql://root@cockroachdb:26257/httpsms?sslmode=disable
```

- [ ] **Step 3: Add DATABASE_MIGRATION_CONSTRAINT_FIX**

Add the following line after `DATABASE_URL_DEDICATED`:

```
DATABASE_MIGRATION_CONSTRAINT_FIX=1
```

This enables the GORM migration workaround for CockroachDB constraint handling (already used in production, see `api/pkg/di/container.go:369-376`).

- [ ] **Step 4: Commit**

```bash
git add tests/.env.test
git commit -m "test: update env vars for cockroachdb connection

Point DATABASE_URL at cockroachdb:26257 with root user (insecure mode).
Enable DATABASE_MIGRATION_CONSTRAINT_FIX for CockroachDB compatibility."
```

---

### Task 3: Update README Documentation

**Files:**

- Modify: `tests/README.md`

- [ ] **Step 1: Update architecture diagram**

Replace the ASCII diagram (lines 7–25) — change `PostgreSQL` to `CockroachDB` and port `5435` to `26257`:

```
┌──────────────┐     HTTP      ┌──────────────┐
│  Test Runner │─────────────▶│   API (Go)   │
│   (Go test)  │              │  Port 8000   │
└──────────────┘              └──────┬───────┘
                                     │
                          FCM Push   │   Events
                          (HTTP)     │   (HTTP)
                                     ▼
                              ┌──────────────┐
                              │   Emulator   │
                              │  (Fiber v3)  │
                              │  Port 9090   │
                              └──────────────┘
                                     │
                              ┌──────┴───────┐
                              │ CockroachDB  │   │    Redis    │
                              │  Port 26257  │   │  Port 6379  │
                              └──────────────┘   └─────────────┘
```

- [ ] **Step 2: Update Components table**

Change the row:

```markdown
| **PostgreSQL** | Database for the API |
```

To:

```markdown
| **CockroachDB** | Database for the API (single-node, insecure mode) |
```

And change:

```markdown
| **Seed** | One-shot container that seeds test data into PostgreSQL |
```

To:

```markdown
| **Seed** | One-shot container that seeds test data into CockroachDB |
```

- [ ] **Step 3: Update "Start the Stack" section**

In the "3. Start the Stack" section (line 85-89), the command stays the same (`docker compose up -d --build --wait`) but update the description:

Change:

```
This starts PostgreSQL, Redis, the API, and the emulator. The `--wait` flag blocks until all health checks pass.
```

To:

```
This starts CockroachDB, Redis, the API, and the emulator. The `--wait` flag blocks until all health checks pass.
```

- [ ] **Step 4: Update "Wait for Seeding" description**

Change line 98:

```
The seed container inserts test users, phones, and API keys into PostgreSQL after the API has run its GORM migrations.
```

To:

```
The seed container inserts test users, phones, and API keys into CockroachDB after the API has run its GORM migrations.
```

- [ ] **Step 5: Update Troubleshooting section**

In "API fails to start" common issues (line 185), change:

```
- PostgreSQL not ready (increase `start_period` in healthcheck)
```

To:

```
- CockroachDB not ready (increase `start_period` in healthcheck)
```

- [ ] **Step 6: Commit**

```bash
git add tests/README.md
git commit -m "docs: update integration test README for cockroachdb

Replace all PostgreSQL references with CockroachDB in the architecture
diagram, components table, and troubleshooting section."
```

---

### Task 4: Validate the Stack

- [ ] **Step 1: Generate Firebase credentials (if not already present)**

```bash
cd tests && bash generate-firebase-credentials.sh
```

- [ ] **Step 2: Set the environment variable**

```bash
export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json)
```

- [ ] **Step 3: Start the stack**

```bash
docker compose up -d --build --wait
```

Expected: All services start. CockroachDB reports healthy. cockroachdb-init exits with code 0. API starts and passes healthcheck.

- [ ] **Step 4: Verify seed ran successfully**

```bash
docker compose wait seed && docker compose logs seed
```

Expected: Output shows `INSERT` statements succeeded (no errors).

- [ ] **Step 5: Run integration tests**

```bash
go test -v -timeout 120s ./...
```

Expected: All tests pass (same tests as before — no test code changed).

- [ ] **Step 6: Tear down**

```bash
docker compose down -v
```

- [ ] **Step 7: Final commit (if any fixes were needed)**

If any adjustments were required during validation, commit them:

```bash
git add -A
git commit -m "test: fix integration test issues found during validation"
```
