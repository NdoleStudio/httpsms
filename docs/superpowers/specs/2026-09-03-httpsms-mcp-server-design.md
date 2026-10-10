# httpSMS MCP Server Design

## Summary

Add a separately deployed Model Context Protocol server for httpSMS at
`https://mcp.httpsms.com/mcp`. The service will use the official
`github.com/modelcontextprotocol/go-sdk`, implement MCP `2026-07-28`, and
temporarily support `2025-11-25` clients during migration.

The server will expose a curated set of tools for sending SMS messages, reading
phones and message history, creating phone API keys, and rotating the user's
primary API key. It will call the existing httpSMS HTTP API for all product
operations and will not access the httpSMS database directly.

## Goals

- Provide a production-ready remote MCP endpoint at `/mcp`.
- Reuse existing Firebase accounts for interactive user login.
- Implement MCP-compliant OAuth 2.1 authorization with resource-specific
  access tokens.
- Keep MCP protocol requests stateless and horizontally scalable.
- Call the existing httpSMS API through a scoped service-to-service identity.
- Preserve current API validation, entitlement, rate-limit, and encryption
  behavior.
- Add end-to-end integration coverage under the repository's `tests/` module.
- Deploy the service independently through the repository's existing Cloud
  Build and Cloud Run pattern.

## Non-Goals

- Generating MCP tools automatically from the complete Swagger document.
- Exposing every httpSMS API endpoint in the first release.
- Reading or writing the httpSMS database from the MCP service.
- Decrypting end-to-end encrypted SMS content.
- Replacing Firebase Authentication or existing API-key authentication.
- Removing the CAPTCHA requirement from the general message search endpoint.
- Supporting deprecated HTTP+SSE as a new transport.

## Repository and Deployment Boundary

Implementation will be developed on branch `feat/mcp-server` in the
`.worktrees/mcp-server` worktree created from `main`.

A new top-level `mcp/` Go module will contain:

- the HTTP server entry point;
- MCP server and tool registration;
- OAuth authorization-server and protected-resource endpoints;
- JWT signing, validation, and JWKS publication;
- Redis-backed authorization and confirmation state;
- a typed httpSMS API client;
- configuration, telemetry, and HTTP middleware;
- unit and component tests;
- a Dockerfile, README, and `cloudbuild.yaml`.

The MCP service will be deployed as a separate Cloud Run service in `us-east1`.
Cloud Build will build and publish its container and deploy it with the Cloud
Run port supplied through `PORT`. The Cloud Run service will allow public
network access because OAuth and MCP discovery endpoints must be reachable;
application-layer authorization will protect every tool call.

Mapping `mcp.httpsms.com` to the Cloud Run service and creating its DNS record
are one-time infrastructure steps documented in `mcp/README.md`, not repeated
on every build.

## Protocol

The implementation will pin a released official Go SDK version that supports
MCP `2026-07-28`; the currently verified release is `v1.7.0`.

The primary transport is Streamable HTTP at:

```text
https://mcp.httpsms.com/mcp
```

For `2026-07-28`:

- the MCP handler will use the stateless protocol core;
- requests will not depend on `initialize`, `initialized`, or
  `Mcp-Session-Id`;
- `server/discover` will advertise server identity, supported versions, and
  capabilities;
- the SDK will validate per-request protocol metadata;
- `Mcp-Method` and `Mcp-Name` headers will be supported as required by the
  transport;
- tool, resource, and discovery responses will use deterministic ordering and
  SDK-supported cache hints where applicable.

The server will also accept `2025-11-25` through the SDK's legacy negotiation
path. It will not enable deprecated HTTP+SSE. Legacy support is a compatibility
window, not a dependency for new features.

The first release exposes tools only. It does not add prompts, resources,
sampling, roots, or logging capabilities.

## OAuth and Firebase Identity

### Token Roles

Three distinct token types prevent audience confusion:

1. A Firebase ID token proves the user's identity during browser login.
2. An MCP access token authorizes calls to `mcp.httpsms.com`.
3. A downstream API JWT authorizes the MCP service to call
   `api.httpsms.com` for that user.

A Firebase ID token will not be accepted directly as an MCP access token. Its
audience is the Firebase project, not the MCP protected resource, so direct use
would not satisfy MCP's audience-bound access-token requirements.

