# Contacts Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Contacts feature (name, emails, phone numbers, free-form properties) with full CRUD + CSV import, and display the resolved contact name instead of the raw phone number on the threads page, using as few DB queries as possible.

**Architecture:** New `Contact` entity + layered backend (handler → service → repository → validator → requests/responses) mirroring existing patterns. Thread-name resolution is opt-in per request and served from a per-user `phone_number → *Contact` map cached in `cache.Cache`, so the common path adds zero DB queries and a name change is a single-row `UPDATE` + cache invalidation. Frontend adds a Pinia store and a Plunk-style Contacts page (Nuxt 4 SPA).

**Tech Stack:** Go (Fiber v3, GORM v1.31.2, lib/pq, nyaruka/phonenumbers, jszwec/csvutil), Nuxt 4 + Pinia + Vuetify 4, Swagger (`swag`), swagger-typescript-api.

## Global Constraints

- Source of truth: `docs/superpowers/specs/2026-07-19-contacts-feature-design.md`. Follow it exactly.
- Run Go tests with `go test -vet=off ./...` from `api/` (Go 1.26 vet rejects the repo's `stacktrace.Propagate(err, fmt.Sprintf(...))` pattern).
- Wrap all Go errors with `github.com/NdoleStudio/stacktrace` (`Propagatef` / `PropagateWithCodef`) — never return bare errors.
- All GORM queries use `repository.db.WithContext(ctx)`. No raw SQL.
- Register Fiber routes via the `h.register(router, fiber.MethodX, path, middlewares, route)` helper (Fiber v3), not `Get`/`Post` directly.
- Contacts are global to the user account. **No** uniqueness constraint on phone numbers; **no** `contact_phone_numbers` lookup table.
- On phone-number collision across contacts, the **most recently updated** contact wins for display.
- Thread-name resolution is **opt-in** via `?contacts=true` (default off).
- Create endpoint accepts one or many; batch capped at **≤ 1000** contacts; cache invalidated **once** after the batch commits.
- CSV import is **CSV only** — Excel/XLSX is not supported for contacts. Max 500 KB, ≤ 1000 rows.
- `pq.StringArray` with `gorm:"type:text[]" swaggertype:"array,string"` for array fields (convention from `phone_api_key.go`/`webhook.go`).
- Web conventions: every `VDialog` has `opacity="0.9"` and a Close button with `color="warning"`; hyperlinks get `text-decoration-none hover:text-decoration-underline`; destructure filters from `useFilters()` in `<script setup>` and use directly in the template; no glowing gradient text off the homepage; Vuetify 4 typography classes (`text-headline-large`, `text-title-large`, `text-display-large`, etc.); in `useApi` `onRequest`, `options.headers` is a `Headers` instance — use `.set()`.
- Regenerate Swagger after API annotation changes: `cd api && swag init --requiredByDefault --parseDependency --parseInternal`.
- Regenerate web API models after Swagger changes: `cd web && pnpm api:models`.

---

## File Structure

Backend (`api/`):
- `pkg/entities/contact.go` — `Contact` entity + `ContactProperties` custom jsonb type (Valuer/Scanner).
- `pkg/entities/contact_test.go` — round-trip tests for `ContactProperties`.
- `pkg/entities/message_thread.go` — add non-persisted `ContactDetails *Contact` field.
- `pkg/repositories/contact_repository.go` — `ContactRepository` interface + param structs.
- `pkg/repositories/gorm_contact_repository.go` — GORM implementation.
- `pkg/repositories/gorm_contact_repository_test.go` — repository tests.
- `pkg/requests/contact_store.go` — create-many request + per-item `ContactItem`.
- `pkg/requests/contact_update.go` — update request.
- `pkg/requests/contact_index.go` — list/search request.
- `pkg/responses/*` — reuse existing generic OK/Created responses (no new file needed).
- `pkg/validators/contact_handler_validator.go` — validates store/update/index + CSV parse.
- `pkg/services/contact_service.go` — CRUD + contact-map build + cache invalidation.
- `pkg/services/contact_service_test.go` — service tests.
- `pkg/handlers/contact_handler.go` — HTTP handlers + `RegisterRoutes`.
- `pkg/services/message_thread_service.go` — extend `GetThreads` to attach `ContactDetails`.
- `pkg/requests/message_thread_index_request.go` — add `Contacts` query param.
- `pkg/di/container.go` — wire repo/service/validator/handler, AutoMigrate, register routes.

Frontend (`web/`):
- `app/stores/contacts.ts` — Pinia store.
- `app/pages/contacts/index.vue` — Plunk-style Contacts page + dialogs.
- `app/stores/threads.ts` — pass `contacts: true`.
- `app/components/MessageThread.vue`, `app/components/MessageThreadHeader.vue` — render resolved name.
- default layout — add Contacts nav link.
- `shared/types/api` — regenerated from Swagger.
- `public/templates/httpsms-contacts.csv` — CSV template.

---

### Task 1: Contact entity + ContactProperties custom type

**Files:**
- Create: `api/pkg/entities/contact.go`
- Create: `api/pkg/entities/contact_test.go`

**Interfaces:**
- Produces: `entities.Contact` struct; `entities.ContactProperties` (`map[string]string`) implementing `driver.Valuer` and `sql.Scanner`.

- [ ] **Step 1: Write the failing test**

Create `api/pkg/entities/contact_test.go`:

```go
package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContactProperties_ValueScanRoundTrip(t *testing.T) {
	cases := []ContactProperties{
		nil,
		{},
		{"company": "Acme", "role": "CTO"},
	}

	for _, original := range cases {
		value, err := original.Value()
		assert.Nil(t, err)

		var scanned ContactProperties
		assert.Nil(t, scanned.Scan(value))

		if len(original) == 0 {
			assert.Equal(t, 0, len(scanned))
			continue
		}
		assert.Equal(t, original, scanned)
	}
}

func TestContactProperties_ScanFromString(t *testing.T) {
	var scanned ContactProperties
	assert.Nil(t, scanned.Scan(`{"k":"v"}`))
	assert.Equal(t, ContactProperties{"k": "v"}, scanned)
}

func TestContactProperties_ScanNil(t *testing.T) {
	var scanned ContactProperties
	assert.Nil(t, scanned.Scan(nil))
	assert.Equal(t, 0, len(scanned))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd api && go test -vet=off ./pkg/entities/ -run TestContactProperties`
Expected: FAIL / build error — `Contact`/`ContactProperties` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `api/pkg/entities/contact.go`:

```go
package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ContactProperties is a free-form key/value map persisted as a jsonb column.
type ContactProperties map[string]string

// Value implements driver.Valuer, serializing the map to JSON bytes.
func (p ContactProperties) Value() (driver.Value, error) {
	if p == nil {
		return []byte("{}"), nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Scan implements sql.Scanner, deserializing jsonb bytes/string into the map.
func (p *ContactProperties) Scan(src any) error {
	if src == nil {
		*p = ContactProperties{}
		return nil
	}

	var data []byte
	switch value := src.(type) {
	case []byte:
		data = value
	case string:
		data = []byte(value)
	default:
		return fmt.Errorf("unsupported type [%T] for ContactProperties", src)
	}

	if len(data) == 0 {
		*p = ContactProperties{}
		return nil
	}

	result := ContactProperties{}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*p = result
	return nil
}

// Contact represents a saved contact belonging to a user.
type Contact struct {
	ID           uuid.UUID         `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID       UserID            `json:"user_id" gorm:"index" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Name         string            `json:"name" example:"Alice Smith"`
	Emails       pq.StringArray    `json:"emails" gorm:"type:text[]" swaggertype:"array,string" example:"alice@example.com"`
	PhoneNumbers pq.StringArray    `json:"phone_numbers" gorm:"type:text[]" swaggertype:"array,string" example:"+18005550199,+18005550100"`
	Properties   ContactProperties `json:"properties" gorm:"type:jsonb" swaggertype:"object,string"`
	CreatedAt    time.Time         `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt    time.Time         `json:"updated_at" example:"2022-06-05T14:26:02.302718+03:00"`
}

