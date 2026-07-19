# Contacts Feature — Design

Date: 2026-07-19
Status: Approved (pending implementation plan)

## 1. Summary

Add a **Contacts** feature to httpSMS. Users can store contacts (name, emails,
phone numbers, and free-form properties) and manage them through a dedicated web
page and a REST API. Once contacts exist, the threads page displays the
contact's **name** instead of the raw phone number.

Design priority (explicit user goal): resolve contact names into the threads
page with **as few database queries as possible**, and make **changing a
contact's name cheap** (no write amplification across threads).

## 2. Goals

- CRUD for contacts via a REST API under `/v1/contacts`.
- A single create endpoint that accepts **one or many** contacts in one request.
- Import contacts from **CSV** (Excel/XLSX is **not** supported for contacts).
- A contact has: `Name` (required), `Emails` (optional array), `PhoneNumbers`
  (array, >= 1), and free-form `Properties` (`map[string]string`).
- Contacts are **global to the user account** (name shows across all the user's
  owner phones).
- Threads display the resolved contact name (and expose full contact details)
  when requested, everywhere the peer number is shown (threads list + thread
  header/title).
- Dedicated Contacts web page with full CRUD + import.

## 3. Non-goals / Explicit decisions

- **No uniqueness constraint** on phone numbers. Multiple contacts may share a
  phone number. There is **no** normalized `contact_phone_numbers` lookup table
  and **no** DB-level unique index on numbers.
- When a thread's number matches **multiple** contacts, the **most recently
  updated** contact wins for display.
- Thread-name resolution is **opt-in** per request (default off).
- `Properties` are API/JSON only; not part of the flat CSV template.

## 4. Data model

### 4.1 `Contact` entity (`api/pkg/entities/contact.go`)

| Field          | Type                | GORM / notes                                  |
| -------------- | ------------------- | --------------------------------------------- |
| `ID`           | `uuid.UUID`         | `primaryKey;type:uuid`                         |
| `UserID`       | `entities.UserID`   | indexed                                        |
| `Name`         | `string`            | required                                       |
| `Emails`       | `pq.StringArray`    | `gorm:"type:text[]"` `swaggertype:"array,string"`; optional, each validated |
| `PhoneNumbers` | `pq.StringArray`    | `gorm:"type:text[]"` `swaggertype:"array,string"`; >= 1, each valid E.164   |
| `Properties`   | `ContactProperties` | `gorm:"type:jsonb"`; free-form key/value map   |
| `CreatedAt`    | `time.Time`         |                                                |
| `UpdatedAt`    | `time.Time`         |                                                |

Follows the existing `pq.StringArray` convention used in `webhook.go` and
`phone_api_key.go`. Auto-migrated in `pkg/di/container.go` alongside the other
entities.

### 4.2 `ContactProperties` custom type

A small custom type to avoid adding a new dependency (`gorm.io/datatypes` is not
currently used):

```go
// ContactProperties is a free-form key/value map persisted as a jsonb column.
type ContactProperties map[string]string

func (p ContactProperties) Value() (driver.Value, error) // json.Marshal -> []byte
func (p *ContactProperties) Scan(src any) error           // json.Unmarshal from []byte/string
```

Includes unit tests for `Value`/`Scan` round-trips (nil, empty, populated).

### 4.3 `MessageThread` additions (`api/pkg/entities/message_thread.go`)

Add a **non-persisted** field carrying the resolved contact:

```go
ContactDetails *Contact `json:"contact_details,omitempty" gorm:"-"`
```

- `gorm:"-"` — never read/written to the DB.
- `omitempty` — absent from responses unless resolution attached it.
- Named `ContactDetails` (JSON `contact_details`) to avoid colliding with the
  existing `Contact string json:"contact"` field, which holds the peer phone
  number.

## 5. API

New `ContactHandler` mirroring existing handler/service/repository/validator
layering. Routes registered via the `handler.register(...)` helper (Fiber v3),
wired in the DI container. All routes are under `/v1/` and behind the standard
auth middleware chain.

| Method   | Route                    | Purpose                                                        |
| -------- | ------------------------ | ------------------------------------------------------------- |
| `GET`    | `/v1/contacts`           | List (paginated; `?query=` searches name / email / number)    |
| `POST`   | `/v1/contacts`           | Create **one or many** — body is a JSON array of contacts     |
| `POST`   | `/v1/contacts/upload`    | Import from CSV (`document` multipart form field; CSV only)    |
| `PUT`    | `/v1/contacts/:contactID`| Update name / emails / phone numbers / properties             |
| `DELETE` | `/v1/contacts/:contactID`| Delete a contact                                              |

### 5.1 Create (one or many)

Request body (array form or `{ "contacts": [...] }`):

```json
{
  "contacts": [
    {
      "name": "Alice Smith",
      "emails": ["alice@example.com"],
      "phone_numbers": ["+18005550199", "+18005550100"],
      "properties": { "company": "Acme", "role": "CTO" }
    }
  ]
}
```

- Per-item validation with row-indexed error messages (mirroring bulk-message
  validation style using `url.Values`).
- Batch capped at <= 1000 contacts.
- Cache invalidated **once** after the batch commits.

### 5.2 CSV import (CSV only)

- Reuses the bulk-message CSV parsing pattern (`csvutil`; <= 500 KB; <= 1000
  rows). **Excel/XLSX is not supported** — only `text/csv` / `.csv` is accepted;
  any other file type returns a validation error.
- Columns: `Name, Emails, PhoneNumbers`. `Emails` and `PhoneNumbers` cells hold
  multiple values separated by `;` (or `,`). `Properties` is not part of the
  flat template.