### Discovery Endpoints

The service will expose:

- OAuth protected-resource metadata for the MCP endpoint;
- OAuth authorization-server metadata;
- a JWKS endpoint for MCP access-token verification and API delegation;
- an authorization endpoint;
- a token endpoint;
- a legacy dynamic-client-registration endpoint for compatible older clients.

Client ID Metadata Documents are the preferred client identity mechanism.
Dynamic Client Registration is supported only for compatibility and will be
isolated behind the same redirect-URI and metadata validation rules.

### Authorization Code Flow

The authorization flow will:

1. validate client metadata, redirect URI, requested scopes, state, and PKCE
   challenge;
2. create a short-lived authorization transaction in Redis;
3. render a browser page using existing Firebase Authentication providers;
4. verify the resulting Firebase ID token server-side;
5. display the scopes requested by the MCP client;
6. issue a random, one-time, PKCE-bound authorization code;
7. exchange the code at the token endpoint after exact redirect-URI and PKCE
   verification.

MCP access tokens will be short-lived asymmetric JWTs with explicit issuer,
audience, subject, client, scope, issued-at, expiry, and key ID claims.

Refresh tokens will be high-entropy opaque values. Only hashes will be stored
in Redis, bound to the user, client, granted scopes, and token family. Refresh
rotation will invalidate the previous value. Authorization codes, transaction
records, registration records, and refresh-token records will have explicit
TTLs.

### Client Metadata Security

Client metadata retrieval will:

- require HTTPS outside explicitly configured local test environments;
- reject private, loopback, link-local, and otherwise non-public targets;
- limit response size and request duration;
- validate content type and required metadata fields;
- reject unsafe redirects;
- cache validated metadata for a bounded period.

These controls prevent the authorization server from becoming an SSRF proxy.

## Delegated MCP-to-API Authentication

The MCP server will not store users' primary httpSMS API keys or Firebase
refresh tokens.

After validating an MCP access token and tool scope, the service will mint a
separate short-lived JWT for the API. The token will contain:

- issuer identifying the MCP service;
- audience identifying `api.httpsms.com`;
- Firebase UID as the subject;
- only the downstream scopes needed by the current tool;
- short issued-at, not-before, and expiry windows;
- a unique token ID and signing key ID.

The API will add an MCP delegation authentication middleware. It will:

- accept only the configured MCP issuer;
- fetch and cache the MCP JWKS;
- validate signature, key ID, audience, issuer, time claims, and scopes;
- load the existing user authentication context from the Firebase UID;
- reject malformed or over-scoped tokens;
- leave existing Firebase bearer and `x-api-key` behavior unchanged.

The delegated identity is valid only for existing authenticated user routes. It
does not grant phone API-key privileges or administrative access implicitly.

## OAuth Scopes

The initial authorization scopes are:

| Scope | Purpose |
| --- | --- |
| `phones:read` | List the user's registered phones and sending numbers. |
| `messages:read` | List threads, thread messages, and incoming messages. |
| `messages:send` | Queue an SMS message for sending. |
| `phone-api-keys:write` | Create a phone API key. |
| `user-api-key:rotate` | Rotate the user's primary API key. |

The authorization page will display requested scopes in user-facing language.
Each MCP tool will require its corresponding MCP scope and mint only the
matching downstream API scope.

## Tool Catalog

### `list_phones`

Calls `GET /v1/phones`.

Inputs include bounded pagination and an optional query. The result contains
the registered phone records needed to select a valid sending number and SIM.

Required scope: `phones:read`.

### `send_sms`

Calls `POST /v1/messages/send`.

Inputs:

- `from`;
- `to`;
- `content`;
- optional `sim`;
- optional `request_id`;
- optional `encrypted`;
- optional attachments supported by the API.

The tool preserves API validation, billing entitlement, scheduling, and
delivery behavior. It will not automatically retry unless the request includes
an idempotency value that makes the retry safe.

Required scope: `messages:send`.

### `list_message_threads`

Calls `GET /v1/message-threads`.

Inputs:

- owner phone number;
- optional archive filter;
- optional contact enrichment;
- optional text query;
- bounded `skip` and `limit`.