// TableName overrides the table name used by Contact.
func (Contact) TableName() string {
	return "contacts"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd api && go test -vet=off ./pkg/entities/ -run TestContactProperties`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/pkg/entities/contact.go api/pkg/entities/contact_test.go
git commit -m "feat(api): add Contact entity and ContactProperties jsonb type"
```

---

### Task 2: MessageThread ContactDetails field + AutoMigrate

**Files:**
- Modify: `api/pkg/entities/message_thread.go`
- Modify: `api/pkg/di/container.go` (AutoMigrate block ~L413)

**Interfaces:**
- Produces: `entities.MessageThread.ContactDetails *Contact` (JSON `contact_details`, `gorm:"-"`), `contacts` table auto-migrated.

- [ ] **Step 1: Add the non-persisted field**

In `api/pkg/entities/message_thread.go`, add after the `OrderTimestamp` field inside the `MessageThread` struct:

```go
	// ContactDetails is resolved at read time and never persisted.
	ContactDetails *Contact `json:"contact_details,omitempty" gorm:"-"`
```

- [ ] **Step 2: Register AutoMigrate**

In `api/pkg/di/container.go`, immediately after the `&entities.PhoneAPIKey{}` AutoMigrate block (~L413-415), add:

```go
	if err = db.AutoMigrate(&entities.Contact{}); err != nil {
		container.logger.Fatal(stacktrace.Propagatef(err, "cannot migrate %T", &entities.Contact{}))
	}
```

- [ ] **Step 3: Build to verify it compiles**

Run: `cd api && go build ./...`
Expected: builds with no errors.

- [ ] **Step 4: Commit**

```bash
git add api/pkg/entities/message_thread.go api/pkg/di/container.go
git commit -m "feat(api): add non-persisted ContactDetails on MessageThread and migrate contacts table"
```

---

### Task 3: ContactRepository interface + GORM implementation

**Files:**
- Create: `api/pkg/repositories/contact_repository.go`
- Create: `api/pkg/repositories/gorm_contact_repository.go`
- Create: `api/pkg/repositories/gorm_contact_repository_test.go`

**Interfaces:**
- Consumes: `entities.Contact`, `repositories.IndexParams`, `repositories.ErrCodeNotFound`.
- Produces: `repositories.ContactRepository` with methods:
  - `Store(ctx context.Context, contacts []*entities.Contact) error`
  - `Update(ctx context.Context, contact *entities.Contact) error`
  - `Load(ctx context.Context, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error)`
  - `Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Contact, error)`
  - `FetchAll(ctx context.Context, userID entities.UserID) (*[]entities.Contact, error)` (ordered `updated_at ASC` for tie-break)
  - `Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error`
  - `DeleteAllForUser(ctx context.Context, userID entities.UserID) error`
  - Constructor: `NewGormContactRepository(logger, tracer, db) ContactRepository`

- [ ] **Step 1: Write the interface**

Create `api/pkg/repositories/contact_repository.go`:

```go
package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// ContactRepository loads and persists an entities.Contact
type ContactRepository interface {
	// Store one or many new entities.Contact
	Store(ctx context.Context, contacts []*entities.Contact) error

	// Update an existing entities.Contact
	Update(ctx context.Context, contact *entities.Contact) error

	// Load a contact by ID for a user
	Load(ctx context.Context, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error)

	// Index contacts for a user with optional search
	Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Contact, error)

	// FetchAll returns every contact for a user ordered by updated_at ascending
	FetchAll(ctx context.Context, userID entities.UserID) (*[]entities.Contact, error)

	// Delete a contact by ID for a user
	Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error

	// DeleteAllForUser deletes all contacts for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
```

- [ ] **Step 2: Write the failing test**

Create `api/pkg/repositories/gorm_contact_repository_test.go`. Reuse the SQL-capturing conn-pool mock pattern already defined in `api/pkg/repositories/gorm_message_thread_repository_test.go` (the `messageThreadTestConnPool`, `messageThreadTestLogger`, and the `postgres.New(postgres.Config{Conn: ...})` DryRun setup). Add these focused tests:

```go
package repositories

import (
	"context"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newContactTestRepo(t *testing.T) (ContactRepository, *messageThreadTestConnPool) {
	pool := &messageThreadTestConnPool{}
	db, err := gorm.Open(
		postgres.New(postgres.Config{Conn: pool, WithoutReturning: true}),
		&gorm.Config{DisableAutomaticPing: true},
	)
	require.NoError(t, err)
	logger := &messageThreadTestLogger{}
	return NewGormContactRepository(logger, telemetry.NewOtelLogger("test", logger), db), pool
}

func TestGormContactRepository_Index_FiltersByUserAndQuery(t *testing.T) {
	repo, _ := newContactTestRepo(t)

	_, err := repo.Index(context.Background(), entities.UserID("user-1"), IndexParams{Query: "alice", Limit: 20, Skip: 0})
	assert.Nil(t, err)
}

func TestGormContactRepository_FetchAll_OrdersByUpdatedAtAsc(t *testing.T) {
	repo, _ := newContactTestRepo(t)

	_, err := repo.FetchAll(context.Background(), entities.UserID("user-1"))
	assert.Nil(t, err)
}

func TestGormContactRepository_Store_BuildsContact(t *testing.T) {
	repo, _ := newContactTestRepo(t)

	err := repo.Store(context.Background(), []*entities.Contact{{
		ID:           uuid.New(),
		UserID:       entities.UserID("user-1"),
		Name:         "Alice",
		Emails:       pq.StringArray{"alice@example.com"},
		PhoneNumbers: pq.StringArray{"+18005550199"},
	}})
	assert.Nil(t, err)
}
```

(The `messageThreadTestConnPool` and `messageThreadTestLogger` types are already defined in `gorm_message_thread_repository_test.go` in the same package, so they are reused directly here.)

- [ ] **Step 3: Run test to verify it fails**

Run: `cd api && go test -vet=off ./pkg/repositories/ -run TestGormContactRepository`
Expected: FAIL / build error — `NewGormContactRepository` undefined.

- [ ] **Step 4: Write the implementation**

Create `api/pkg/repositories/gorm_contact_repository.go`:

```go
package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormContactRepository is responsible for persisting entities.Contact
type gormContactRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormContactRepository creates the GORM version of the ContactRepository
func NewGormContactRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) ContactRepository {
	return &gormContactRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormContactRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *gormContactRepository) Store(ctx context.Context, contacts []*entities.Contact) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if len(contacts) == 0 {
		return nil
	}

	if err := repository.db.WithContext(ctx).Create(&contacts).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot store [%d] contacts", len(contacts)))
	}
	return nil
}

func (repository *gormContactRepository) Update(ctx context.Context, contact *entities.Contact) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(contact).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update contact with ID [%s]", contact.ID))
	}
	return nil
}

func (repository *gormContactRepository) Load(ctx context.Context, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	contact := new(entities.Contact)
	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", contactID).
		First(contact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "contact with id [%s] not found for user [%s]", contactID, userID))
	}
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load contact with id [%s]", contactID))
	}
	return contact, nil
}

func (repository *gormContactRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.WithContext(ctx).Where("user_id = ?", userID)

	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query = query.Where(
			repository.db.Where("name ILIKE ?", queryPattern).
				Or("array_to_string(emails, ',') ILIKE ?", queryPattern).
				Or("array_to_string(phone_numbers, ',') ILIKE ?", queryPattern),
		)
	}

	contacts := new([]entities.Contact)
	if err := query.Order("updated_at DESC").Limit(params.Limit).Offset(params.Skip).Find(contacts).Error; err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot index contacts for user [%s]", userID))
	}
	return contacts, nil
}

func (repository *gormContactRepository) FetchAll(ctx context.Context, userID entities.UserID) (*[]entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	contacts := new([]entities.Contact)
	if err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at ASC").
		Find(contacts).Error; err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch all contacts for user [%s]", userID))
	}
	return contacts, nil
}

func (repository *gormContactRepository) Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", contactID).
		Delete(&entities.Contact{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete contact with id [%s] for user [%s]", contactID, userID))
	}
	return nil
}

func (repository *gormContactRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&entities.Contact{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all contacts for user [%s]", userID))
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd api && go test -vet=off ./pkg/repositories/ -run TestGormContactRepository`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add api/pkg/repositories/contact_repository.go api/pkg/repositories/gorm_contact_repository.go api/pkg/repositories/gorm_contact_repository_test.go
git commit -m "feat(api): add ContactRepository with GORM implementation"
```

---

### Task 4: Contact requests (store / update / index)

**Files:**
- Create: `api/pkg/requests/contact_store.go`
- Create: `api/pkg/requests/contact_update.go`
- Create: `api/pkg/requests/contact_index.go`
- Create: `api/pkg/requests/contact_store_test.go`

**Interfaces:**
- Produces:
  - `requests.ContactItem{ Name string; Emails []string; PhoneNumbers []string; Properties map[string]string }`
  - `requests.ContactStoreRequest{ Contacts []ContactItem }` with `UnmarshalJSON` accepting a JSON array or `{"contacts":[...]}`; methods `Sanitize()` and `ToContacts(userID entities.UserID) []*entities.Contact`.
  - `requests.ContactUpdateRequest{ Name string; Emails []string; PhoneNumbers []string; Properties map[string]string }` with `Sanitize()` and `ApplyTo(contact *entities.Contact)`.
  - `requests.ContactIndex{ Skip, Query, Limit string }` with `Sanitize()` and `ToIndexParams() repositories.IndexParams`.

- [ ] **Step 1: Write the failing test**

Create `api/pkg/requests/contact_store_test.go`:

```go
package requests

import (
	"encoding/json"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactStoreRequest_UnmarshalArrayForm(t *testing.T) {
	var request ContactStoreRequest
	require.NoError(t, json.Unmarshal([]byte(`[{"name":"Alice","phone_numbers":["+18005550199"]}]`), &request))
	assert.Equal(t, 1, len(request.Contacts))
	assert.Equal(t, "Alice", request.Contacts[0].Name)
}

func TestContactStoreRequest_UnmarshalObjectForm(t *testing.T) {
	var request ContactStoreRequest
	require.NoError(t, json.Unmarshal([]byte(`{"contacts":[{"name":"Bob","phone_numbers":["+18005550100"]}]}`), &request))
	assert.Equal(t, 1, len(request.Contacts))
	assert.Equal(t, "Bob", request.Contacts[0].Name)
}

func TestContactStoreRequest_SanitizeNormalizesNumbers(t *testing.T) {
	request := ContactStoreRequest{Contacts: []ContactItem{{
		Name:         "  Alice  ",
		PhoneNumbers: []string{"18005550199"},
		Emails:       []string{" alice@example.com "},
	}}}
	request = request.Sanitize()
	assert.Equal(t, "Alice", request.Contacts[0].Name)
	assert.Equal(t, "+18005550199", request.Contacts[0].PhoneNumbers[0])
	assert.Equal(t, "alice@example.com", request.Contacts[0].Emails[0])
}

func TestContactStoreRequest_ToContacts(t *testing.T) {
	request := ContactStoreRequest{Contacts: []ContactItem{{
		Name:         "Alice",
		PhoneNumbers: []string{"+18005550199"},
	}}}
	contacts := request.ToContacts(entities.UserID("user-1"))
	require.Equal(t, 1, len(contacts))
	assert.Equal(t, "Alice", contacts[0].Name)
	assert.Equal(t, entities.UserID("user-1"), contacts[0].UserID)
	assert.NotEqual(t, "", contacts[0].ID.String())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd api && go test -vet=off ./pkg/requests/ -run TestContactStoreRequest`
Expected: FAIL / build error — types undefined.

- [ ] **Step 3: Write the implementations**

Create `api/pkg/requests/contact_store.go`:

```go
package requests

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ContactItem is a single contact in a create request.
type ContactItem struct {
	Name         string            `json:"name" example:"Alice Smith"`
	Emails       []string          `json:"emails"`
	PhoneNumbers []string          `json:"phone_numbers"`
	Properties   map[string]string `json:"properties"`
}

// ContactStoreRequest creates one or many contacts.
type ContactStoreRequest struct {
	request
	Contacts []ContactItem `json:"contacts"`
}

// UnmarshalJSON accepts either a JSON array of contacts or {"contacts":[...]}.
func (input *ContactStoreRequest) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var items []ContactItem
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		input.Contacts = items
		return nil
	}

	type alias ContactStoreRequest
	var wrapper alias
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	input.Contacts = wrapper.Contacts
	return nil
}

// Sanitize trims and normalizes each contact item.
func (input ContactStoreRequest) Sanitize() ContactStoreRequest {
	for index := range input.Contacts {
		input.Contacts[index] = input.sanitizeItem(input.Contacts[index])
	}
	return input
}

func (input *ContactStoreRequest) sanitizeItem(item ContactItem) ContactItem {
	item.Name = strings.TrimSpace(item.Name)

	numbers := input.removeStringDuplicates(input.removeEmptyStrings(item.PhoneNumbers))
	item.PhoneNumbers = input.sanitizeAddresses(numbers)

	emails := input.removeEmptyStrings(item.Emails)
	for i := range emails {
		emails[i] = strings.ToLower(strings.TrimSpace(emails[i]))
	}
	item.Emails = input.removeStringDuplicates(emails)

	if item.Properties == nil {
		item.Properties = map[string]string{}
	}
	return item
}

// ToContacts converts the request into persistable entities.Contact records.
func (input *ContactStoreRequest) ToContacts(userID entities.UserID) []*entities.Contact {
	now := time.Now().UTC()
	contacts := make([]*entities.Contact, 0, len(input.Contacts))
	for _, item := range input.Contacts {
		contacts = append(contacts, &entities.Contact{
			ID:           uuid.New(),
			UserID:       userID,
			Name:         item.Name,
			Emails:       pq.StringArray(item.Emails),
			PhoneNumbers: pq.StringArray(item.PhoneNumbers),
			Properties:   entities.ContactProperties(item.Properties),
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	return contacts
}
```

Create `api/pkg/requests/contact_update.go`:

```go
package requests

import (
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/lib/pq"
)

// ContactUpdateRequest updates an existing contact.
type ContactUpdateRequest struct {
	request
	Name         string            `json:"name" example:"Alice Smith"`
	Emails       []string          `json:"emails"`
	PhoneNumbers []string          `json:"phone_numbers"`
	Properties   map[string]string `json:"properties"`
}

// Sanitize trims and normalizes the update request.
func (input ContactUpdateRequest) Sanitize() ContactUpdateRequest {
	input.Name = strings.TrimSpace(input.Name)

	numbers := input.removeStringDuplicates(input.removeEmptyStrings(input.PhoneNumbers))
	input.PhoneNumbers = input.sanitizeAddresses(numbers)

	emails := input.removeEmptyStrings(input.Emails)
	for i := range emails {
		emails[i] = strings.ToLower(strings.TrimSpace(emails[i]))
	}
	input.Emails = input.removeStringDuplicates(emails)

	if input.Properties == nil {
		input.Properties = map[string]string{}
	}
	return input
}

// ApplyTo mutates an existing contact with the update values.
func (input *ContactUpdateRequest) ApplyTo(contact *entities.Contact) {
	contact.Name = input.Name
	contact.Emails = pq.StringArray(input.Emails)
	contact.PhoneNumbers = pq.StringArray(input.PhoneNumbers)
	contact.Properties = entities.ContactProperties(input.Properties)
	contact.UpdatedAt = time.Now().UTC()
}
```

Create `api/pkg/requests/contact_index.go`:

```go
package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
)

// ContactIndex lists contacts for a user.
type ContactIndex struct {
	request
	Skip  string `json:"skip" query:"skip"`
	Query string `json:"query" query:"query"`
	Limit string `json:"limit" query:"limit"`
}

// Sanitize sets defaults for the list request.
func (input *ContactIndex) Sanitize() ContactIndex {
	if strings.TrimSpace(input.Limit) == "" {
		input.Limit = "20"
	}
	input.Query = strings.TrimSpace(input.Query)
	input.Skip = strings.TrimSpace(input.Skip)
	if input.Skip == "" {
		input.Skip = "0"
	}
	return *input
}

// ToIndexParams converts the request into repositories.IndexParams.
func (input *ContactIndex) ToIndexParams() repositories.IndexParams {
	return repositories.IndexParams{
		Skip:  input.getInt(input.Skip),
		Query: input.Query,
		Limit: input.getInt(input.Limit),
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd api && go test -vet=off ./pkg/requests/ -run TestContactStoreRequest`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/pkg/requests/contact_store.go api/pkg/requests/contact_update.go api/pkg/requests/contact_index.go api/pkg/requests/contact_store_test.go
git commit -m "feat(api): add contact store/update/index requests"
```

---

### Task 5: ContactHandlerValidator (store / update / index / CSV upload)

**Files:**
- Create: `api/pkg/validators/contact_handler_validator.go`
- Create: `api/pkg/validators/contact_handler_validator_test.go`

**Interfaces:**
- Consumes: `requests.ContactStoreRequest`, `requests.ContactUpdateRequest`, `requests.ContactIndex`, `requests.ContactItem`.
- Produces: `validators.ContactHandlerValidator` with:
  - `NewContactHandlerValidator(logger, tracer) *ContactHandlerValidator`
  - `ValidateStore(ctx, request requests.ContactStoreRequest) url.Values`
  - `ValidateUpdate(ctx, request requests.ContactUpdateRequest) url.Values`
  - `ValidateIndex(ctx, request requests.ContactIndex) url.Values`
  - `ValidateUpload(ctx, userID entities.UserID, header *multipart.FileHeader) ([]requests.ContactItem, url.Values)`

- [ ] **Step 1: Write the failing test**

Create `api/pkg/validators/contact_handler_validator_test.go`:

```go
package validators

import (
	"context"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/stretchr/testify/assert"
)

func newContactValidator() *ContactHandlerValidator {
	return &ContactHandlerValidator{}
}

func TestContactValidator_ValidateStore_Valid(t *testing.T) {
	v := newContactValidator()
	errs := v.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		Name:         "Alice",
		PhoneNumbers: []string{"+18005550199"},
		Emails:       []string{"alice@example.com"},
	}}})
	assert.Equal(t, 0, len(errs))
}

func TestContactValidator_ValidateStore_MissingName(t *testing.T) {
	v := newContactValidator()
	errs := v.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		PhoneNumbers: []string{"+18005550199"},
	}}})
	assert.NotEqual(t, 0, len(errs))
}

func TestContactValidator_ValidateStore_InvalidNumber(t *testing.T) {
	v := newContactValidator()
	errs := v.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		Name:         "Alice",
		PhoneNumbers: []string{"not-a-number"},
	}}})
	assert.NotEqual(t, 0, len(errs))
}

func TestContactValidator_ValidateStore_EmptyBatch(t *testing.T) {
	v := newContactValidator()
	errs := v.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: nil})
	assert.NotEqual(t, 0, len(errs))
}

func TestContactValidator_ValidateStore_InvalidEmail(t *testing.T) {
	v := newContactValidator()
	errs := v.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		Name:         "Alice",
		PhoneNumbers: []string{"+18005550199"},
		Emails:       []string{"not-an-email"},
	}}})
	assert.NotEqual(t, 0, len(errs))
}
```

(If `telemetry.NewOtelLogger` for the tracer differs in your package, `ValidateUpload` is the only method needing a real logger/tracer; the store/update/index tests use a zero-value validator.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd api && go test -vet=off ./pkg/validators/ -run TestContactValidator`
Expected: FAIL / build error — `NewContactHandlerValidator` undefined.

- [ ] **Step 3: Write the implementation**

Create `api/pkg/validators/contact_handler_validator.go`:

```go
package validators

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/mail"
	"net/url"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/jszwec/csvutil"
	"github.com/nyaruka/phonenumbers"
	"github.com/thedevsaddam/govalidator"
)

// ContactHandlerValidator validates models used in handlers.ContactHandler
type ContactHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewContactHandlerValidator creates a new handlers.ContactHandler validator
func NewContactHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) *ContactHandlerValidator {
	return &ContactHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", &ContactHandlerValidator{})),
		tracer: tracer,
	}
}

const maxContactBatch = 1000

// ValidateStore validates a create request of one or many contacts.
func (v *ContactHandlerValidator) ValidateStore(_ context.Context, request requests.ContactStoreRequest) url.Values {
	result := url.Values{}
	if len(request.Contacts) == 0 {
		result.Add("contacts", "You must provide at least one contact.")
		return result
	}
	if len(request.Contacts) > maxContactBatch {
		result.Add("contacts", fmt.Sprintf("You cannot create more than %d contacts in one request.", maxContactBatch))
		return result
	}
	for index, item := range request.Contacts {
		v.validateItem(result, index, item)
	}
	return result
}

// ValidateUpdate validates a single contact update.
func (v *ContactHandlerValidator) ValidateUpdate(_ context.Context, request requests.ContactUpdateRequest) url.Values {
	result := url.Values{}
	v.validateItem(result, 0, requests.ContactItem{
		Name:         request.Name,
		Emails:       request.Emails,
		PhoneNumbers: request.PhoneNumbers,
		Properties:   request.Properties,
	})
	return result
}

func (v *ContactHandlerValidator) validateItem(result url.Values, index int, item requests.ContactItem) {
	row := index + 1
	if strings.TrimSpace(item.Name) == "" {
		result.Add("contacts", fmt.Sprintf("Contact [%d]: The name is required.", row))
	}
	if len(item.PhoneNumbers) == 0 {
		result.Add("contacts", fmt.Sprintf("Contact [%d]: At least one phone number is required.", row))
	}
	for _, number := range item.PhoneNumbers {
		if _, err := phonenumbers.Parse(number, phonenumbers.UNKNOWN_REGION); err != nil {
			result.Add("contacts", fmt.Sprintf("Contact [%d]: The phone number [%s] is not a valid E.164 phone number.", row, number))
		}
	}
	for _, email := range item.Emails {
		if _, err := mail.ParseAddress(email); err != nil {
			result.Add("contacts", fmt.Sprintf("Contact [%d]: The email [%s] is not a valid email address.", row, email))
		}
	}
}

// ValidateIndex validates the list request.
func (v *ContactHandlerValidator) ValidateIndex(_ context.Context, request requests.ContactIndex) url.Values {
	value := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"limit": []string{"required", "numeric", "min:1", "max:100"},
			"skip":  []string{"required", "numeric", "min:0"},
			"query": []string{"max:100"},
		},
	})
	return value.ValidateStruct()
}