- Published template: `httpsms-contacts.csv` (same location/convention as the
  bulk-message template).

### 5.3 Validation rules

- `Name` required and non-empty per contact.
- Each phone number must parse as E.164 (`nyaruka/phonenumbers`); >= 1 required.
- Each email (if present) must be a valid email.
- `properties` keys/values are free-form strings.

## 6. Thread name resolution + caching

### 6.1 Contact map cache

- A per-user map `phone_number -> *Contact` is built from **all** the user's
  contacts. On phone-number collisions, the **most recently updated** contact
  wins (sort by `UpdatedAt` ascending while building so later writes overwrite).
- Serialized as JSON in `cache.Cache` under key `contacts.map.<userID>`,
  TTL ~24h.
- `cache.Cache` exposes only `Get`/`Set` (no delete), so **invalidation =
  overwrite** the key with an empty marker after any contact mutation; the next
  request that needs it lazily rebuilds from the DB.

### 6.2 `GetThreads` flow

1. Load threads (existing single query, unchanged).
2. **If** the request's `Contacts` flag is true: fetch the user's contact map
   (cache hit → 0 DB queries; miss → one query to rebuild), then attach
   `ContactDetails` to each thread in memory by looking up `thread.contact`.
3. If the flag is false/absent: skip resolution entirely (no cache/DB work);
   behavior identical to today.

Cost profile:

- Threads read: **1 DB query** (unchanged).
- Contact resolution: usually a **cache hit → 0 DB queries**; DB touched only on
  cache miss or right after a contact write.
- Contact **name change**: single-row `UPDATE` + cache invalidation.
  **Zero thread writes / zero propagation.**

### 6.3 Opt-in filter

- `GET /v1/message-threads` gains an optional query param `?contacts=true`
  bound onto `requests.MessageThreadIndex` (`Contacts bool`, default `false`).
- Threaded through `MessageThreadGetParams` to the service, controlling whether
  step 2 above runs.

## 7. Frontend (Nuxt 4 / Pinia)

- **`useContactsStore`** (`web/app/stores/contacts.ts`): `loadContacts`,
  `saveContacts` (array), `updateContact`, `deleteContact`, `uploadCsv`.
- **Contacts page** (`web/app/pages/contacts/index.vue`) — layout modeled on the
  provided reference (Plunk-style):
  - **Header row**: large `Contacts` title (`text-display-large`, no glowing
    gradient) with a subtitle like `Manage your contacts. {total} total`, and,
    aligned top-right, an outlined **Import CSV** button and a filled primary
    **Add Contact** button.
  - **Search bar** below the header filtering by name / email / phone number
    (drives the `?query=` API param, debounced).
  - **Data table** (`VDataTable`) with columns: `Name`, `Phone Numbers`,
    `Emails`, `Created`, `Updated`, and a right-aligned **Actions** column with
    per-row **edit** (pencil, `mdiPencil`) and **delete** (trash,
    `mdiDelete`) icon buttons.
  - Timestamps rendered relatively (e.g. "31 minutes ago") via `useFilters()`.
  - Pagination for large lists.
  - **Modals** (all `VDialog` with `opacity="0.9"`, Close button
    `color="warning"`):
    - **Add / Edit Contact** dialog — form for `Name`, repeatable
      `PhoneNumbers`, repeatable `Emails`, and free-form `Properties`
      (key/value rows). Same dialog component for create and edit.
    - **Delete Contact** confirmation dialog.
    - **Import CSV** dialog — file input accepting `.csv` only, a link to the
      `httpsms-contacts.csv` template, and inline row-indexed error display.
  - Hyperlinks use `text-decoration-none hover:text-decoration-underline`.
- **Threads UI**:
  - `threads` store `loadThreads` passes `contacts: true`.
  - `MessageThread.vue` title and `MessageThreadHeader.vue` render
    `thread.contact_details?.name ?? formatPhoneNumber(thread.contact)`; the
    avatar uses the resolved name's first letter when present.
  - Filter helpers pulled from `useFilters()` in `<script setup>` and used
    directly in the template.
- **Types**: regenerate `shared/types/api` from Swagger (`pnpm api:models`)
  after the API annotations land — provides `EntitiesContact` and
  `contact_details` on `EntitiesMessageThread`.
- **Navigation**: add a Contacts link in the default layout.

## 8. Testing

### API (Go) — run with `go test -vet=off ./...`

- `ContactProperties` `Value`/`Scan` round-trip tests.
- Contact repository: create (one/many), update, delete, list + search,
  contact-map build with most-recently-updated tie-break.
- Validator: name required, E.164 numbers, email format, batch cap, CSV
  parsing (valid + malformed); non-CSV upload rejected.
- Handler tests mirroring `message_thread_handler_test.go`.
- Resolution: threads get `ContactDetails` attached only when `contacts=true`;
  cache-hit path performs no DB query; tie-break correctness; unaffected when
  the flag is off.

### Web

- Store unit tests for contacts CRUD/import.
- Thread rendering falls back to formatted number when no contact is resolved.

## 9. Rollout / migration

- `Contact` auto-migrated by GORM (new table). No backfill required — resolution
  is computed at read time, so existing threads immediately benefit once
  contacts are created. No changes to existing thread rows.

## 10. Efficiency summary (why this design)

- Threads list stays **one query**.
- Contact names are resolved from an in-memory per-user **cache**, so the common
  path adds **zero** DB queries.
- Because the name lives only on the contact row, a **name change is a single
  `UPDATE`** plus a cache invalidation — no fan-out writes across threads.
- Resolution is **opt-in**, so callers that don't need names pay nothing.