Required scope: `messages:read`.

### `list_thread_messages`

Calls `GET /v1/messages`.

Inputs:

- owner phone number;
- contact phone number;
- optional text query;
- bounded `skip` and `limit`.

Required scope: `messages:read`.

### `list_incoming_messages`

Calls a new API endpoint, `GET /v1/messages/incoming`.

The existing `GET /v1/messages/search` route requires Cloudflare Turnstile and
is not suitable for server-to-server calls. The new endpoint will use normal
user authentication and the MCP delegated JWT path. It will expose:

- optional owner filters;
- optional received status filters supported by the use case;
- optional text query;
- bounded pagination;
- supported sort inputs.

The handler will reuse `MessageService.SearchMessages` while forcing the
message type to `mobile-originated`. It will not weaken or bypass CAPTCHA on
the general search route. Missed calls are excluded from the initial tool.

Required scope: `messages:read`.

### `create_phone_api_key`

Calls `POST /v1/phone-api-keys`.

Input: the key name.

The result includes the newly created phone API key as a sensitive, one-time
display value. The value must never be logged or added to telemetry.

Required scope: `phone-api-keys:write`.

### `rotate_user_api_key`

Calls `DELETE /v1/users/{authenticated-user}/api-keys`.

The user ID is derived from the authenticated subject and is never accepted as
a tool argument. The operation invalidates the current primary API key and
returns its replacement as a sensitive, one-time display value.

For `2026-07-28`, the tool will use Multi Round-Trip Requests and return
`input_required` before rotation. For legacy clients, the first call will
return a short-lived random confirmation handle stored as a hash in Redis; a
second call must present that handle. Handles are user-, client-, operation-,
and expiry-bound and are consumed once.

Required scope: `user-api-key:rotate`.

## API Changes

The API changes are intentionally narrow:

1. Add configuration for the MCP delegated JWT issuer, audience, JWKS URL, and
   permitted scopes.
2. Add middleware that validates delegated MCP API JWTs and loads the existing
   authentication context.
3. Add a request model and validator for incoming-message filters.
4. Add `GET /v1/messages/incoming`.
5. Reuse `MessageService.SearchMessages` with a fixed
   `mobile-originated` type.
6. Register the route through the existing dependency-injection container.
7. Add handler, middleware, service-boundary, and integration tests.
8. Regenerate Swagger documentation after changing annotations.

No raw SQL or new direct database access is required.

## API Client and Result Mapping

The `mcp/` module will contain a typed client only for endpoints used by the
tool catalog. The client will:

- use `context.Context` deadlines and cancellation;
- apply bounded connection, header, and overall request timeouts;
- send the downstream delegated JWT;
- propagate a request ID;
- set explicit content types;
- enforce response-size limits;
- decode the standard httpSMS response envelope and error envelope;
- close response bodies on every path.

MCP tools will return structured content with stable field names. Upstream
errors remain distinguishable:

- invalid tool input;
- API field validation;
- unauthenticated or insufficient scope;
- payment or entitlement failure;
- not found;
- rate limited;
- API unavailable or timed out;
- unexpected API response.

The server will not convert failures into success-shaped empty results.

## Sensitive Data and Encryption

The MCP service will not receive or store the user's SMS encryption key.
Encrypted message content will be returned exactly as stored by the API.

The following values must be redacted from logs, traces, metrics, and error
messages:

- MCP and downstream bearer tokens;
- Firebase ID tokens;
- authorization codes and refresh tokens;
- PKCE verifiers;
- primary and phone API keys;
- SMS content and attachment payloads.

Tool results that contain a newly created or rotated key will identify it as a
sensitive one-time value. The values will not be cached by the MCP service.

## Rate Limiting and Reliability

Redis-backed limits will apply by authenticated user and tool. They complement,
rather than replace, API-side entitlement and sending limits.

The MCP service will not retry non-idempotent calls such as SMS sending, phone
API-key creation, or primary API-key rotation by default. Read-only calls may
use a small bounded retry for connection failures and retryable upstream
statuses while respecting request deadlines.

The service will expose a lightweight unauthenticated health endpoint for
Cloud Run checks. Health will report process readiness without disclosing
dependency details. OAuth and MCP handlers will fail explicitly when Redis,
key material, or the API is unavailable.