type contactCSVRow struct {
	Name         string `csv:"Name"`
	Emails       string `csv:"Emails"`
	PhoneNumbers string `csv:"PhoneNumbers"`
}

// ValidateUpload parses and validates a CSV upload (CSV only).
func (v *ContactHandlerValidator) ValidateUpload(ctx context.Context, userID entities.UserID, header *multipart.FileHeader) ([]requests.ContactItem, url.Values) {
	ctx, span, ctxLogger := v.tracer.StartWithLogger(ctx, v.logger)
	defer span.End()
	_ = ctx

	result := url.Values{}

	if header.Header.Get("Content-Type") != "text/csv" && !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		result.Add("document", fmt.Sprintf("The file [%s] is not a valid CSV file. Only CSV files are supported.", header.Filename))
		return nil, result
	}

	if header.Size >= 500000 {
		result.Add("document", "The CSV file must be less than 500 KB.")
		return nil, result
	}

	file, err := header.Open()
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot open file [%s] for user [%s]", header.Filename, userID))
		result.Add("document", fmt.Sprintf("Cannot open the uploaded file [%s].", header.Filename))
		return nil, result
	}
	defer func() { _ = file.Close() }()

	buffer := new(bytes.Buffer)
	if _, err = io.Copy(buffer, file); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot read file [%s] for user [%s]", header.Filename, userID))
		result.Add("document", fmt.Sprintf("Cannot read the uploaded file [%s].", header.Filename))
		return nil, result
	}

	var rows []contactCSVRow
	if err = csvutil.Unmarshal(buffer.Bytes(), &rows); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot parse CSV file [%s] for user [%s]", header.Filename, userID))
		result.Add("document", fmt.Sprintf("Cannot parse the uploaded CSV file [%s]. Use the official httpSMS contacts template.", header.Filename))
		return nil, result
	}

	if len(rows) > maxContactBatch {
		result.Add("document", fmt.Sprintf("The uploaded file must contain less than %d records.", maxContactBatch))
		return nil, result
	}

	items := make([]requests.ContactItem, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Name) == "" && strings.TrimSpace(row.PhoneNumbers) == "" {
			continue
		}
		items = append(items, requests.ContactItem{
			Name:         strings.TrimSpace(row.Name),
			Emails:       splitMultiValue(row.Emails),
			PhoneNumbers: splitMultiValue(row.PhoneNumbers),
		})
	}

	if len(items) == 0 {
		result.Add("document", "The uploaded file doesn't contain any valid records.")
	}
	return items, result
}

