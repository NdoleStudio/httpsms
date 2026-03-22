# Copilot Instructions for httpSMS

httpSMS is a service that turns an Android phone into an SMS gateway via an HTTP API. This is a monorepo with three components:

- **`api/`** — Go backend (Fiber, GORM, PostgreSQL)
- **`web/`** — Nuxt 2 frontend (Vue 2, Vuetify 2, TypeScript)
- **`android/`** — Native Android app (Kotlin)

## Build, Test, and Lint Commands

### API (Go)

```bash
cd api

# Development with hot-reload
air

# Build
go build -o ./tmp/main.exe .

# Run tests
go test ./...

# Run a single test
go test ./pkg/services/ -run TestMessageService

# Generate Swagger docs (required after changing API annotations)
swag init --requiredByDefault --parseDependency --parseInternal

# Pre-commit hooks run: go-fumpt, go-imports, go-lint, go-mod-tidy
```

### Web (Nuxt/Vue)

```bash
cd web

# Install dependencies
pnpm install

# Development server (port 3000)
pnpm dev

# Lint (eslint + stylelint + prettier)
pnpm lint

# Auto-fix lint issues
pnpm lintfix

# Run tests (Jest)
pnpm test

# Static site generation (production build)
pnpm run generate

# Regenerate TypeScript API models from Swagger
pnpm api:models
```

### Android (Kotlin)

```bash
cd android

# Build
./gradlew build

# Debug APK
./gradlew assembleDebug

# Release APK
./gradlew assembleRelease
```

### Docker (full stack)

```bash
# Start all services (PostgreSQL, Redis, API, Web)
docker compose up --build
# API at localhost:8000, Web at localhost:3000
```

## Architecture

### API — Layered Architecture with Event-Driven Processing

The API uses a **DI container** (`pkg/di/container.go`) that lazily initializes all services as singletons. The layered architecture flows as:

**Handlers → Services → Repositories → GORM/PostgreSQL**

- **Handlers** (`pkg/handlers/`) — Fiber HTTP handlers. Each has a `RegisterRoutes()` method and embeds a base `handler` struct with standardized response methods (`responseBadRequest`, `responseNotFound`, etc.).
- **Services** (`pkg/services/`) — Business logic. Orchestrate repositories and dispatch events.
- **Repositories** (`pkg/repositories/`) — Data access via GORM. Interfaces defined alongside GORM implementations (prefixed `gorm*`).
- **Validators** (`pkg/validators/`) — One validator per handler, return `url.Values` for field errors.
- **Entities** (`pkg/entities/`) — Domain models, auto-migrated by GORM.

**Event system**: Uses CloudEvents spec (`cloudevents/sdk-go`). Events defined in `pkg/events/` (31 event types). Listeners in `pkg/listeners/` process events either synchronously or via Google Cloud Tasks queue (emulator mode for local dev).

**Entry point**: `main.go` loads `.env` in local mode, creates the DI container, and starts Fiber on `APP_PORT`.

### Web — Nuxt 2 Static SPA

- **State management**: Single Vuex store (`store/index.ts`) — actions make API calls via Axios, mutations update state, getters expose computed values.
- **Components**: Use `vue-property-decorator` class syntax with `@Component`, `@Prop`, `@Watch` decorators.
- **API client**: Axios configured in `plugins/axios.ts` with Firebase bearer token auth and `x-api-key` header support.
- **API models**: TypeScript types in `models/` are auto-generated from the Swagger spec via `swagger-typescript-api`.
- **Auth**: Firebase Authentication (Email/Password, Google, GitHub) with `auth` and `guest` middleware for route guards.
- **Real-time**: Pusher.js for live message updates.

### Android — Task-Oriented, Event-Driven

- **No MVVM/Clean Architecture** — uses a flat package structure with Activities, Services, BroadcastReceivers, and WorkManager tasks.
- **FCM integration**: `MyFirebaseMessagingService` receives push notifications → schedules `SendSmsWorker` via WorkManager → fetches message from API → sends SMS.
- **Dual SIM support**: Independent settings per SIM via `Settings` singleton (SharedPreferences).
- **HTTP client**: OkHttp with `x-api-key` authentication against the API.
- **Encryption**: AES-256/CFB with SHA-256 key derivation (`Encrypter.kt`).

## Key Conventions

### API (Go)

- **Error handling**: Use `github.com/palantir/stacktrace` — wrap errors with `stacktrace.Propagate(err, "context")` or `stacktrace.PropagateWithCode()`. Never return bare errors.
- **Database queries**: Always use GORM query builder with context propagation (`repository.db.WithContext(ctx)`). No raw SQL.
- **Route registration**: Each handler defines `RegisterRoutes()` called from the DI container. Routes follow REST conventions under `/v1/`.
- **Middleware chain**: HTTP Logger → OpenTelemetry → CORS → Request Logger → Bearer Auth → API Key Auth.
- **Observability**: All layers are instrumented with OpenTelemetry (Fiber, GORM, Redis). Pass `logger` and `tracer` to constructors.
- **Code formatting**: `go-fumpt` (not `gofmt`), enforced via pre-commit hooks.

### Web (Vue/TypeScript)

- **Formatting**: No semicolons, single quotes, 2-space indentation (Prettier + ESLint).
- **Component style**: Class-based with `vue-property-decorator`, not Options API (though some pages use `Vue.extend()`).
- **Store pattern**: Actions handle async API calls and commit mutations. Access store from components via `this.$store`.

### Android (Kotlin)

- **API calls**: Use `HttpSmsApiService` singleton (static `create()` factory). OkHttp client with `x-api-key` header.
- **Background work**: Use WorkManager for tasks that must survive process death. Direct `Thread { }` for lightweight background ops.
- **State**: `Settings` object (SharedPreferences singleton) for all persistent state.
- **Phone number formatting**: Use `libphonenumber` for E.164 format validation.