## Observability

The MCP service will follow the repository's OpenTelemetry and structured
logging conventions. Traces will cover:

- OAuth authorization and token exchange;
- MCP request parsing and authorization;
- tool execution;
- downstream API calls;
- Redis grant and confirmation operations.

Allowed telemetry attributes include tool name, protocol version, user ID,
OAuth client ID, scope set, request ID, status, and latency. Sensitive values
listed above are prohibited.

## Testing

### MCP Module Tests

Unit and component tests under `mcp/` will cover:

- configuration validation;
- Firebase identity-token verification with configurable test issuer/JWKS;
- JWT signing, JWKS publication, rotation, audience, issuer, and time claims;
- PKCE verification and exact redirect-URI matching;
- one-time authorization code use;
- refresh-token rotation and replay rejection;
- scope enforcement;
- CIMD validation and SSRF protections;
- DCR compatibility;
- Redis-backed confirmation handles;
- API request construction and response/error mapping;
- every tool handler;
- secret redaction.

Tests will use `httptest` and an isolated Redis test dependency already
available through the integration stack where persistence semantics matter.

### End-to-End Integration Tests

The repository's `tests/docker-compose.yml` will add the MCP service and the
test identity/JWKS endpoints needed to issue deterministic Firebase-style
identity tokens. Integration tests under `/tests` will cover:

1. protected-resource and authorization-server metadata;
2. unauthenticated MCP rejection with `WWW-Authenticate`;
3. OAuth authorization-code exchange with PKCE;
4. invalid issuer, audience, redirect URI, code replay, and insufficient scope;
5. MCP `2026-07-28` discovery, tool listing, and tool calls;
6. MCP `2025-11-25` initialization and tool calls;
7. listing phones;
8. listing message threads and thread messages;
9. listing incoming messages through `/v1/messages/incoming`;
10. sending an SMS through MCP and observing delivery through the existing
    phone emulator;
11. creating a phone API key;
12. refusing unconfirmed primary API-key rotation;
13. completing confirmed rotation and proving the previous key is invalid.

The existing integration-test GitHub Actions workflow will build the MCP
container and run these tests with the rest of the stack.

## Deployment Configuration

`mcp/cloudbuild.yaml` will:

1. build the MCP Docker image;
2. publish commit-specific and `latest` tags;
3. deploy the dedicated Cloud Run service in `us-east1`;
4. configure the service port;
5. inject only secret references and non-sensitive environment configuration;
6. leave the service publicly reachable for protocol and OAuth discovery.

Signing keys, Firebase credentials, Redis credentials, and other secrets will
come from Google Secret Manager or Cloud Run secret references, not committed
files or plain-text Cloud Build substitutions.

## Rollout

1. Deploy API support for delegated MCP JWTs and the incoming-message endpoint.
2. Deploy the MCP Cloud Run service with a temporary Cloud Run URL.
3. Run protocol, OAuth, tool, and full integration tests against the deployed
   services.
4. Map `mcp.httpsms.com` and publish DNS.
5. Verify discovery metadata uses the final HTTPS issuer and resource URLs.
6. Enable access for initial users while monitoring authentication failures,
   tool errors, rate limits, and API latency.
7. Remove `2025-11-25` support in a later change after client usage confirms it
   is no longer needed.

## Acceptance Criteria

- `https://mcp.httpsms.com/mcp` serves MCP `2026-07-28`.
- `2025-11-25` clients work through the documented compatibility path.
- OAuth uses Firebase for identity and issues MCP audience-bound tokens.
- MCP access tokens cannot be used directly against the API.
- Downstream API JWTs cannot be used against the MCP endpoint.
- Every tool enforces its documented OAuth scope.
- All product operations go through `api.httpsms.com`.
- `/v1/messages/search` remains CAPTCHA-protected.
- `/v1/messages/incoming` returns only authenticated users' mobile-originated
  messages.
- API-key rotation requires user confirmation and never accepts a user ID from
  tool input.
- Secrets and SMS content do not appear in logs or traces.
- Unit and `/tests` integration suites cover both protocol versions and every
  tool.
- Cloud Build deploys the MCP service independently to Cloud Run.