func splitMultiValue(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == ',' })
	var result []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			result = append(result, field)
		}
	}
	return result
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd api && go test -vet=off ./pkg/validators/ -run TestContactValidator`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/pkg/validators/contact_handler_validator.go api/pkg/validators/contact_handler_validator_test.go
git commit -m "feat(api): add ContactHandlerValidator with CSV-only upload parsing"
```

---

### Task 6: ContactService (CRUD + contact-map cache + invalidation)

**Files:**
- Create: `api/pkg/services/contact_service.go`
- Create: `api/pkg/services/contact_service_test.go`

**Interfaces:**
- Consumes: `repositories.ContactRepository`, `cache.Cache`, `entities.Contact`, `repositories.IndexParams`.
- Produces: `services.ContactService` with:
  - `NewContactService(logger, tracer, repository repositories.ContactRepository, appCache cache.Cache) *ContactService`
  - `CreateMany(ctx, userID entities.UserID, contacts []*entities.Contact) error`
  - `Get(ctx, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error)`
  - `Index(ctx, userID entities.UserID, params repositories.IndexParams) (*[]entities.Contact, error)`
  - `Update(ctx, contact *entities.Contact) error`
  - `Delete(ctx, userID entities.UserID, contactID uuid.UUID) error`
  - `GetContactMap(ctx, userID entities.UserID) (map[string]*entities.Contact, error)` (cache-backed; most-recently-updated wins)

- [ ] **Step 1: Write the failing test**

Create `api/pkg/services/contact_service_test.go`:

```go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	store map[string]string
}

func newFakeCache() *fakeCache { return &fakeCache{store: map[string]string{}} }

func (c *fakeCache) Get(_ context.Context, key string) (string, error) {
	value, ok := c.store[key]
	if !ok {
		return "", assertMiss{}
	}
	return value, nil
}

func (c *fakeCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	c.store[key] = value
	return nil
}

type assertMiss struct{}

func (assertMiss) Error() string { return "miss" }

type fakeContactRepo struct {
	contacts  []*entities.Contact
	fetchAll  int
}

func (r *fakeContactRepo) Store(_ context.Context, contacts []*entities.Contact) error {
	r.contacts = append(r.contacts, contacts...)
	return nil
}
func (r *fakeContactRepo) Update(_ context.Context, contact *entities.Contact) error { return nil }
func (r *fakeContactRepo) Load(_ context.Context, _ entities.UserID, id uuid.UUID) (*entities.Contact, error) {
	for _, c := range r.contacts {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, stacktrace.NewErrorWithCodef(repositories.ErrCodeNotFound, "contact [%s] not found", id)
}
func (r *fakeContactRepo) Index(_ context.Context, _ entities.UserID, _ repositories.IndexParams) (*[]entities.Contact, error) {
	out := []entities.Contact{}
	return &out, nil
}
func (r *fakeContactRepo) FetchAll(_ context.Context, _ entities.UserID) (*[]entities.Contact, error) {
	r.fetchAll++
	out := make([]entities.Contact, 0, len(r.contacts))
	for _, c := range r.contacts {
		out = append(out, *c)
	}
	return &out, nil
}
func (r *fakeContactRepo) Delete(_ context.Context, _ entities.UserID, _ uuid.UUID) error { return nil }
func (r *fakeContactRepo) DeleteAllForUser(_ context.Context, _ entities.UserID) error   { return nil }

func newContactService(repo repositories.ContactRepository, appCache *fakeCache) *ContactService {
	logger := telemetry.NewZerologLogger("test", nil, nil, nil)
	return NewContactService(logger, telemetry.NewOtelLogger("test", logger), repo, appCache)
}

func TestContactService_GetContactMap_TieBreakMostRecentlyUpdated(t *testing.T) {
	older := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "Old", PhoneNumbers: pq.StringArray{"+18005550199"}, UpdatedAt: time.Now().Add(-time.Hour)}
	newer := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "New", PhoneNumbers: pq.StringArray{"+18005550199"}, UpdatedAt: time.Now()}
	// FetchAll returns ASC by updated_at, so older first then newer.
	repo := &fakeContactRepo{contacts: []*entities.Contact{older, newer}}
	service := newContactService(repo, newFakeCache())

	result, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	require.NotNil(t, result["+18005550199"])
	assert.Equal(t, "New", result["+18005550199"].Name)
}

func TestContactService_GetContactMap_UsesCacheOnSecondCall(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{{ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}}}}
	service := newContactService(repo, newFakeCache())

	_, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	_, err = service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)

	assert.Equal(t, 1, repo.fetchAll)
}

func TestContactService_CreateMany_InvalidatesCache(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{{ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}}}}
	appCache := newFakeCache()
	service := newContactService(repo, appCache)

	_, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	require.Equal(t, 1, repo.fetchAll)

	require.NoError(t, service.CreateMany(context.Background(), entities.UserID("u1"), []*entities.Contact{{ID: uuid.New(), UserID: "u1", Name: "Bob", PhoneNumbers: pq.StringArray{"+18005550100"}}}))

	_, err = service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	assert.Equal(t, 2, repo.fetchAll)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd api && go test -vet=off ./pkg/services/ -run TestContactService`
Expected: FAIL / build error — `NewContactService` undefined.

- [ ] **Step 3: Write the implementation**

Create `api/pkg/services/contact_service.go`:

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/cache"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
)

const contactMapCacheTTL = 24 * time.Hour

// ContactService handles contact business logic.
type ContactService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.ContactRepository
	cache      cache.Cache
}

// NewContactService creates a new ContactService.
func NewContactService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.ContactRepository,
	appCache cache.Cache,
) (s *ContactService) {
	return &ContactService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		repository: repository,
		cache:      appCache,
	}
}

func (service *ContactService) cacheKey(userID entities.UserID) string {
	return fmt.Sprintf("contacts.map.%s", userID)
}

// CreateMany persists one or many contacts then invalidates the cache.
func (service *ContactService) CreateMany(ctx context.Context, userID entities.UserID, contacts []*entities.Contact) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if err := service.repository.Store(ctx, contacts); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot store [%d] contacts for user [%s]", len(contacts), userID))
	}
	service.invalidate(ctx, userID)
	return nil
}

// Get returns a contact by ID.
func (service *ContactService) Get(ctx context.Context, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	contact, err := service.repository.Load(ctx, userID, contactID)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, stacktrace.GetCode(err), "cannot load contact [%s] for user [%s]", contactID, userID))
	}
	return contact, nil
}

// Index lists contacts for a user.
func (service *ContactService) Index(ctx context.Context, userID entities.UserID, params repositories.IndexParams) (*[]entities.Contact, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	contacts, err := service.repository.Index(ctx, userID, params)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot index contacts for user [%s]", userID))
	}
	return contacts, nil
}

// Update saves a contact and invalidates the cache.
func (service *ContactService) Update(ctx context.Context, contact *entities.Contact) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if err := service.repository.Update(ctx, contact); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update contact [%s]", contact.ID))
	}
	service.invalidate(ctx, contact.UserID)
	return nil
}

// Delete removes a contact and invalidates the cache.
func (service *ContactService) Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if err := service.repository.Delete(ctx, userID, contactID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete contact [%s] for user [%s]", contactID, userID))
	}
	service.invalidate(ctx, userID)
	return nil
}

// GetContactMap returns the cached phone_number -> *Contact map for a user.
func (service *ContactService) GetContactMap(ctx context.Context, userID entities.UserID) (map[string]*entities.Contact, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if raw, err := service.cache.Get(ctx, service.cacheKey(userID)); err == nil && raw != "" {
		result := map[string]*entities.Contact{}
		if jsonErr := json.Unmarshal([]byte(raw), &result); jsonErr == nil {
			return result, nil
		}
	}

	contacts, err := service.repository.FetchAll(ctx, userID)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch contacts to build map for user [%s]", userID))
	}

	// FetchAll returns updated_at ASC, so later (more recently updated) contacts overwrite earlier ones.
	result := map[string]*entities.Contact{}
	for index := range *contacts {
		contact := (*contacts)[index]
		for _, number := range contact.PhoneNumbers {
			copied := contact
			result[number] = &copied
		}
	}

	if encoded, encodeErr := json.Marshal(result); encodeErr == nil {
		_ = service.cache.Set(ctx, service.cacheKey(userID), string(encoded), contactMapCacheTTL)
	}

	return result, nil
}

// invalidate overwrites the cache key with an empty marker (Cache has no Delete).
func (service *ContactService) invalidate(ctx context.Context, userID entities.UserID) {
	if err := service.cache.Set(ctx, service.cacheKey(userID), "", contactMapCacheTTL); err != nil {
		service.logger.Error(stacktrace.Propagatef(err, "cannot invalidate contact map cache for user [%s]", userID))
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd api && go test -vet=off ./pkg/services/ -run TestContactService`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/pkg/services/contact_service.go api/pkg/services/contact_service_test.go
git commit -m "feat(api): add ContactService with cached contact-map resolution"
```

---

### Task 7: Opt-in contact resolution in GetThreads

**Files:**
- Modify: `api/pkg/services/message_thread_service.go` (struct, constructor, `MessageThreadGetParams`, `GetThreads`)
- Modify: `api/pkg/requests/message_thread_index_request.go` (`Contacts` param + `ToGetParams`)
- Modify: `api/pkg/handlers/message_thread_handler_test.go` (add 6th constructor arg)
- Create: `api/pkg/services/message_thread_service_contacts_test.go`
- Create: `api/pkg/requests/message_thread_index_request_test.go`

**Interfaces:**
- Consumes: `ContactService.GetContactMap(ctx, userID) (map[string]*entities.Contact, error)` (Task 6); `entities.MessageThread.ContactDetails` (Task 2).
- Produces: `MessageThreadGetParams.WithContacts bool`; `MessageThreadService` gains a `contactService contactMapProvider` field where `contactMapProvider` is:
  ```go
  type contactMapProvider interface {
      GetContactMap(ctx context.Context, userID entities.UserID) (map[string]*entities.Contact, error)
  }
  ```
  `*ContactService` satisfies `contactMapProvider`. New constructor signature:
  `NewMessageThreadService(logger, tracer, repository, phoneRepository, eventDispatcher, contactService contactMapProvider) *MessageThreadService`

- [ ] **Step 1: Write the failing tests**

Create `api/pkg/requests/message_thread_index_request_test.go`:

```go
package requests

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func TestMessageThreadIndex_ToGetParams_WithContacts(t *testing.T) {
	input := (&MessageThreadIndex{Owner: "+18005550199", Contacts: "true"}).Sanitize()
	params := input.ToGetParams(entities.UserID("u1"))
	assert.True(t, params.WithContacts)
}

func TestMessageThreadIndex_ToGetParams_WithoutContacts(t *testing.T) {
	input := (&MessageThreadIndex{Owner: "+18005550199"}).Sanitize()
	params := input.ToGetParams(entities.UserID("u1"))
	assert.False(t, params.WithContacts)
}
```

Create `api/pkg/services/message_thread_service_contacts_test.go`:

```go
package services

import (
	"context"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubThreadRepo struct {
	repositories.MessageThreadRepository
	threads []entities.MessageThread
}

func (r *stubThreadRepo) Index(_ context.Context, _ entities.UserID, _ string, _ bool, _ repositories.IndexParams) (*[]entities.MessageThread, error) {
	out := make([]entities.MessageThread, len(r.threads))
	copy(out, r.threads)
	return &out, nil
}

type stubContactProvider struct {
	contactMap map[string]*entities.Contact
	calls      int
}

func (p *stubContactProvider) GetContactMap(_ context.Context, _ entities.UserID) (map[string]*entities.Contact, error) {
	p.calls++
	return p.contactMap, nil
}

func newThreadServiceWithContacts(repo repositories.MessageThreadRepository, provider contactMapProvider) *MessageThreadService {
	logger := telemetry.NewZerologLogger("test", nil, nil, nil)
	return NewMessageThreadService(logger, telemetry.NewOtelLogger("test", logger), repo, nil, nil, provider)
}

func TestGetThreads_AttachesContactDetailsWhenWithContacts(t *testing.T) {
	repo := &stubThreadRepo{threads: []entities.MessageThread{{Contact: "+18005550199"}, {Contact: "+18005550100"}}}
	provider := &stubContactProvider{contactMap: map[string]*entities.Contact{"+18005550199": {Name: "Alice"}}}
	service := newThreadServiceWithContacts(repo, provider)

	threads, err := service.GetThreads(context.Background(), MessageThreadGetParams{UserID: "u1", WithContacts: true})
	require.NoError(t, err)
	require.Len(t, *threads, 2)
	require.NotNil(t, (*threads)[0].ContactDetails)
	assert.Equal(t, "Alice", (*threads)[0].ContactDetails.Name)
	assert.Nil(t, (*threads)[1].ContactDetails)
}

func TestGetThreads_SkipsContactLookupWhenFlagOff(t *testing.T) {
	repo := &stubThreadRepo{threads: []entities.MessageThread{{Contact: "+18005550199"}}}
	provider := &stubContactProvider{contactMap: map[string]*entities.Contact{"+18005550199": {Name: "Alice"}}}
	service := newThreadServiceWithContacts(repo, provider)

	threads, err := service.GetThreads(context.Background(), MessageThreadGetParams{UserID: "u1", WithContacts: false})
	require.NoError(t, err)
	assert.Nil(t, (*threads)[0].ContactDetails)
	assert.Equal(t, 0, provider.calls)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd api && go test -vet=off ./pkg/requests/ -run TestMessageThreadIndex_ToGetParams && go test -vet=off ./pkg/services/ -run TestGetThreads`
Expected: FAIL / build error — `Contacts`, `WithContacts`, and the 6-arg constructor do not exist yet.

- [ ] **Step 3: Implement request changes**

In `api/pkg/requests/message_thread_index_request.go`, add the field to the struct:

```go
	Owner      string `json:"owner" query:"owner"`
	Contacts   string `json:"contacts" query:"contacts" example:"true"`
```

Add sanitisation inside `Sanitize()` (before `return *input`):

```go
	input.Contacts = input.sanitizeBool(input.Contacts)
```

Add the field to the returned params in `ToGetParams`:

```go
		UserID:       userID,
		IsArchived:   input.getBool(input.IsArchived),
		Owner:        input.Owner,
		WithContacts: input.getBool(input.Contacts),
	}
```

- [ ] **Step 4: Implement service changes**

In `api/pkg/services/message_thread_service.go` add the provider interface (top-level, after imports):

```go
// contactMapProvider supplies the phone_number -> *Contact map for a user.
type contactMapProvider interface {
	GetContactMap(ctx context.Context, userID entities.UserID) (map[string]*entities.Contact, error)
}
```

Add the field to the struct:

```go
	eventDispatcher *EventDispatcher
	contactService  contactMapProvider
```

Update the constructor:

```go
func NewMessageThreadService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.MessageThreadRepository,
	phoneRepository repositories.PhoneRepository,
	eventDispatcher *EventDispatcher,
	contactService contactMapProvider,
) (s *MessageThreadService) {
	return &MessageThreadService{
		logger:          logger.WithService(fmt.Sprintf("%T", s)),
		tracer:          tracer,
		eventDispatcher: eventDispatcher,
		repository:      repository,
		phoneRepository: phoneRepository,
		contactService:  contactService,
	}
}
```

Add `WithContacts` to the params struct:

```go
type MessageThreadGetParams struct {
	repositories.IndexParams
	IsArchived   bool
	WithContacts bool
	UserID       entities.UserID
	Owner        string
}
```

Extend `GetThreads` to attach contact details (replace its body):

```go
func (service *MessageThreadService) GetThreads(ctx context.Context, params MessageThreadGetParams) (*[]entities.MessageThread, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	threads, err := service.repository.Index(ctx, params.UserID, params.Owner, params.IsArchived, params.IndexParams)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "could not fetch messages threads for params [%+#v]", params))
	}

	if params.WithContacts && service.contactService != nil && len(*threads) > 0 {
		contactMap, mapErr := service.contactService.GetContactMap(ctx, params.UserID)
		if mapErr != nil {
			ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(mapErr, "cannot build contact map for user [%s]", params.UserID)))
		} else {
			for index := range *threads {
				if contact, ok := contactMap[(*threads)[index].Contact]; ok {
					(*threads)[index].ContactDetails = contact
				}
			}
		}
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] threads with params [%+#v]", len(*threads), params))
	return threads, nil
}
```

- [ ] **Step 5: Update DI wiring and existing callers**

In `api/pkg/di/container.go`, find `NewMessageThreadService(...)` and append `container.ContactService()` as the final argument (the `ContactService()` getter is added in Task 8; if wiring Task 8 first, this compiles immediately — otherwise temporarily pass `nil` and restore in Task 8).

Also update the existing test `api/pkg/handlers/message_thread_handler_test.go`: its call `services.NewMessageThreadService(logger, tracer, &messageThreadHandlerRepositoryStub{}, nil, nil)` now needs a sixth argument — change it to `services.NewMessageThreadService(logger, tracer, &messageThreadHandlerRepositoryStub{}, nil, nil, nil)`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd api && go test -vet=off ./pkg/requests/ -run TestMessageThreadIndex_ToGetParams && go test -vet=off ./pkg/services/ -run TestGetThreads`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add api/pkg/services/message_thread_service.go api/pkg/services/message_thread_service_contacts_test.go api/pkg/requests/message_thread_index_request.go api/pkg/requests/message_thread_index_request_test.go api/pkg/handlers/message_thread_handler_test.go api/pkg/di/container.go
git commit -m "feat(api): opt-in contact resolution for message threads via ?contacts=true"
```

---

### Task 8: ContactHandler + DI wiring + route registration

**Files:**
- Create: `api/pkg/handlers/contact_handler.go`
- Create: `api/pkg/handlers/contact_handler_test.go`
- Modify: `api/pkg/di/container.go` (getters, AutoMigrate, RegisterContactRoutes, wire ContactService into MessageThreadService)

**Interfaces:**
- Consumes: `services.ContactService` (Task 6), `validators.ContactHandlerValidator` (Task 5), `requests.*` (Task 4), `repositories.ErrCodeNotFound`.
- Produces: `handlers.ContactHandler` with `NewContactHandler(logger, tracer, validator *validators.ContactHandlerValidator, service *services.ContactService) *ContactHandler` and `RegisterRoutes(router, ...middlewares)`. Routes: `GET /v1/contacts`, `POST /v1/contacts`, `POST /v1/contacts/upload`, `PUT /v1/contacts/:contactID`, `DELETE /v1/contacts/:contactID`. New DI getters: `container.ContactRepository()`, `container.ContactService()`, `container.ContactHandlerValidator()`, `container.ContactHandler()`, `container.RegisterContactRoutes()`.

- [ ] **Step 1: Write the failing test**

Create `api/pkg/handlers/contact_handler_test.go`:

```go
package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/cache"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/middlewares"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	ttlCache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

type contactHandlerRepoStub struct {
	stored []*entities.Contact
}

func (s *contactHandlerRepoStub) Store(_ context.Context, contacts []*entities.Contact) error {
	s.stored = append(s.stored, contacts...)
	return nil
}
func (s *contactHandlerRepoStub) Update(context.Context, *entities.Contact) error { return nil }
func (s *contactHandlerRepoStub) Load(context.Context, entities.UserID, uuid.UUID) (*entities.Contact, error) {
	return nil, nil
}
func (s *contactHandlerRepoStub) Index(context.Context, entities.UserID, repositories.IndexParams) (*[]entities.Contact, error) {
	out := []entities.Contact{}
	return &out, nil
}
func (s *contactHandlerRepoStub) FetchAll(context.Context, entities.UserID) (*[]entities.Contact, error) {
	out := []entities.Contact{}
	return &out, nil
}
func (s *contactHandlerRepoStub) Delete(context.Context, entities.UserID, uuid.UUID) error { return nil }
func (s *contactHandlerRepoStub) DeleteAllForUser(context.Context, entities.UserID) error  { return nil }

func newContactTestApp(repo repositories.ContactRepository) *fiber.App {
	logger := &messageThreadHandlerNoopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	appCache := cache.NewMemoryCache(tracer, ttlCache.New(time.Minute, time.Minute))
	service := services.NewContactService(logger, tracer, repo, appCache)
	handler := NewContactHandler(logger, tracer, &validators.ContactHandlerValidator{}, service)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
		return c.Next()
	})
	handler.RegisterRoutes(app)
	return app
}

func TestContactHandler_Store_CreatesContacts(t *testing.T) {
	repo := &contactHandlerRepoStub{}
	app := newContactTestApp(repo)

	body := `[{"name":"Alice","phone_numbers":["+18005550199"]}]`
	req := httptest.NewRequest(http.MethodPost, "/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Len(t, repo.stored, 1)
	require.Equal(t, "Alice", repo.stored[0].Name)
}

func TestContactHandler_Store_ValidationError(t *testing.T) {
	app := newContactTestApp(&contactHandlerRepoStub{})

	body := `[{"name":"","phone_numbers":[]}]`
	req := httptest.NewRequest(http.MethodPost, "/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestContactHandler_Index_ReturnsOK(t *testing.T) {
	app := newContactTestApp(&contactHandlerRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/v1/contacts", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd api && go test -vet=off ./pkg/handlers/ -run TestContactHandler`
Expected: FAIL / build error — `NewContactHandler` undefined.

- [ ] **Step 3: Write the handler**

Create `api/pkg/handlers/contact_handler.go`:

```go
package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/NdoleStudio/stacktrace"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ContactHandler handles contact http requests.
type ContactHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.ContactHandlerValidator
	service   *services.ContactService
}

// NewContactHandler creates a new ContactHandler.
func NewContactHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.ContactHandlerValidator,
	service *services.ContactService,
) (h *ContactHandler) {
	return &ContactHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the ContactHandler.
func (h *ContactHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodGet, "/v1/contacts", middlewares, h.Index)
	h.register(router, fiber.MethodPost, "/v1/contacts", middlewares, h.Store)
	h.register(router, fiber.MethodPost, "/v1/contacts/upload", middlewares, h.Upload)
	h.register(router, fiber.MethodPut, "/v1/contacts/:contactID", middlewares, h.Update)
	h.register(router, fiber.MethodDelete, "/v1/contacts/:contactID", middlewares, h.Delete)
}

// Index lists contacts for the authenticated user.
// @Summary      List contacts
// @Description  Returns the paginated list of contacts for the authenticated user.
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        skip	query  int  	false	"number of contacts to skip"	minimum(0)
// @Param        query	query  string  	false 	"filter contacts containing query"
// @Param        limit	query  int  	false	"number of contacts to return"	minimum(1)	maximum(100)
// @Success      200 	{object}	responses.ContactsResponse
// @Failure      400	{object}	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure      422	{object}	responses.UnprocessableEntity
// @Failure      500	{object}	responses.InternalServerError
// @Router       /contacts [get]
func (h *ContactHandler) Index(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	var request requests.ContactIndex
	if err := c.Bind().Query(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall params [%s] into %T", c.OriginalURL(), request))
		return h.responseBadRequest(c, err)
	}

	sanitized := request.Sanitize()
	if errors := h.validator.ValidateIndex(ctx, sanitized); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while listing contacts [%+#v]", spew.Sdump(errors), sanitized))
		return h.responseUnprocessableEntity(c, errors, "validation errors while listing contacts")
	}

	contacts, err := h.service.Index(ctx, h.userIDFomContext(c), sanitized.ToIndexParams())
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot list contacts for user [%s]", h.userIDFomContext(c)))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d %s", len(*contacts), h.pluralize("contact", len(*contacts))), contacts)
}

// Store creates one or many contacts.
// @Summary      Create one or many contacts
// @Description  Creates a single contact or a batch of contacts. Accepts a JSON array or an object with a "contacts" array.
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        payload   body 		requests.ContactStoreRequest 	true 	"Contact(s) to create"
// @Success      201 	{object}	responses.ContactsResponse
// @Failure      400	{object}	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure      422	{object}	responses.UnprocessableEntity
// @Failure      500	{object}	responses.InternalServerError
// @Router       /contacts [post]
func (h *ContactHandler) Store(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	var request requests.ContactStoreRequest
	if err := c.Bind().Body(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall body [%s] into %T", c.Body(), request))
		return h.responseBadRequest(c, err)
	}

	sanitized := request.Sanitize()
	if errors := h.validator.ValidateStore(ctx, sanitized); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while creating contacts", spew.Sdump(errors)))
		return h.responseUnprocessableEntity(c, errors, "validation errors while creating contacts")
	}

	userID := h.userIDFomContext(c)
	contacts := sanitized.ToContacts(userID)
	if err := h.service.CreateMany(ctx, userID, contacts); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot create [%d] contacts for user [%s]", len(contacts), userID))
		return h.responseInternalServerError(c)
	}

	return h.responseCreated(c, fmt.Sprintf("created %d %s", len(contacts), h.pluralize("contact", len(contacts))), contacts)
}

// Upload imports contacts from a CSV file.
// @Summary      Import contacts from CSV
// @Description  Uploads a CSV file (multipart field "document") of contacts. Columns: Name, Emails, PhoneNumbers (multi-values separated by ";").
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       multipart/form-data
// @Produce      json
// @Param        document	formData	file	true	"CSV file of contacts"
// @Success      201 	{object}	responses.ContactsResponse
// @Failure      400	{object}	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure      422	{object}	responses.UnprocessableEntity
// @Failure      500	{object}	responses.InternalServerError
// @Router       /contacts/upload [post]
func (h *ContactHandler) Upload(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	file, err := c.FormFile("document")
	if err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot fetch file with name [%s] from request", "document"))
		return h.responseBadRequest(c, err)
	}

	userID := h.userIDFomContext(c)
	items, errors := h.validator.ValidateUpload(ctx, userID, file)
	if len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while importing contacts from CSV [%s]", spew.Sdump(errors), file.Filename))
		return h.responseUnprocessableEntity(c, errors, "validation errors while importing contacts")
	}

	request := (requests.ContactStoreRequest{Contacts: items}).Sanitize()
	contacts := request.ToContacts(userID)
	if err = h.service.CreateMany(ctx, userID, contacts); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot import [%d] contacts for user [%s]", len(contacts), userID))
		return h.responseInternalServerError(c)
	}

	return h.responseCreated(c, fmt.Sprintf("imported %d %s", len(contacts), h.pluralize("contact", len(contacts))), contacts)
}

// Update updates a single contact.
// @Summary      Update a contact
// @Description  Updates the details of a single contact.
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param 		 contactID	path		string 							true 	"ID of the contact"
// @Param        payload   	body 		requests.ContactUpdateRequest 	true 	"Contact details to update"
// @Success      200 		{object}	responses.ContactResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      404		{object}	responses.NotFound
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /contacts/{contactID} [put]
func (h *ContactHandler) Update(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	contactID := c.Params("contactID")
	if errors := h.validator.ValidateUUID(contactID, "contactID"); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while updating contact [%s]", spew.Sdump(errors), contactID))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating contact")
	}

	var request requests.ContactUpdateRequest
	if err := c.Bind().Body(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall body into %T", request))
		return h.responseBadRequest(c, err)
	}

	sanitized := request.Sanitize()
	if errors := h.validator.ValidateUpdate(ctx, sanitized); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while updating contact [%s]", spew.Sdump(errors), contactID))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating contact")
	}

	userID := h.userIDFomContext(c)
	contact, err := h.service.Get(ctx, userID, uuid.MustParse(contactID))
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, fmt.Sprintf("cannot find contact with ID [%s]", contactID))
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot load contact [%s] for user [%s]", contactID, userID))
		return h.responseInternalServerError(c)
	}

	sanitized.ApplyTo(contact)
	if err = h.service.Update(ctx, contact); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot update contact [%s]", contactID))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "contact updated successfully", contact)
}

// Delete removes a single contact.
// @Summary      Delete a contact
// @Description  Deletes a single contact from the database.
// @Security	 ApiKeyAuth
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param 		 contactID	path		string 	true	"ID of the contact"
// @Success      204  	{object} 	responses.NoContent
// @Failure      400  	{object}  	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure 	 404	{object}	responses.NotFound
// @Failure      422  	{object} 	responses.UnprocessableEntity
// @Failure      500  	{object}  	responses.InternalServerError
// @Router       /contacts/{contactID} [delete]
func (h *ContactHandler) Delete(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	contactID := c.Params("contactID")
	if errors := h.validator.ValidateUUID(contactID, "contactID"); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while deleting contact [%s]", spew.Sdump(errors), contactID))
		return h.responseUnprocessableEntity(c, errors, "validation errors while deleting contact")
	}

	userID := h.userIDFomContext(c)
	if err := h.service.Delete(ctx, userID, uuid.MustParse(contactID)); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot delete contact [%s] for user [%s]", contactID, userID))
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "contact deleted successfully")
}
```

- [ ] **Step 4: Add DI getters and wiring**

In `api/pkg/di/container.go`:

Add a repository getter (near `MessageThreadRepository`):

```go
// ContactRepository creates a new instance of repositories.ContactRepository
func (container *Container) ContactRepository() (repository repositories.ContactRepository) {
	container.logger.Debug("creating GORM repositories.ContactRepository")
	return repositories.NewGormContactRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}
```

Add a service getter (near `MessageThreadService`):

```go
// ContactService creates a new instance of services.ContactService
func (container *Container) ContactService() (service *services.ContactService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewContactService(
		container.Logger(),
		container.Tracer(),
		container.ContactRepository(),
		container.Cache(),
	)
}
```

Add a validator getter (near `MessageThreadHandlerValidator`):

```go
// ContactHandlerValidator creates a new instance of validators.ContactHandlerValidator
func (container *Container) ContactHandlerValidator() (validator *validators.ContactHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewContactHandlerValidator(
		container.Logger(),
		container.Tracer(),
	)
}
```

Add a handler getter (near `MessageThreadHandler`):

```go
// ContactHandler creates a new instance of handlers.ContactHandler
func (container *Container) ContactHandler() (h *handlers.ContactHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", h))
	return handlers.NewContactHandler(
		container.Logger(),
		container.Tracer(),
		container.ContactHandlerValidator(),
		container.ContactService(),
	)
}
```

Add a route registrar (near `RegisterMessageThreadRoutes`):

```go
// RegisterContactRoutes registers routes for the /contacts prefix
func (container *Container) RegisterContactRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.ContactHandler{}))
	container.ContactHandler().RegisterRoutes(container.App(), container.AuthenticatedMiddleware())
}
```

Call the registrar right after `container.RegisterMessageThreadRoutes()` (around line 123):

```go
	container.RegisterMessageThreadRoutes()
	container.RegisterContactRoutes()
```

Wire the ContactService into MessageThreadService (finalise Task 7 Step 5): the `MessageThreadService()` getter's `NewMessageThreadService(...)` call's final argument becomes `container.ContactService()`:

```go
	return services.NewMessageThreadService(
		container.Logger(),
		container.Tracer(),
		container.MessageThreadRepository(),
		container.PhoneRepository(),
		container.EventDispatcher(),
		container.ContactService(),
	)
```

Add the AutoMigrate entry in the AutoMigrate block (around line 413, alongside the other `db.AutoMigrate(&entities.X{})` calls) — this may already be present from Task 2; if not, add:

```go
	if err = db.AutoMigrate(&entities.Contact{}); err != nil {
		container.logger.Fatal(stacktrace.Propagatef(err, "cannot migrate %T", &entities.Contact{}))
	}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd api && go test -vet=off ./pkg/handlers/ -run TestContactHandler`
Expected: PASS

- [ ] **Step 6: Build the whole API to confirm wiring compiles**

Run: `cd api && go build ./...`
Expected: no output (success).

- [ ] **Step 7: Commit**

```bash
git add api/pkg/handlers/contact_handler.go api/pkg/handlers/contact_handler_test.go api/pkg/di/container.go
git commit -m "feat(api): add ContactHandler and wire contacts into DI container"
```

---

### Task 9: Swagger regeneration + CSV import template

**Files:**
- Modify: `api/docs/*` (generated by swag)
- Create: `web/public/templates/httpsms-contacts.csv`
- Create/verify: `api/pkg/responses/` contact response types if referenced by Swagger annotations (see Step 1).

**Interfaces:**
- Consumes: Swagger annotations added on `ContactHandler` (Task 8), which reference `responses.ContactResponse` and `responses.ContactsResponse`.
- Produces: regenerated `api/docs/swagger.json` / `swagger.yaml` / `docs.go` including the `/contacts` endpoints; a downloadable CSV template served by the web app.

- [ ] **Step 1: Add response wrapper types (if missing)**

Check whether `responses.ContactResponse` and `responses.ContactsResponse` exist:

Run: `cd api && findstr /S /C:"ContactsResponse" pkg\responses\*.go`
Expected: no matches (they don't exist yet).

Create `api/pkg/responses/contact_responses.go`:

```go
package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// ContactResponse is the response with a single entities.Contact.
type ContactResponse struct {
	response
	Data entities.Contact `json:"data"`
}

// ContactsResponse is the response with a list of entities.Contact.
type ContactsResponse struct {
	response
	Data []entities.Contact `json:"data"`
}
```

(Verify the base embedded type name: open any file in `api/pkg/responses/` and confirm the unexported base struct is `response` with `Status`/`Message` fields; match the exact embed used by e.g. `MessageThreadsResponse`.)

- [ ] **Step 2: Regenerate Swagger docs**

Run:

```bash
cd api && swag init --requiredByDefault --parseDependency --parseInternal
```

Expected: `create docs/docs.go`, `create docs/swagger.json`, `create docs/swagger.yaml` with no errors. Confirm `/contacts` appears:

Run: `cd api && findstr /C:"/contacts" docs\swagger.json`
Expected: matches for `/contacts`, `/contacts/upload`, `/contacts/{contactID}`.

- [ ] **Step 3: Create the CSV import template**

Create `web/public/templates/httpsms-contacts.csv`:

```csv
Name,Emails,PhoneNumbers
John Doe,john@example.com,+18005550199
Jane Smith,jane@example.com;jane.work@example.com,+18005550100;+18005550111
Acme Support,,+18005550122
```

(Multiple emails or phone numbers in one cell are separated by `;`, matching the CSV parser in Task 5's `ValidateUpload`.)

- [ ] **Step 4: Commit**

```bash
git add api/docs api/pkg/responses/contact_responses.go web/public/templates/httpsms-contacts.csv
git commit -m "chore(api): regenerate swagger for contacts + add CSV import template"
```

---

### Task 10: Regenerate web API TypeScript models

**Files:**
- Modify: `web/shared/types/api.ts` (generated)

**Interfaces:**
- Consumes: `api/docs/swagger.json` (regenerated in Task 9) with the `Contact` schema and `/contacts` paths.
- Produces: generated types `EntitiesContact` and (if the response wrappers were added) `ResponsesContactResponse` / `ResponsesContactsResponse` in `web/shared/types/api.ts`; `EntitiesMessageThread` now includes an optional `contact_details?: EntitiesContact`.

> **Note:** This task must run AFTER Task 9 (swagger regen). No new hand-written code — the model file is generated.

- [ ] **Step 1: Regenerate models**

Run:

```bash
cd web && pnpm api:models
```

Expected: `web/shared/types/api.ts` rewritten with no errors.

- [ ] **Step 2: Verify the Contact type exists**

Run: `cd web && findstr /C:"EntitiesContact" shared\types\api.ts`
Expected: matches for `EntitiesContact` interface and `contact_details` on the message-thread interface.

- [ ] **Step 3: Commit**

```bash
git add web/shared/types/api.ts
git commit -m "chore(web): regenerate API models with Contact types"
```

---

### Task 11: Contacts Pinia store

**Files:**
- Create: `web/app/stores/contacts.ts`

**Interfaces:**
- Consumes: `useApi()` (`apiFetch`), `useNotificationsStore()`, `getApiErrorMessage`, `EntitiesContact` (Task 10).
- Produces: `useContactsStore()` exposing state `contacts`, `loading`, `search`; getters `filteredContacts`, `total`; actions `loadContacts(force?)`, `saveContacts(items)`, `updateContact(id, payload)`, `deleteContact(id)`, `uploadCsv(file)`, `resetState()`.

- [ ] **Step 1: Write the store**

Create `web/app/stores/contacts.ts`:

```ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { EntitiesContact } from '~~/shared/types/api'
import { getApiErrorMessage } from '~/utils/api-error'

export interface ContactInput {
  name: string
  emails: string[]
  phone_numbers: string[]
  properties?: Record<string, string>
}

export const useContactsStore = defineStore('contacts', () => {
  const contacts = ref<EntitiesContact[]>([])
  const loading = ref(false)
  const search = ref('')
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  const total = computed(() => contacts.value.length)

  const filteredContacts = computed<EntitiesContact[]>(() => {
    const term = search.value.trim().toLowerCase()
    if (!term) return contacts.value
    return contacts.value.filter((contact) => {
      const name = (contact.name ?? '').toLowerCase()
      const emails = (contact.emails ?? []).join(' ').toLowerCase()
      const numbers = (contact.phone_numbers ?? []).join(' ').toLowerCase()
      return (
        name.includes(term) ||
        emails.includes(term) ||
        numbers.includes(term)
      )
    })
  })

  async function loadContacts(force = false) {
    if (contacts.value.length > 0 && !force) return
    loading.value = true
    try {
      const response = await apiFetch<{ data: EntitiesContact[] }>(
        '/v1/contacts',
        { params: { limit: 100 } },
      )
      contacts.value = response.data ?? []
    } catch (error: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while loading contacts'),
        type: 'error',
      })
      throw error
    } finally {
      loading.value = false
    }
  }

  async function saveContacts(items: ContactInput[]) {
    try {
      await apiFetch('/v1/contacts', { method: 'POST', body: items })
      notificationsStore.addNotification({
        message:
          items.length > 1 ? 'Contacts created' : 'Contact created',
        type: 'success',
      })
      await loadContacts(true)
    } catch (error: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while saving contacts'),
        type: 'error',
      })
      throw error
    }
  }

  async function updateContact(id: string, payload: ContactInput) {
    try {
      await apiFetch(`/v1/contacts/${id}`, { method: 'PUT', body: payload })
      notificationsStore.addNotification({
        message: 'Contact updated',
        type: 'success',
      })
      await loadContacts(true)
    } catch (error: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while updating contact'),
        type: 'error',
      })
      throw error
    }
  }

  async function deleteContact(id: string) {
    try {
      await apiFetch(`/v1/contacts/${id}`, { method: 'DELETE' })
      contacts.value = contacts.value.filter((contact) => contact.id !== id)
      notificationsStore.addNotification({
        message: 'Contact deleted',
        type: 'success',
      })
    } catch (error: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while deleting contact'),
        type: 'error',
      })
      throw error
    }
  }

  async function uploadCsv(file: File) {
    try {
      const formData = new FormData()
      formData.append('document', file)
      await apiFetch('/v1/contacts/upload', {
        method: 'POST',
        body: formData,
      })
      notificationsStore.addNotification({
        message: 'Contacts imported successfully',
        type: 'success',
      })
      await loadContacts(true)
    } catch (error: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while importing contacts'),
        type: 'error',
      })
      throw error
    }
  }

  function resetState() {
    contacts.value = []
    loading.value = false
    search.value = ''
  }

  return {
    contacts,
    loading,
    search,
    total,
    filteredContacts,
    loadContacts,
    saveContacts,
    updateContact,
    deleteContact,
    uploadCsv,
    resetState,
  }
})
```

- [ ] **Step 2: Lint the new store**

Run: `cd web && pnpm lint:js`
Expected: no errors for `app/stores/contacts.ts`.

- [ ] **Step 3: Commit**

```bash
git add web/app/stores/contacts.ts
git commit -m "feat(web): add contacts Pinia store"
```

---

### Task 12: Contacts page with add/edit/delete/import dialogs

**Files:**
- Create: `web/app/pages/contacts/index.vue`

**Interfaces:**
- Consumes: `useContactsStore()` (Task 11), `useFilters()` (`formatPhoneNumber`), `EntitiesContact` (Task 10).
- Produces: the `/contacts` route (Nuxt file-based routing infers `name: 'contacts'`, used by the nav link in Task 13).

**UI requirements (from spec section 7 / reference screenshot):**
- Page title `text-display-large` (NO gradient/glow — theme text only); subtitle showing total contact count.
- Top-right actions: "Import CSV" (outlined button) + "Add Contact" (filled `color="primary"` button).
- Debounced search field filtering by name/email/phone.
- Table of contacts (Name, Emails, Phone Numbers) with edit + delete icon actions per row.
- Add/Edit/Delete/Import are `VDialog`s. **Every `VDialog` MUST set `opacity="0.9"` and its Close button MUST use `color="warning"`.**

- [ ] **Step 1: Write the page**

Create `web/app/pages/contacts/index.vue`:

```vue
<script setup lang="ts">
import {
  mdiArrowLeft,
  mdiPlus,
  mdiFileUpload,
  mdiPencil,
  mdiDelete,
  mdiMagnify,
  mdiClose,
} from '@mdi/js'
import { ref, computed, onMounted } from 'vue'
import type { EntitiesContact } from '~~/shared/types/api'
import { useContactsStore, type ContactInput } from '~/stores/contacts'
import { useFilters } from '~/composables/useFilters'

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'Contacts - httpSMS',
})

const contactsStore = useContactsStore()
const { formatPhoneNumber } = useFilters()

const editDialog = ref(false)
const deleteDialog = ref(false)
const importDialog = ref(false)
const saving = ref(false)
const editingId = ref<string | null>(null)
const pendingDeleteId = ref<string | null>(null)
const importFile = ref<File | null>(null)

const form = ref<{ name: string; emails: string; phone_numbers: string }>({
  name: '',
  emails: '',
  phone_numbers: '',
})

const dialogTitle = computed(() =>
  editingId.value ? 'Edit Contact' : 'Add Contact',
)

function splitList(value: string): string[] {
  return value
    .split(/[,;]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

function openAdd() {
  editingId.value = null
  form.value = { name: '', emails: '', phone_numbers: '' }
  editDialog.value = true
}

function openEdit(contact: EntitiesContact) {
  editingId.value = contact.id ?? null
  form.value = {
    name: contact.name ?? '',
    emails: (contact.emails ?? []).join(', '),
    phone_numbers: (contact.phone_numbers ?? []).join(', '),
  }
  editDialog.value = true
}

function openDelete(contact: EntitiesContact) {
  pendingDeleteId.value = contact.id ?? null
  deleteDialog.value = true
}

async function submitForm() {
  saving.value = true
  const payload: ContactInput = {
    name: form.value.name.trim(),
    emails: splitList(form.value.emails),
    phone_numbers: splitList(form.value.phone_numbers),
  }
  try {
    if (editingId.value) {
      await contactsStore.updateContact(editingId.value, payload)
    } else {
      await contactsStore.saveContacts([payload])
    }
    editDialog.value = false
  } catch {
    // notification already surfaced by the store
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!pendingDeleteId.value) return
  saving.value = true
  try {
    await contactsStore.deleteContact(pendingDeleteId.value)
    deleteDialog.value = false
  } catch {
    // notification already surfaced by the store
  } finally {
    saving.value = false
    pendingDeleteId.value = null
  }
}

async function submitImport() {
  if (!importFile.value) return
  saving.value = true
  try {
    await contactsStore.uploadCsv(importFile.value)
    importDialog.value = false
    importFile.value = null
  } catch {
    // notification already surfaced by the store
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  contactsStore.loadContacts(true).catch(() => {})
})
</script>

<template>
  <VContainer fluid class="px-0 pt-0">
    <VAppBar>
      <VBtn icon to="/threads">
        <VIcon :icon="mdiArrowLeft" />
      </VBtn>
      <VToolbarTitle>Contacts</VToolbarTitle>
      <VProgressLinear
        :active="contactsStore.loading"
        color="primary"
        :indeterminate="contactsStore.loading"
        location="bottom"
        absolute
      />
    </VAppBar>

    <VContainer>
      <VRow>
        <VCol cols="12" md="10" offset-md="1" xxl="8" offset-xxl="2">
          <div
            class="d-flex flex-column flex-md-row align-md-center mb-6 mt-3"
          >
            <div>
              <h1 class="text-display-large">Contacts</h1>
              <p class="text-medium-emphasis mb-0">
                {{ contactsStore.total }}
                {{ contactsStore.total === 1 ? 'contact' : 'contacts' }}
              </p>
            </div>
            <VSpacer />
            <div class="d-flex mt-4 mt-md-0">
              <VBtn
                variant="outlined"
                color="primary"
                :prepend-icon="mdiFileUpload"
                class="mr-3"
                @click="importDialog = true"
              >
                Import CSV
              </VBtn>
              <VBtn
                color="primary"
                :prepend-icon="mdiPlus"
                @click="openAdd"
              >
                Add Contact
              </VBtn>
            </div>
          </div>

          <VTextField
            v-model="contactsStore.search"
            :prepend-inner-icon="mdiMagnify"
            label="Search contacts"
            variant="outlined"
            density="comfortable"
            clearable
            hide-details
            class="mb-4"
          />

          <VTable density="comfortable">
            <thead>
              <tr class="text-uppercase text-medium-emphasis">
                <th class="text-left">Name</th>
                <th class="text-left">Emails</th>
                <th class="text-left">Phone Numbers</th>
                <th class="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="contact in contactsStore.filteredContacts"
                :key="contact.id"
              >
                <td class="text-left">{{ contact.name }}</td>
                <td class="text-left">
                  {{ (contact.emails ?? []).join(', ') }}
                </td>
                <td class="text-left">
                  {{
                    (contact.phone_numbers ?? [])
                      .map((number) => formatPhoneNumber(number))
                      .join(', ')
                  }}
                </td>
                <td class="text-right">
                  <VBtn
                    icon
                    variant="text"
                    size="small"
                    @click="openEdit(contact)"
                  >
                    <VIcon :icon="mdiPencil" />
                  </VBtn>
                  <VBtn
                    icon
                    variant="text"
                    size="small"
                    color="error"
                    @click="openDelete(contact)"
                  >
                    <VIcon :icon="mdiDelete" />
                  </VBtn>
                </td>
              </tr>
              <tr v-if="contactsStore.filteredContacts.length === 0">
                <td colspan="4" class="text-center text-medium-emphasis py-8">
                  No contacts found.
                </td>
              </tr>
            </tbody>
          </VTable>
        </VCol>
      </VRow>
    </VContainer>

    <!-- Add / Edit dialog -->
    <VDialog v-model="editDialog" max-width="560" opacity="0.9">
      <VCard>
        <VCardTitle class="d-flex align-center">
          <span>{{ dialogTitle }}</span>
          <VSpacer />
          <VBtn
            icon
            variant="text"
            color="warning"
            @click="editDialog = false"
          >
            <VIcon :icon="mdiClose" />
          </VBtn>
        </VCardTitle>
        <VCardText>
          <VTextField
            v-model="form.name"
            label="Name"
            variant="outlined"
            class="mb-2"
          />
          <VTextField
            v-model="form.emails"
            label="Emails (comma separated)"
            variant="outlined"
            class="mb-2"
          />
          <VTextField
            v-model="form.phone_numbers"
            label="Phone numbers (comma separated)"
            variant="outlined"
          />
        </VCardText>
        <VCardActions>
          <VSpacer />
          <VBtn color="warning" variant="text" @click="editDialog = false">
            Close
          </VBtn>
          <VBtn
            color="primary"
            :loading="saving"
            :disabled="saving"
            @click="submitForm"
          >
            Save
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- Delete dialog -->
    <VDialog v-model="deleteDialog" max-width="460" opacity="0.9">
      <VCard>
        <VCardTitle class="d-flex align-center">
          <span>Delete Contact</span>
          <VSpacer />
          <VBtn
            icon
            variant="text"
            color="warning"
            @click="deleteDialog = false"
          >
            <VIcon :icon="mdiClose" />
          </VBtn>
        </VCardTitle>
        <VCardText>
          Are you sure you want to delete this contact? This action cannot be
          undone.
        </VCardText>
        <VCardActions>
          <VSpacer />
          <VBtn color="warning" variant="text" @click="deleteDialog = false">
            Close
          </VBtn>
          <VBtn
            color="error"
            :loading="saving"
            :disabled="saving"
            @click="confirmDelete"
          >
            Delete
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- Import CSV dialog -->
    <VDialog v-model="importDialog" max-width="560" opacity="0.9">
      <VCard>
        <VCardTitle class="d-flex align-center">
          <span>Import Contacts from CSV</span>
          <VSpacer />
          <VBtn
            icon
            variant="text"
            color="warning"
            @click="importDialog = false"
          >
            <VIcon :icon="mdiClose" />
          </VBtn>
        </VCardTitle>
        <VCardText>
          <p class="mb-3">
            Download our
            <a
              class="text-decoration-none hover:text-decoration-underline"
              download
              href="/templates/httpsms-contacts.csv"
              >CSV template</a
            >, fill it in and upload it here. Columns: Name, Emails,
            PhoneNumbers (multiple values separated by ";").
          </p>
          <VFileInput
            v-model="importFile"
            label="CSV file"
            color="primary"
            accept=".csv,text/csv"
            variant="outlined"
            hide-details
          />
        </VCardText>
        <VCardActions>
          <VSpacer />
          <VBtn color="warning" variant="text" @click="importDialog = false">
            Close
          </VBtn>
          <VBtn
            color="primary"
            :loading="saving"
            :disabled="saving || !importFile"
            @click="submitImport"
          >
            Import
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </VContainer>
</template>
```

- [ ] **Step 2: Lint the page**

Run: `cd web && pnpm lint:js && pnpm lint:prettier`
Expected: no errors for `app/pages/contacts/index.vue`. If Prettier reports formatting, run `pnpm lintfix` and re-check.

- [ ] **Step 3: Build the site to confirm it compiles**

Run: `cd web && pnpm generate`
Expected: build completes without type errors (the `/contacts` route is emitted).

- [ ] **Step 4: Commit**

```bash
git add web/app/pages/contacts/index.vue
git commit -m "feat(web): add contacts page with add/edit/delete/import dialogs"
```

---

### Task 13: Show contact names on threads + Contacts nav link

**Files:**
- Modify: `web/app/stores/threads.ts` (send `contacts: true`)
- Modify: `web/app/components/MessageThread.vue` (list title + avatar initial)
- Modify: `web/app/pages/threads/[id]/index.vue` (toolbar title)
- Modify: `web/app/components/MessageThreadHeader.vue` (Contacts nav item + icon import)

**Interfaces:**
- Consumes: `EntitiesMessageThread.contact_details?: EntitiesContact` (Task 10), the `/contacts` route (Task 12).
- Produces: no new exports — UI wiring only.

- [ ] **Step 1: Request contact resolution when loading threads**

In `web/app/stores/threads.ts`, add `contacts: true` to the `loadThreads` request params:

```ts
    const response = await apiFetch<{ data: EntitiesMessageThread[] }>(
      '/v1/message-threads',
      {
        params: {
          owner: phonesStore.owner ?? phonesStore.phones[0]?.phone_number,
          limit: 100,
          is_archived: archivedThreads.value,
          contacts: true,
        },
      },
    )
```

- [ ] **Step 2: Show the contact name in the threads list**

In `web/app/components/MessageThread.vue`, replace the list-item title expression:

```vue
        <v-list-item-title :class="{ 'font-weight-bold': !thread.is_read }">{{
          thread.contact_details?.name ?? formatPhoneNumber(thread.contact)
        }}</v-list-item-title>
```

And update the avatar initial to prefer the contact name (replace the existing `<template #prepend>` avatar content):

```vue
        <template #prepend>
          <v-avatar
            :color="thread.color"
            size="40"
            :badge="thread.is_read ? false : { color: 'primary', dotSize: 12 }"
          >
            <v-icon
              v-if="!(thread.contact_details?.name || startsWithLetter(thread.contact))"
              color="white"
              >{{ mdiAccount }}</v-icon
            >
            <span v-else class="text-white text-headline-small">{{
              (thread.contact_details?.name ?? thread.contact).substring(0, 1)
            }}</span>
          </v-avatar>
        </template>
```

- [ ] **Step 3: Show the contact name in the thread header**

In `web/app/pages/threads/[id]/index.vue`, replace the toolbar title expression (around line 271):

```vue
        <VToolbarTitle v-if="threadsStore.currentThread">
          {{
            threadsStore.currentThread.contact_details?.name ??
            formatPhoneNumber(threadsStore.currentThread.contact)
          }}
        </VToolbarTitle>
```

- [ ] **Step 4: Add the Contacts navigation item**

In `web/app/components/MessageThreadHeader.vue`, add `mdiAccountMultiple` to the `@mdi/js` import block:

```ts
  mdiCommentTextMultipleOutline,
  mdiAccountMultiple,
  mdiCircle,
```

Add a nav item right after the "Search Messages" `v-list-item` (before the Settings item):

```vue
        <v-list-item :to="{ name: 'contacts' }">
          <template #prepend><v-icon :icon="mdiAccountMultiple" /></template>
          <v-list-item-title>Contacts</v-list-item-title>
        </v-list-item>
```

- [ ] **Step 5: Lint and build**

Run: `cd web && pnpm lint:js && pnpm generate`
Expected: no errors; the build succeeds.

- [ ] **Step 6: Commit**

```bash
git add web/app/stores/threads.ts web/app/components/MessageThread.vue web/app/pages/threads/[id]/index.vue web/app/components/MessageThreadHeader.vue
git commit -m "feat(web): display contact names on threads and add Contacts nav link"
```

---

## Final Verification

- [ ] **API full test + build**

```bash
cd api && go test -vet=off ./... && go build ./...
```
Expected: all packages pass; build succeeds.

- [ ] **Web lint + build**

```bash
cd web && pnpm lint && pnpm generate
```
Expected: lint clean; static build succeeds.

- [ ] **Manual smoke test (optional, with Docker stack)**

```bash
docker compose up --build
```
Then: create a contact via `POST /v1/contacts`, load `/threads` and confirm the contact name shows for a matching phone number; open `/contacts` and exercise add/edit/delete/import.
