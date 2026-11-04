# httpSMS MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and deploy a standards-compliant remote MCP server that authenticates existing httpSMS users with Firebase, calls the httpSMS API, and exposes the approved SMS, message-history, and API-key tools.

**Architecture:** Add an independent `mcp/` Go module using `github.com/modelcontextprotocol/go-sdk` v1.7.0 and stateless Streamable HTTP. The service implements an OAuth 2.1 authorization facade backed by Firebase identity and Redis, issues audience-bound MCP JWTs, mints short-lived delegated API JWTs, and calls only `api.httpsms.com`; the API gains delegated-JWT authentication and a CAPTCHA-free, narrowly scoped incoming-message endpoint.

**Tech Stack:** Go 1.25, official MCP Go SDK v1.7.0, Fiber v3, Firebase Authentication, Redis, `golang-jwt/jwt/v5`, OpenTelemetry, Cloud Build, Cloud Run, Docker Compose, Testify.

**Spec:** `docs/superpowers/specs/2026-09-03-httpsms-mcp-server-design.md`

## Global Constraints

- Develop only in `.worktrees/mcp-server` on branch `feat/mcp-server`.
- Serve MCP at `https://mcp.httpsms.com/mcp`.
- Implement MCP `2026-07-28` and retain `2025-11-25` compatibility.
- Use `github.com/modelcontextprotocol/go-sdk` v1.7.0; do not use `mark3labs/mcp-go`.
- Keep MCP protocol handling stateless; store OAuth grants and confirmation state in Redis.
- Use Firebase ID tokens only as browser-login identity proof, never as MCP access tokens.
- Issue separate audience-bound JWTs for MCP access and downstream API delegation.
- Do not store user Firebase refresh tokens or primary httpSMS API keys.
- Call all product operations through the httpSMS HTTP API; do not access its database from `mcp/`.
- Keep `/v1/messages/search` CAPTCHA-protected.
- Never log SMS content, bearer tokens, authorization codes, refresh tokens, PKCE verifiers, or API-key values.
- Use `stacktrace.Propagate` or `stacktrace.PropagateWithCode` for API errors; never return bare API errors.
- Run `go-fumpt` and `go-imports` through the repository's existing formatting workflow.
- Regenerate Swagger after API annotation changes.

---

## File Map

### Existing API

- `api/pkg/auth/mcp_claims.go`: delegated token claims and scope parsing.
- `api/pkg/auth/mcp_jwks.go`: bounded JWKS fetch/cache and RSA key selection.
- `api/pkg/auth/mcp_token_verifier.go`: issuer, audience, signature, time, and scope validation.
- `api/pkg/middlewares/mcp_delegation_auth_middleware.go`: load the delegated Firebase UID into `AuthContext`.
- `api/pkg/middlewares/bearer_auth_middleware.go`: skip Firebase verification when an earlier middleware authenticated the request and stop logging raw tokens.
- `api/pkg/requests/message_incoming_request.go`: request model and conversion to fixed-type search params.
- `api/pkg/validators/message_handler_validator.go`: incoming-message filter validation without Turnstile.
- `api/pkg/handlers/message_handler.go`: register and implement `GET /v1/messages/incoming`.
- `api/pkg/di/container.go`: construct and order delegated authentication middleware.
- `api/docs/docs.go`, `api/docs/swagger.json`, `api/docs/swagger.yaml`: regenerated API documentation.

### MCP Module

- `mcp/go.mod`, `mcp/go.sum`: isolated Go module and pinned dependencies.
- `mcp/cmd/server/main.go`: load config, construct dependencies, serve HTTP, and shut down.
- `mcp/internal/config/config.go`: validated environment configuration.
- `mcp/internal/observability/observability.go`: structured logging and OpenTelemetry setup.
- `mcp/internal/auth/claims.go`: MCP and API JWT claims, principals, scopes, and context helpers.
- `mcp/internal/auth/keys.go`: RSA private-key loading, signing, and JWKS publication.
- `mcp/internal/auth/firebase.go`: Firebase identity-token verification against the configured certificate endpoint.
- `mcp/internal/auth/middleware.go`: official SDK bearer middleware adapter and per-tool scope checks.
- `mcp/internal/oauth/store.go`: Redis records for transactions, codes, refresh tokens, DCR clients, and confirmations.
- `mcp/internal/oauth/metadata.go`: protected-resource, authorization-server, and JWKS HTTP handlers.
- `mcp/internal/oauth/clients.go`: CIMD retrieval/validation and DCR compatibility.
- `mcp/internal/oauth/authorize.go`: authorization request, Firebase completion, consent, and code issuance.
- `mcp/internal/oauth/token.go`: authorization-code and refresh-token grants.
- `mcp/internal/oauth/templates/authorize.html`: Firebase login and scope-consent page.
- `mcp/internal/httpsms/client.go`: typed downstream HTTP client and standard error decoding.
- `mcp/internal/httpsms/models.go`: request/response models used by approved tools.
- `mcp/internal/tools/phones.go`: `list_phones`.
- `mcp/internal/tools/messages.go`: `send_sms`, `list_message_threads`, `list_thread_messages`, and `list_incoming_messages`.
- `mcp/internal/tools/api_keys.go`: `create_phone_api_key` and `rotate_user_api_key`.
- `mcp/internal/tools/register.go`: deterministic tool registration.
- `mcp/internal/server/rate_limit.go`: Redis-backed per-user/per-tool limits.
- `mcp/internal/server/server.go`: route assembly, MCP handler, health endpoint, and middleware chain.
- `mcp/Dockerfile`, `mcp/cloudbuild.yaml`, `mcp/README.md`: build, deployment, and operations.

### Integration Suite

- `tests/mcp_helpers_test.go`: OAuth, MCP client, PKCE, and test-token helpers.
- `tests/mcp_integration_test.go`: metadata, protocol, tool, scope, and confirmation tests.
- `tests/docker-compose.yml`: MCP container and test identity/certificate configuration.
- `tests/.env.test`: delegated-auth and MCP test configuration.
- `tests/generate-firebase-credentials.sh`: also generate the throwaway MCP/Firebase test key and WireMock certificate mapping.
- `tests/.gitignore`: exclude generated test keys, certificates, and mappings.
- `.github/workflows/api.yml`: wait for MCP and run the expanded integration suite before deployment.

---

### Task 1: Add delegated MCP JWT authentication to the API

**Files:**
- Create: `api/pkg/auth/mcp_claims.go`
- Create: `api/pkg/auth/mcp_jwks.go`
- Create: `api/pkg/auth/mcp_token_verifier.go`
- Create: `api/pkg/auth/mcp_token_verifier_test.go`
- Create: `api/pkg/middlewares/mcp_delegation_auth_middleware.go`
- Create: `api/pkg/middlewares/mcp_delegation_auth_middleware_test.go`
- Modify: `api/pkg/middlewares/bearer_auth_middleware.go:15-49`
- Modify: `api/pkg/di/container.go:100-225`
- Modify: `api/go.mod`
- Modify: `api/go.sum`

**Interfaces:**
- Produces: `auth.NewMCPTokenVerifier(config MCPTokenVerifierConfig) (*MCPTokenVerifier, error)`.
- Produces: `(*MCPTokenVerifier).VerifyRequest(ctx context.Context, raw, method, path string) (*MCPClaims, error)`.
- Produces: `middlewares.MCPDelegationAuth(logger telemetry.Logger, tracer telemetry.Tracer, verifier MCPTokenVerifier, users repositories.UserRepository) fiber.Handler`.
- Consumes later: API requests with `Authorization: Bearer <delegated-api-jwt>`.

- [ ] **Step 1: Write failing verifier tests**

```go
func TestMCPTokenVerifierVerify(t *testing.T) {
	privateKey, publicKey := testRSAKey(t)
	jwks := httptest.NewServer(testJWKSHandler(t, "test-key", publicKey))
	defer jwks.Close()

	verifier, err := NewMCPTokenVerifier(MCPTokenVerifierConfig{
		Issuer:   "https://mcp.httpsms.com",
		Audience: "https://api.httpsms.com",
		JWKSURL:  jwks.URL,
		HTTPClient: jwks.Client(),
	})
	require.NoError(t, err)

	raw := signDelegatedToken(t, privateKey, "test-key", MCPClaims{
		Scopes: []string{"messages:read"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://mcp.httpsms.com",
			Subject:   "firebase-user-id",
			Audience:  jwt.ClaimStrings{"https://api.httpsms.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})

	claims, err := verifier.Verify(context.Background(), raw, "messages:read")
	require.NoError(t, err)
	assert.Equal(t, "firebase-user-id", claims.Subject)
}
```

Add table cases for wrong issuer, wrong audience, expired token, unknown `kid`,
missing subject, and missing required scope.

- [ ] **Step 2: Run the verifier test and confirm failure**

Run: `cd api && go test ./pkg/auth -run TestMCPTokenVerifierVerify -count=1`

Expected: FAIL because `NewMCPTokenVerifier`, `MCPClaims`, and `Verify` do not exist.

- [ ] **Step 3: Implement claims, JWKS caching, and verification**

```go
type MCPClaims struct {
	Scopes []string `json:"scopes"`
	Method string   `json:"http_method"`
	Path   string   `json:"http_path"`
	jwt.RegisteredClaims
}

type MCPTokenVerifierConfig struct {
	Issuer     string
	Audience   string
	JWKSURL    string
	HTTPClient *http.Client
	CacheTTL   time.Duration
}

func (v *MCPTokenVerifier) VerifyRequest(
	ctx context.Context,
	raw string,
	method string,
	path string,
) (*MCPClaims, error) {
	claims := new(MCPClaims)
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		v.keyfunc(ctx),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	if err != nil {
		return nil, stacktrace.PropagateWithCodef(err, ErrCodeInvalidToken, "invalid MCP delegated token")
	}
	if !token.Valid || claims.Subject == "" {
		return nil, stacktrace.NewErrorWithCodef(ErrCodeInvalidToken, "invalid MCP delegated token")
	}
	requiredScope, ok := requiredMCPDelegatedScope(method, path)
	if !ok || claims.Method != method || claims.Path != path {
		return nil, stacktrace.NewErrorWithCodef(ErrCodeInvalidToken, "MCP delegated token is not valid for this API operation")
	}
	if !containsAllScopes(claims.Scopes, []string{requiredScope}) {
		return nil, stacktrace.NewErrorWithCodef(ErrCodeInsufficientScope, "MCP delegated token has insufficient scope")
	}
	return claims, nil
}
```

Implement `requiredMCPDelegatedScope` for only the approved method/path pairs:
phones read, SMS send, message/thread reads, incoming-message reads, phone
API-key creation, and primary API-key rotation. Implement the JWKS loader with
a 2-second HTTP timeout, 1 MiB response limit, RSA-only keys, `kid` lookup, and
a 15-minute default cache. Refresh once when a requested `kid` is absent, then
fail closed.

- [ ] **Step 4: Run verifier tests**

Run: `cd api && go test ./pkg/auth -count=1`

Expected: PASS.

- [ ] **Step 5: Write failing middleware tests**

```go
func TestMCPDelegationAuthSetsAuthContext(t *testing.T) {
	app := fiber.New()
	app.Use(MCPDelegationAuth(logger, tracer, verifierStub{
		claims: &auth.MCPClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "user-id"}},
	}, userRepositoryStub{
		user: &entities.User{ID: "user-id", Email: "user@example.com"},
	}))
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(c.Locals(ContextKeyAuthUserID))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer delegated-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
```

Add cases for non-bearer requests passing through, invalid delegated tokens
passing through for existing auth middleware, unknown users, and no raw token in
logs.

- [ ] **Step 6: Run middleware tests and confirm failure**

Run: `cd api && go test ./pkg/middlewares -run 'TestMCPDelegationAuth|TestBearerAuth' -count=1`

Expected: FAIL because the middleware is not implemented and `BearerAuth`
still attempts Firebase verification after delegated authentication.

- [ ] **Step 7: Implement middleware ordering and bearer short-circuit**

```go
func MCPDelegationAuth(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	verifier interface {
		VerifyRequest(context.Context, string, string, string) (*auth.MCPClaims, error)
	},
	users repositories.UserRepository,
) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := bearerToken(c.Get(authHeaderBearer))
		if raw == "" {
			return c.Next()
		}
		claims, err := verifier.VerifyRequest(c.Context(), raw, c.Method(), c.Path())
		if err != nil {
			if stacktrace.GetCode(err) == auth.ErrCodeInsufficientScope ||
				stacktrace.GetCode(err) == auth.ErrCodeOperationDenied {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"status": "error", "message": "MCP token cannot access this API operation",
				})
			}
			return c.Next()
		}
		user, err := users.Load(c.Context(), entities.UserID(claims.Subject))
		if err != nil {
			return c.Next()
		}
		c.Locals(ContextKeyAuthUserID, entities.AuthContext{ID: user.ID, Email: user.Email})
		return c.Next()
	}
}
```

At the top of `BearerAuth`, return `c.Next()` when
`ContextKeyAuthUserID` already contains a non-noop `entities.AuthContext`.
Replace the existing invalid-token log message so it never interpolates
`authToken`. A cryptographically valid MCP token with the wrong scope or
method/path binding must return 403 directly; malformed or non-MCP bearer
tokens continue to the existing Firebase middleware.

Construct the verifier from `MCP_AUTH_ISSUER`, `MCP_AUTH_AUDIENCE`, and
`MCP_AUTH_JWKS_URL`. Register `MCPDelegationAuth` before `BearerAuth` in
`Container.App()`. Disable it only when all three values are empty; reject
partially configured values during container construction.

- [ ] **Step 8: Run targeted API tests**

Run: `cd api && go test ./pkg/auth ./pkg/middlewares ./pkg/di -count=1`

Expected: PASS.

- [ ] **Step 9: Commit delegated API authentication**

```bash
git add api/go.mod api/go.sum api/pkg/auth api/pkg/middlewares api/pkg/di/container.go
git commit -m "feat(api): trust scoped MCP tokens"
```

---

### Task 2: Add the incoming-message API endpoint

**Files:**
- Create: `api/pkg/requests/message_incoming_request.go`
- Create: `api/pkg/requests/message_incoming_request_test.go`
- Modify: `api/pkg/validators/message_handler_validator.go:284-340`
- Modify: `api/pkg/validators/message_handler_validator_test.go`
- Modify: `api/pkg/handlers/message_handler.go:51-58,507-550`
- Create: `api/pkg/handlers/message_handler_incoming_test.go`
- Modify: `api/docs/docs.go`
- Modify: `api/docs/swagger.json`
- Modify: `api/docs/swagger.yaml`

**Interfaces:**
- Produces: `GET /v1/messages/incoming`.
- Produces: `requests.MessageIncoming.ToSearchParams(userID entities.UserID) *services.MessageSearchParams`.
- Consumes: existing `MessageService.SearchMessages`.

- [ ] **Step 1: Write failing request conversion tests**

```go
func TestMessageIncomingToSearchParamsForcesMobileOriginated(t *testing.T) {
	request := MessageIncoming{
		Owners:         []string{"+18005550199"},
		Statuses:       []string{"received"},
		SortBy:         "created_at",
		SortDescending: true,
		Limit:          "25",
	}

	params := request.Sanitize().ToSearchParams(entities.UserID("user-id"))

	assert.Equal(t, []entities.MessageType{entities.MessageTypeMobileOriginated}, params.Types)
	assert.Equal(t, []entities.MessageStatus{entities.MessageStatusReceived}, params.Statuses)
	assert.Equal(t, 25, params.Limit)
}
```

- [ ] **Step 2: Run the request test and confirm failure**

Run: `cd api && go test ./pkg/requests -run TestMessageIncoming -count=1`

Expected: FAIL because `MessageIncoming` does not exist.

- [ ] **Step 3: Implement the request model**

```go
type MessageIncoming struct {
	request
	Skip           string   `json:"skip" query:"skip"`
	Owners         []string `json:"owners" query:"owners"`
	Statuses       []string `json:"statuses" query:"statuses"`
	Query          string   `json:"query" query:"query"`
	SortBy         string   `json:"sort_by" query:"sort_by"`
	SortDescending bool     `json:"sort_descending" query:"sort_descending"`
	Limit          string   `json:"limit" query:"limit"`
}

func (input MessageIncoming) ToSearchParams(userID entities.UserID) *services.MessageSearchParams {
	statuses := make([]entities.MessageStatus, 0, len(input.Statuses))
	for _, status := range input.Statuses {
		statuses = append(statuses, entities.MessageStatus(status))
	}
	return &services.MessageSearchParams{
		IndexParams: repositories.IndexParams{
			Skip: input.getInt(input.Skip), Query: input.Query,
			SortBy: input.SortBy, SortDescending: input.SortDescending,
			Limit: input.getInt(input.Limit),
		},
		UserID: userID,
		Owners: input.Owners,
		Types: []entities.MessageType{entities.MessageTypeMobileOriginated},
		Statuses: statuses,
	}
}
```

Use defaults `skip=0`, `limit=100`, `sort_by=created_at`, and
`sort_descending=true`.

- [ ] **Step 4: Write failing validator and handler tests**

The validator test must prove no Turnstile token is requested. The handler test
must capture repository search arguments and assert the fixed message type:

```go
require.Equal(t,
	[]entities.MessageType{entities.MessageTypeMobileOriginated},
	repository.searchTypes,
)
require.Equal(t, entities.UserID("user-id"), repository.searchUserID)
```

- [ ] **Step 5: Run validator and handler tests and confirm failure**

Run: `cd api && go test ./pkg/validators ./pkg/handlers -run 'MessageIncoming|Incoming' -count=1`

Expected: FAIL because the validator, route, and handler are missing.

- [ ] **Step 6: Implement validation and handler**

Add `ValidateMessageIncoming` with:

```go
"owners":   {multipleContactPhoneNumberRule},
"statuses": {multipleInRule + ":" + entities.MessageStatusReceived},
"sort_by":  {"in:created_at,owner,contact,status"},
"limit":    {"required", "numeric", "min:1", "max:200"},
"skip":     {"required", "numeric", "min:0"},
"query":    {"max:50"},
```

Register the route before `/:messageID`:

```go
h.register(router, fiber.MethodGet, "/v1/messages/incoming", middlewares, h.Incoming)
```

Implement `Incoming` by binding the query, sanitizing, validating with
`ValidateMessageIncoming`, calling `SearchMessages`, and returning the existing
standard response envelope. Add Swagger annotations documenting that only
mobile-originated messages are returned.

- [ ] **Step 7: Run endpoint tests**

Run: `cd api && go test ./pkg/requests ./pkg/validators ./pkg/handlers -run 'MessageIncoming|Incoming' -count=1`

Expected: PASS.

- [ ] **Step 8: Regenerate and verify Swagger**

Run:

```bash
cd api
swag init --requiredByDefault --parseDependency --parseInternal
grep -n '"/messages/incoming"' docs/swagger.json
```

Expected: Swagger generation succeeds and the route is present.

- [ ] **Step 9: Commit the incoming-message endpoint**

```bash
git add api/pkg/requests api/pkg/validators api/pkg/handlers api/docs
git commit -m "feat(api): add incoming message endpoint"
```

---

### Task 3: Bootstrap the MCP module, configuration, and key handling

**Files:**
- Create: `mcp/go.mod`
- Create: `mcp/go.sum`
- Create: `mcp/internal/config/config.go`
- Create: `mcp/internal/config/config_test.go`
- Create: `mcp/internal/auth/claims.go`
- Create: `mcp/internal/auth/keys.go`
- Create: `mcp/internal/auth/keys_test.go`
- Create: `mcp/internal/observability/observability.go`

**Interfaces:**
- Produces: `config.Load() (config.Config, error)`.
- Produces: `auth.NewKeySet(privateKeyPEM []byte, keyID string) (*auth.KeySet, error)`.
- Produces: `(*auth.KeySet).SignMCPAccessToken(principal auth.Principal, clientID string, scopes []string, ttl time.Duration) (string, error)`.
- Produces: `(*auth.KeySet).SignAPIDelegationToken(principal auth.Principal, scopes []string, method, path string, ttl time.Duration) (string, error)`.
- Produces: `(*auth.KeySet).JWKS() auth.JWKS`.

- [ ] **Step 1: Create the module and pin dependencies**

```go
module github.com/NdoleStudio/httpsms/mcp

go 1.25.0

require (
	firebase.google.com/go v3.13.0+incompatible
	github.com/modelcontextprotocol/go-sdk v1.7.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/redis/go-redis/v9 v9.21.0
	github.com/rs/zerolog v1.35.1
	github.com/stretchr/testify v1.12.1
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.71.0
	go.opentelemetry.io/otel v1.46.0
)
```

Run: `cd mcp && go mod tidy`

Expected: dependencies resolve and `go.sum` is created.

- [ ] **Step 2: Write failing configuration tests**

```go
func TestLoadRejectsPartialConfiguration(t *testing.T) {
	t.Setenv("MCP_BASE_URL", "https://mcp.httpsms.com")
	t.Setenv("HTTPSMS_API_URL", "")
	_, err := Load()
	require.ErrorContains(t, err, "HTTPSMS_API_URL")
}
```

Cover required URLs, Redis URL, Firebase project/config, RSA PEM, key ID,
audiences, access-token TTL, refresh-token TTL, HTTP timeout, and production
HTTPS enforcement.

- [ ] **Step 3: Run configuration tests and confirm failure**

Run: `cd mcp && go test ./internal/config -count=1`

Expected: FAIL because `Load` does not exist.

- [ ] **Step 4: Implement validated configuration**

```go
type Config struct {
	Environment             string
	Port                    string
	BaseURL                 *url.URL
	APIURL                  *url.URL
	RedisURL                string
	FirebaseProjectID       string
	FirebaseAPIKey          string
	FirebaseAuthDomain      string
	FirebaseCertsURL        *url.URL
	SigningPrivateKeyPEM    []byte
	SigningKeyID            string
	MCPAudience             string
	APIAudience             string
	AccessTokenTTL          time.Duration
	APIDelegationTokenTTL   time.Duration
	AuthorizationCodeTTL    time.Duration
	RefreshTokenTTL         time.Duration
	ConfirmationTTL         time.Duration
	HTTPTimeout             time.Duration
	ReadToolsPerMinute      int
	SendToolsPerMinute      int
	KeyCreatesPerHour       int
	KeyRotationsPerHour     int
}
```

Use defaults: `PORT=8080`, MCP access token `15m`, API delegation token `2m`,
authorization code `2m`, refresh token `30d`, confirmation `5m`, and HTTP
timeout `10s`. Load key material from `MCP_SIGNING_PRIVATE_KEY`; when
`MCP_SIGNING_PRIVATE_KEY_FILE` is set, read that file instead and reject
configurations that set both. Rate-limit defaults are 120 read calls/minute,
30 SMS sends/minute, 10 phone API-key creations/hour, and 3 primary API-key
rotations/hour. Production URLs must use HTTPS.

- [ ] **Step 5: Write failing key-set tests**

```go
func TestKeySetSignsAudienceBoundTokens(t *testing.T) {
	keys := newTestKeySet(t)
	raw, err := keys.SignMCPAccessToken(
		Principal{UserID: "user-id", Email: "user@example.com"},
		"https://client.example/metadata.json",
		[]string{"messages:read"},
		15*time.Minute,
	)
	require.NoError(t, err)
	claims := parseTestClaims(t, raw, keys.PublicKey())
	assert.Equal(t, "https://mcp.httpsms.com/mcp", claims.Audience[0])
	assert.Equal(t, "user-id", claims.Subject)
}
```

Also assert the API token uses `https://api.httpsms.com`, has only requested
scopes, includes `kid`, and never exceeds the configured TTL.

- [ ] **Step 6: Implement claims, signing, and JWKS**

```go
type Principal struct {
	UserID string
	Email  string
}

type AccessClaims struct {
	ClientID string   `json:"client_id"`
	Email    string   `json:"email,omitempty"`
	Scopes   []string `json:"scopes"`
	Method   string   `json:"http_method,omitempty"`
	Path     string   `json:"http_path,omitempty"`
	jwt.RegisteredClaims
}
```

Load only PKCS#8 or PKCS#1 RSA private keys of at least 2048 bits. Sign with
RS256. Generate JWKS `n` and `e` values from the RSA public key and publish only
the public key.

- [ ] **Step 7: Add observability bootstrap**

Expose:

```go
func New(ctx context.Context, serviceName, version string) (
	logger zerolog.Logger,
	shutdown func(context.Context) error,
	err error,
)
```

Configure JSON logs, service/version fields, W3C propagation, and an
OpenTelemetry tracer provider. Provide a no-exporter local mode when no OTLP or
Google exporter configuration is present.

- [ ] **Step 8: Run MCP foundational tests**

Run: `cd mcp && go test ./internal/config ./internal/auth ./internal/observability -count=1`

Expected: PASS.

- [ ] **Step 9: Commit the MCP foundation**

```bash
git add mcp/go.mod mcp/go.sum mcp/internal/config mcp/internal/auth mcp/internal/observability
git commit -m "feat(mcp): add service foundation"
```

---

### Task 4: Implement Redis OAuth state and client registration

**Files:**
- Create: `mcp/internal/oauth/store.go`
- Create: `mcp/internal/oauth/store_test.go`
- Create: `mcp/internal/oauth/clients.go`
- Create: `mcp/internal/oauth/clients_test.go`
- Create: `mcp/internal/oauth/metadata.go`
- Create: `mcp/internal/oauth/metadata_test.go`

**Interfaces:**
- Produces: `oauth.Store` for transactions, codes, refresh tokens, DCR clients, and confirmations.
- Produces: `oauth.ClientResolver.Resolve(ctx context.Context, clientID string) (oauth.Client, error)`.
- Produces: metadata handlers mounted by Task 9.

- [ ] **Step 1: Define the store interface and failing one-time-use tests**

```go
type Store interface {
	PutAuthorizationTransaction(context.Context, AuthorizationTransaction, time.Duration) error
	GetAuthorizationTransaction(context.Context, string) (AuthorizationTransaction, error)
	PutAuthorizationCode(context.Context, AuthorizationCode, time.Duration) error
	ConsumeAuthorizationCode(context.Context, string) (AuthorizationCode, error)
	PutRefreshToken(context.Context, RefreshGrant, time.Duration) error
	RotateRefreshToken(context.Context, string, RefreshGrant, time.Duration) error
	PutDynamicClient(context.Context, Client, time.Duration) error
	GetDynamicClient(context.Context, string) (Client, error)
	PutConfirmation(context.Context, Confirmation, time.Duration) error
	ConsumeConfirmation(context.Context, string) (Confirmation, error)
}
```

The test must call `ConsumeAuthorizationCode` twice and assert the second call
returns `ErrNotFound`.

- [ ] **Step 2: Run store tests and confirm failure**

Run: `cd mcp && go test ./internal/oauth -run 'Store|AuthorizationCode' -count=1`

Expected: FAIL because the Redis store is missing.

- [ ] **Step 3: Implement Redis records and atomic consumption**

Use namespaced keys:

```text
httpsms:mcp:oauth:transaction:<sha256(id)>
httpsms:mcp:oauth:code:<sha256(code)>
httpsms:mcp:oauth:refresh:<sha256(token)>
httpsms:mcp:oauth:client:<sha256(client-id)>
httpsms:mcp:confirmation:<sha256(handle)>
```

Generate public values with `crypto/rand`, store only SHA-256 hashes, serialize
records as JSON, and consume codes/confirmations atomically with Redis
`GETDEL`. Rotate refresh tokens in a transaction that deletes the old hash and
creates the new hash with TTL.

- [ ] **Step 4: Write failing CIMD and DCR tests**

```go
func TestClientResolverRejectsPrivateMetadataTarget(t *testing.T) {
	resolver := NewClientResolver(http.DefaultClient, store)
	_, err := resolver.Resolve(context.Background(), "https://127.0.0.1/client.json")
	require.ErrorIs(t, err, ErrUnsafeClientMetadataURL)
}
```

Cover HTTPS enforcement, loopback exception only for redirect URIs, private and
link-local DNS results, response-size limit, unsafe redirects, exact
`client_id`, supported grant/response types, and `token_endpoint_auth_method=none`.

- [ ] **Step 5: Implement client metadata resolution**

```go
type Client struct {
	ID                      string   `json:"client_id"`
	Name                    string   `json:"client_name"`
	URI                     string   `json:"client_uri,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}
```

Fetch CIMD documents with a dedicated transport that rejects resolved private
addresses, disables automatic redirects, limits bodies to 256 KiB, and uses a
5-second timeout. Cache validated documents for 15 minutes. Implement
`POST /oauth/register` by validating the same fields, assigning a random
`client_id`, storing the record for 24 hours, and returning HTTP 201.

- [ ] **Step 6: Write and implement metadata handlers**

Assert exact JSON fields:

```json
{
  "resource": "https://mcp.httpsms.com/mcp",
  "authorization_servers": ["https://mcp.httpsms.com"],
  "scopes_supported": [
    "phones:read",
    "messages:read",
    "messages:send",
    "phone-api-keys:write",
    "user-api-key:rotate"
  ]
}
```

Authorization-server metadata must include issuer, authorization endpoint,
token endpoint, registration endpoint, JWKS URI, code and refresh grant types,
`S256`, supported scopes, and
`"client_id_metadata_document_supported": true`.

- [ ] **Step 7: Run OAuth state and metadata tests**

Run: `cd mcp && go test ./internal/oauth -run 'Store|Client|Metadata|Registration' -count=1`

Expected: PASS.

- [ ] **Step 8: Commit OAuth state and registration**

```bash
git add mcp/internal/oauth
git commit -m "feat(mcp): add OAuth state and metadata"
```

---

### Task 5: Implement Firebase login, authorization codes, and token grants

**Files:**
- Create: `mcp/internal/auth/firebase.go`
- Create: `mcp/internal/auth/firebase_test.go`
- Create: `mcp/internal/oauth/authorize.go`
- Create: `mcp/internal/oauth/authorize_test.go`
- Create: `mcp/internal/oauth/token.go`
- Create: `mcp/internal/oauth/token_test.go`
- Create: `mcp/internal/oauth/templates/authorize.html`

**Interfaces:**
- Produces: `auth.IdentityVerifier.Verify(ctx context.Context, raw string) (auth.Principal, error)`.
- Produces: `oauth.Server.HandleAuthorize`, `HandleFirebaseComplete`, and `HandleToken`.
- Consumes: Task 3 key set and Task 4 store/client resolver.

- [ ] **Step 1: Write failing Firebase verifier tests**

Serve a Firebase-style certificate map from `httptest`:

```json
{"firebase-test-key":"-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n"}
```

Sign a token with issuer
`https://securetoken.google.com/httpsms-test`, audience `httpsms-test`,
`sub=user-id`, `user_id=user-id`, and `email=user@example.com`. Assert valid
tokens return that principal and wrong issuer/audience/expiry fail.

- [ ] **Step 2: Implement the Firebase verifier**

```go
type IdentityVerifier interface {
	Verify(context.Context, string) (Principal, error)
}

func (v *FirebaseVerifier) Verify(ctx context.Context, raw string) (Principal, error) {
	claims := new(firebaseClaims)
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		v.keyfunc(ctx),
		jwt.WithIssuer("https://securetoken.google.com/"+v.projectID),
		jwt.WithAudience(v.projectID),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	if err != nil || !token.Valid || claims.Subject == "" {
		return Principal{}, ErrInvalidIdentityToken
	}
	return Principal{UserID: claims.Subject, Email: claims.Email}, nil
}
```

Use the same bounded refresh-on-missing-`kid` behavior as the API JWKS loader,
but decode Google's certificate-map response.

- [ ] **Step 3: Write failing authorization flow tests**

Test:

1. `/oauth/authorize` rejects missing state, PKCE, resource, or redirect URI.
2. a valid request creates a transaction and renders the Firebase page;
3. `/oauth/firebase/complete` rejects a bad identity token;
4. a valid token and approved scopes issue a one-time code redirect;
5. success and error redirects include the RFC 9207 `iss` parameter;
6. denial redirects with `error=access_denied`.

- [ ] **Step 4: Implement authorization and consent**

```go
type AuthorizationTransaction struct {
	ID                  string
	ClientID            string
	RedirectURI         string
	State               string
	Resource            string
	Scopes              []string
	CodeChallenge       string
	CodeChallengeMethod string
	CreatedAt           time.Time
}
```

Render `authorize.html` with the Firebase API key, auth domain, transaction ID,
client name, and human-readable scopes. The page must post the Firebase ID
token and approved scope list to `/oauth/firebase/complete`; it must never put
the token in a query string.

- [ ] **Step 5: Write failing token endpoint tests**

```go
func TestTokenEndpointConsumesCodeAndChecksPKCE(t *testing.T) {
	code := issueTestAuthorizationCode(t, store, "verifier")
	response := postToken(t, server, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {"verifier"},
		"client_id":     {"https://client.example/client.json"},
		"redirect_uri":  {"http://127.0.0.1:3210/callback"},
		"resource":      {"https://mcp.httpsms.com/mcp"},
	})
	require.Equal(t, http.StatusOK, response.StatusCode)
	postAgain := postToken(t, server, sameValues)
	require.Equal(t, http.StatusBadRequest, postAgain.StatusCode)
}
```

Cover code replay, wrong verifier, wrong client, wrong redirect URI, wrong
resource, refresh rotation, old-refresh replay, and scope narrowing.

- [ ] **Step 6: Implement authorization-code and refresh grants**

Return:

```json
{
  "access_token": "<mcp-jwt>",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "<opaque-value>",
  "scope": "messages:read messages:send"
}
```

Require the `resource` value to equal the MCP endpoint. Bind codes and refresh
grants to client ID, user, resource, redirect URI where applicable, and granted
scopes. Rotate refresh tokens on every use and reject scope expansion.

- [ ] **Step 7: Run OAuth flow tests**

Run: `cd mcp && go test ./internal/auth ./internal/oauth -count=1`

Expected: PASS.

- [ ] **Step 8: Commit Firebase-backed OAuth**

```bash
git add mcp/internal/auth/firebase.go mcp/internal/auth/firebase_test.go mcp/internal/oauth
git commit -m "feat(mcp): add Firebase OAuth exchange"
```

---

### Task 6: Build the typed httpSMS API client

**Files:**
- Create: `mcp/internal/httpsms/models.go`
- Create: `mcp/internal/httpsms/client.go`
- Create: `mcp/internal/httpsms/client_test.go`

**Interfaces:**
- Produces: `httpsms.Client` methods used by all MCP tools.
- Consumes: delegated API JWT strings supplied per call.

- [ ] **Step 1: Define the client interface and failing tests**

```go
type Client interface {
	ListPhones(context.Context, string, ListPhonesParams) ([]Phone, error)
	SendSMS(context.Context, string, SendSMSParams) (Message, error)
	ListMessageThreads(context.Context, string, ListMessageThreadsParams) ([]MessageThread, error)
	ListThreadMessages(context.Context, string, ListThreadMessagesParams) ([]Message, error)
	ListIncomingMessages(context.Context, string, ListIncomingMessagesParams) ([]Message, error)
	CreatePhoneAPIKey(context.Context, string, CreatePhoneAPIKeyParams) (PhoneAPIKey, error)
	RotateUserAPIKey(context.Context, string, string) (User, error)
}
```

For each method, use `httptest.Server` to assert method, path, encoded query or
JSON body, `Authorization: Bearer`, content type, request ID, and response
decoding.

- [ ] **Step 2: Run client tests and confirm failure**

Run: `cd mcp && go test ./internal/httpsms -count=1`

Expected: FAIL because the client and models do not exist.

- [ ] **Step 3: Implement API models and standard envelopes**

```go
type Response[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type APIError struct {
	StatusCode int
	Message    string
	Fields     map[string][]string
	RequestID  string
}
```

Define only fields required by MCP output schemas. Keep phone numbers, UUIDs,
timestamps, SIM, message type/status, encryption flag, content, attachments,
thread unread/archive state, and secret API-key fields.

- [ ] **Step 4: Implement the bounded HTTP client**

```go
func (c *client) do(
	ctx context.Context,
	token string,
	method string,
	path string,
	query url.Values,
	input any,
	output any,
) error
```

Use an `http.Client` with explicit timeout, `otelhttp.Transport`, connection
pool limits, 2 MiB response cap, and no automatic retries for writes. Decode
non-2xx responses into `APIError`; return an error when the body is malformed
or exceeds the limit. Never include request bodies or bearer tokens in errors.

- [ ] **Step 5: Run API client tests**

Run: `cd mcp && go test ./internal/httpsms -count=1`

Expected: PASS.

- [ ] **Step 6: Commit the API client**

```bash
git add mcp/internal/httpsms
git commit -m "feat(mcp): add httpSMS API client"
```

---

### Task 7: Implement read and send MCP tools

**Files:**
- Create: `mcp/internal/auth/middleware.go`
- Create: `mcp/internal/auth/middleware_test.go`
- Create: `mcp/internal/tools/phones.go`
- Create: `mcp/internal/tools/messages.go`
- Create: `mcp/internal/tools/messages_test.go`
- Create: `mcp/internal/tools/register.go`

**Interfaces:**
- Produces: typed handlers registered with `mcp.AddTool`.
- Consumes: `auth.PrincipalFromContext`, `auth.RequireScope`, `auth.KeySet`, and `httpsms.Client`.

- [ ] **Step 1: Write failing MCP bearer middleware tests**

Use `auth.RequireBearerToken` from the official SDK:

```go
middleware := mcpauth.RequireBearerToken(verifier.VerifyMCPToken, &mcpauth.RequireBearerTokenOptions{
	ResourceMetadataURL: "https://mcp.httpsms.com/.well-known/oauth-protected-resource",
})
```

Assert missing, expired, wrong-audience, and invalid tokens return `401` with
`WWW-Authenticate`, while a valid token stores `mcpauth.TokenInfo` containing
user ID, expiry, scopes, and the full principal in `Extra`.

- [ ] **Step 2: Implement the MCP token verifier adapter**

```go
func (v *Verifier) VerifyMCPToken(
	ctx context.Context,
	raw string,
	_ *http.Request,
) (*mcpauth.TokenInfo, error) {
	claims, err := v.VerifyAccessToken(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid access token", mcpauth.ErrInvalidToken)
	}
	return &mcpauth.TokenInfo{
		UserID:     claims.Subject,
		Scopes:     claims.Scopes,
		Expiration: claims.ExpiresAt.Time,
		Extra: map[string]any{
			"principal": Principal{UserID: claims.Subject, Email: claims.Email},
			"client_id": claims.ClientID,
		},
	}, nil
}
```

Add `RequireScope(ctx, scope)` and `PrincipalFromContext(ctx)` helpers that read
`mcpauth.TokenInfoFromContext`.

- [ ] **Step 3: Write failing typed-tool tests**

Create an in-memory MCP client/server pair. Register tools against an API client
stub and a test key set. Assert:

- `list_phones` returns stable structured content;
- `send_sms` forwards all supported optional fields;
- `list_message_threads` enforces a maximum limit of 20;
- `list_thread_messages` requires owner and contact;
- `list_incoming_messages` calls the dedicated incoming endpoint;
- missing scopes produce tool errors without API calls.

- [ ] **Step 4: Define typed tool inputs and outputs**

```go
type SendSMSInput struct {
	From        string   `json:"from" jsonschema:"registered httpSMS phone number in E.164 format"`
	To          string   `json:"to" jsonschema:"destination phone number in E.164 format"`
	Content     string   `json:"content" jsonschema:"SMS content"`
	SIM         string   `json:"sim,omitempty" jsonschema:"SIM1, SIM2, or DEFAULT"`
	RequestID   string   `json:"request_id,omitempty"`
	Encrypted   bool     `json:"encrypted,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type MessageListOutput struct {
	Messages []httpsms.Message `json:"messages"`
	Count    int               `json:"count"`
}
```

Define corresponding inputs/outputs for phones, threads, thread messages, and
incoming messages. Use pointer fields where omission differs from a zero value.

- [ ] **Step 5: Implement scoped handlers**

Each handler follows this sequence:

```go
principal, err := auth.RequireScope(ctx, auth.ScopeMessagesRead)
if err != nil {
	return nil, Output{}, err
}
delegated, err := keys.SignAPIDelegationToken(
	principal,
	[]string{auth.ScopeMessagesRead},
	http.MethodGet,
	"/v1/messages/incoming",
	apiTokenTTL,
)
if err != nil {
	return nil, Output{}, fmt.Errorf("sign API delegation token: %w", err)
}
items, err := api.ListIncomingMessages(ctx, delegated, params)
if err != nil {
	return toolError(err), Output{}, nil
}
return nil, MessageListOutput{Messages: items, Count: len(items)}, nil
```

Use `mcp.ToolAnnotations` to mark read tools as read-only and `send_sms` as
destructive/non-idempotent. Register tools in the approved deterministic order:
phones, send, threads, thread messages, incoming messages.

- [ ] **Step 6: Run tool tests**

Run: `cd mcp && go test ./internal/auth ./internal/tools -run 'Phones|SMS|Message|Scope' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit read and send tools**

```bash
git add mcp/internal/auth/middleware.go mcp/internal/auth/middleware_test.go mcp/internal/tools
git commit -m "feat(mcp): add SMS and message tools"
```

---

### Task 8: Implement API-key tools and confirmed rotation

**Files:**
- Create: `mcp/internal/tools/api_keys.go`
- Create: `mcp/internal/tools/api_keys_test.go`
- Modify: `mcp/internal/tools/register.go`
- Modify: `mcp/internal/oauth/store.go`

**Interfaces:**
- Produces: `create_phone_api_key`.
- Produces: `rotate_user_api_key` with MRTR confirmation and Redis state.
- Consumes: Task 4 confirmation store and Task 6 API client.

- [ ] **Step 1: Write failing phone API-key creation tests**

Assert the handler requires `phone-api-keys:write`, forwards only the name, and
returns:

```go
type CreatePhoneAPIKeyOutput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	APIKey    string `json:"api_key"`
	Sensitive bool   `json:"sensitive"`
}
```

Also assert the secret never appears in captured logs.

- [ ] **Step 2: Implement `create_phone_api_key`**

Mark the tool as non-idempotent and sensitive in its description. Mint only
`phone-api-keys:write` for the downstream call. Return
`Sensitive: true` and text instructing the user to store the key immediately.

- [ ] **Step 3: Write failing rotation confirmation tests**

The first invocation must not call the API:

```go
result, output, err := handler(ctx, requestWithoutInputResponses, RotateUserAPIKeyInput{})
require.NoError(t, err)
require.Nil(t, output)
require.Contains(t, result.InputRequests, "confirm_rotation")
require.NotEmpty(t, result.RequestState)
require.Zero(t, api.rotateCalls)
```

The confirmed retry must consume the stored handle, verify user/client/tool
binding, call the API once, and reject replay. Add a legacy explicit
`confirmation_handle` test for clients that cannot complete MRTR.

- [ ] **Step 4: Implement confirmation state and MRTR**

```go
type Confirmation struct {
	UserID    string
	ClientID  string
	Operation string
	CreatedAt time.Time
}
```

On the first call:

1. generate and store a five-minute confirmation handle;
2. return `InputRequests` with an `mcp.ElicitParams` boolean confirmation;
3. set `RequestState` to the opaque handle;
4. include a warning that the current primary API key will stop working.

On retry, require an accepted elicitation response or the explicit legacy
handle, atomically consume the handle, compare constant-time bindings, mint the
`user-api-key:rotate` API JWT, and call
`DELETE /v1/users/{principal.UserID}/api-keys`.

- [ ] **Step 5: Run API-key tool tests**

Run: `cd mcp && go test ./internal/tools -run 'APIKey|Rotate|Confirmation' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit API-key tools**

```bash
git add mcp/internal/tools mcp/internal/oauth/store.go
git commit -m "feat(mcp): add confirmed API key tools"
```

---

### Task 9: Assemble the MCP and OAuth HTTP server

**Files:**
- Create: `mcp/internal/server/rate_limit.go`
- Create: `mcp/internal/server/rate_limit_test.go`
- Create: `mcp/internal/server/server.go`
- Create: `mcp/internal/server/server_test.go`
- Create: `mcp/cmd/server/main.go`
- Create: `mcp/cmd/server/main_test.go`

**Interfaces:**
- Produces: `server.New(config.Config, Dependencies) (http.Handler, error)`.
- Produces: executable `mcp-server`.
- Consumes: all MCP, OAuth, auth, storage, API client, and observability components.

- [ ] **Step 1: Write failing route and protocol tests**

Assert:

- `GET /health` returns 200;
- metadata, JWKS, authorize, token, and registration routes are mounted;
- unauthenticated `POST /mcp` returns 401 and protected-resource metadata;
- authenticated `server/discover` negotiates `2026-07-28`;
- legacy `initialize` negotiates `2025-11-25`;
- `tools/list` order is deterministic;
- `GET /mcp` and `DELETE /mcp` are rejected in stateless mode.

- [ ] **Step 2: Write and implement Redis tool rate-limit tests**

```go
func TestToolRateLimiterSeparatesUsersAndTools(t *testing.T) {
	limiter := NewToolRateLimiter(redisClient, Limits{ReadPerMinute: 2})
	require.NoError(t, limiter.Allow(ctx, "user-a", "list_phones"))
	require.NoError(t, limiter.Allow(ctx, "user-a", "list_phones"))
	require.ErrorIs(t, limiter.Allow(ctx, "user-a", "list_phones"), ErrRateLimited)
	require.NoError(t, limiter.Allow(ctx, "user-b", "list_phones"))
}
```

Use Redis `INCR` plus `EXPIRE` in a transaction or Lua script so the first
increment sets the window atomically. Key by SHA-256 user ID, tool name, and
window start. Apply the configured read, send, key-create, and key-rotation
budgets before tool execution. Return a structured MCP rate-limit error with a
retry-after duration.

- [ ] **Step 3: Configure the official Streamable HTTP handler**

```go
mcpServer := mcp.NewServer(
	&mcp.Implementation{Name: "httpSMS", Version: version},
	&mcp.ServerOptions{},
)
tools.Register(mcpServer, dependencies.Tools)

mcpHandler := mcp.NewStreamableHTTPHandler(
	func(*http.Request) *mcp.Server { return mcpServer },
	&mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		MaxRequestBodyBytes:          1 << 20,
		Logger:                       slogLogger,
	},
)
```

Use the SDK defaults that support `2026-07-28` and `2025-11-25`. Add an
explicit protocol-version test so a future SDK upgrade cannot silently remove
either required version.

- [ ] **Step 4: Assemble the middleware chain**

Order:

1. request ID;
2. panic recovery;
3. secure response headers;
4. OpenTelemetry HTTP middleware;
5. redacted structured request logging;
6. OAuth/public routes;
7. official `auth.RequireBearerToken` around `/mcp`;
8. per-user/per-tool Redis rate limiting using `Mcp-Name`;
9. MCP Streamable HTTP handler.

Set `Cache-Control: no-store` on token, authorization, Firebase completion,
secret-result, and error responses. Set permissive CORS only on public metadata
handlers; do not enable wildcard credentialed CORS.

- [ ] **Step 5: Implement dependency construction and graceful shutdown**

`main.go` must:

```go
cfg, err := config.Load()
if err != nil {
	log.Fatal().Err(err).Msg("load configuration")
}
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

handler, shutdown, err := build(ctx, cfg, Version)
if err != nil {
	log.Fatal().Err(err).Msg("build MCP server")
}
httpServer := &http.Server{
	Addr:              ":" + cfg.Port,
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       15 * time.Second,
	WriteTimeout:      30 * time.Second,
	IdleTimeout:       60 * time.Second,
}
```

Run the server, wait for cancellation or a serve error, then shut down HTTP,
Redis, MCP handler resources, and telemetry with a 10-second deadline.

- [ ] **Step 6: Run server tests and a local smoke test**

Run:

```bash
cd mcp
go test ./internal/server ./cmd/server -count=1
go build ./cmd/server
```

Expected: tests pass and the binary builds.

- [ ] **Step 7: Commit server assembly**

```bash
git add mcp/internal/server mcp/cmd/server
git commit -m "feat(mcp): serve stateless MCP over HTTP"
```

---

### Task 10: Add container and Cloud Run deployment configuration

**Files:**
- Create: `mcp/Dockerfile`
- Create: `mcp/cloudbuild.yaml`
- Create: `mcp/.dockerignore`
- Create: `mcp/README.md`

**Interfaces:**
- Produces: container listening on `$PORT`.
- Produces: Cloud Build deployment for service `http-sms-mcp`.

- [ ] **Step 1: Add the multi-stage Dockerfile**

```dockerfile
FROM golang:1.25-alpine AS builder
ARG GIT_COMMIT
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w -X main.Version=${GIT_COMMIT}" \
    -o /out/mcp-server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S mcp && adduser -S mcp -G mcp
USER mcp
COPY --from=builder /out/mcp-server /usr/local/bin/mcp-server
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mcp-server"]
```

- [ ] **Step 2: Build and inspect the container**

Run:

```bash
docker build -t httpsms-mcp:test mcp
docker image inspect httpsms-mcp:test --format '{{.Config.User}} {{.Config.ExposedPorts}}'
```

Expected: image builds, runs as `mcp`, and exposes `8080/tcp`.

- [ ] **Step 3: Add Cloud Build deployment**

Mirror `api/cloudbuild.yaml` with:

```yaml
substitutions:
  _SERVICE_NAME: http-sms-mcp
  _REGION: us-east1
```

Build from `mcp/Dockerfile`, publish commit and `latest` tags, and deploy:

```bash
gcloud run deploy $_SERVICE_NAME \
  --image=us.gcr.io/$PROJECT_ID/$_SERVICE_NAME:$SHORT_SHA \
  --region=$_REGION \
  --platform=managed \
  --allow-unauthenticated \
  --port=8080
```

Use Cloud Run secret references for signing key, Redis URL, and Firebase
configuration. Do not place secret values in YAML.

- [ ] **Step 4: Document operations**

`mcp/README.md` must document:

- required environment variables and safe defaults;
- local startup;
- health and MCP URLs;
- Cloud Build invocation;
- one-time `gcloud run domain-mappings create --service http-sms-mcp --domain mcp.httpsms.com --region us-east1`;
- DNS verification;
- signing-key rotation order;
- removal of `2025-11-25` compatibility;
- redaction and secret-management requirements.

- [ ] **Step 5: Commit deployment files**

```bash
git add mcp/Dockerfile mcp/.dockerignore mcp/cloudbuild.yaml mcp/README.md
git commit -m "build(mcp): add Cloud Run deployment"
```

---

### Task 11: Add MCP end-to-end integration tests

**Files:**
- Create: `tests/mcp_helpers_test.go`
- Create: `tests/mcp_integration_test.go`
- Modify: `tests/generate-firebase-credentials.sh`
- Modify: `tests/.gitignore`
- Modify: `tests/docker-compose.yml`
- Modify: `tests/.env.test`
- Modify: `tests/seed.sql`
- Modify: `tests/go.mod`
- Modify: `tests/go.sum`
- Modify: `tests/README.md`

**Interfaces:**
- Produces: full-stack tests against `http://localhost:8082/mcp`.
- Consumes: API, Redis, WireMock, database seed, and phone emulator stack.

- [ ] **Step 1: Add integration dependencies and helpers**

Add `github.com/modelcontextprotocol/go-sdk v1.7.0` to `tests/go.mod`.

Implement:

```go
const mcpBaseURL = "http://localhost:8082"
const mcpTestUserID = "mcp-test-user-id"

func newMCPClient(t *testing.T, accessToken, protocolVersion string) *mcp.ClientSession
func completeOAuthCodeFlow(t *testing.T, scopes []string) tokenResponse
func signFirebaseTestToken(t *testing.T, userID, email string) string
func pkcePair(t *testing.T) (verifier, challenge string)
```

Extend `generate-firebase-credentials.sh` so the same invocation also writes:

```text
tests/mcp-test-signing-key.pem
tests/mcp-test-signing-cert.pem
tests/wiremock/mappings/firebase-certs.generated.json
```

Generate the RSA key and self-signed certificate with OpenSSL, emit a WireMock
mapping whose response body is a Firebase certificate map keyed by
`mcp-test-key`, and add all three generated paths to `tests/.gitignore`.
`signFirebaseTestToken` reads the generated private key. The MCP container
mounts that key read-only. No private key or generated certificate is committed.

Add a dedicated `mcp-test-user-id` user with primary key
`mcp-test-user-api-key` to `tests/seed.sql`. All MCP integration tokens use
that Firebase UID so key rotation cannot invalidate the shared user used by
pre-existing integration tests.

- [ ] **Step 2: Extend Docker Compose**

Add:

```yaml
mcp:
  build:
    context: ../mcp
  ports:
    - "8082:8080"
  depends_on:
    api:
      condition: service_healthy
    redis:
      condition: service_healthy
    wiremock:
      condition: service_healthy
  env_file:
    - .env.test
  environment:
    PORT: "8080"
    MCP_BASE_URL: http://localhost:8082
    HTTPSMS_API_URL: http://api:8000
    FIREBASE_CERTS_URL: http://wiremock:8080/firebase-certs
    MCP_SIGNING_PRIVATE_KEY_FILE: /run/secrets/mcp-test-signing-key.pem
  volumes:
    - ./mcp-test-signing-key.pem:/run/secrets/mcp-test-signing-key.pem:ro
```

Add an MCP health check at `http://localhost:8080/health`. Configure the API
with MCP issuer, audience, and JWKS URL using the Docker service URL.

- [ ] **Step 3: Write metadata, OAuth, and authorization tests**

Cover:

- protected-resource and authorization-server metadata;
- unauthenticated 401 and `WWW-Authenticate`;
- PKCE authorization-code exchange;
- wrong issuer, audience, redirect URI, verifier, and replay;
- refresh-token rotation;
- insufficient scope.

Run: `cd tests && go test -run 'TestMCPMetadata|TestMCPOAuth|TestMCPAuthorization' -count=1`

Expected before the stack changes are complete: FAIL because MCP is unavailable.

- [ ] **Step 4: Write protocol compatibility tests**

Use the official client transport with an authenticated `http.Client`. Assert
`server/discover` and tool calls work for `2026-07-28`, and legacy initialize
works for `2025-11-25`. Assert the tool names exactly match the approved seven
tools.

- [ ] **Step 5: Write read-tool integration tests**

Assert:

- `list_phones` returns the seeded phone;
- `list_message_threads` returns seeded/created threads;
- `list_thread_messages` returns the expected conversation;
- a received SMS created through the phone endpoint appears in
  `list_incoming_messages`;
- a missed call does not appear in incoming messages.

- [ ] **Step 6: Write send-SMS integration test**

Call `send_sms`, wait for the existing FCM emulator request, fire SENT and
DELIVERED events, and assert the message reaches `delivered`. Reuse
`waitForFCMPush`, `fireEvent`, and `pollMessageStatus`.

- [ ] **Step 7: Write API-key integration tests**

Assert:

- `create_phone_api_key` returns a `pk_` secret and the API accepts it;
- first rotation call does not rotate;
- confirmed rotation returns a new `uk_` secret;
- the old seeded user API key returns 401;
- the replacement key authenticates successfully;
- the confirmation handle cannot be replayed.

Use only `mcp-test-user-api-key` for this assertion; never rotate the existing
`test-user-api-key`.

- [ ] **Step 8: Run the complete integration stack**

Run:

```bash
cd tests
bash generate-firebase-credentials.sh firebase-credentials.json
export FIREBASE_CREDENTIALS=$(jq -c . firebase-credentials.json)
docker compose up -d --build --wait
docker compose wait seed
sleep 2
go test -v -timeout 300s ./...
docker compose down -v
```

Expected: all existing and MCP integration tests pass.

- [ ] **Step 9: Update integration documentation**

Update `tests/README.md` architecture, service table, test coverage, startup,
ports, troubleshooting, and CI sections for the MCP service.

- [ ] **Step 10: Commit integration coverage**

```bash
git add tests
git commit -m "test(mcp): add full-stack integration coverage"
```

---

### Task 12: Wire CI, format, and verify the complete feature

**Files:**
- Modify: `.github/workflows/api.yml`
- Modify as produced by formatting: all changed Go files

**Interfaces:**
- Produces: required CI gate for API and MCP tests.
- Consumes: every prior task.

- [ ] **Step 1: Extend CI service readiness checks**

After the API health loop, add an MCP health loop:

```bash
echo "Waiting for MCP to be healthy..."
for i in $(seq 1 40); do
  if docker compose exec mcp wget -qO- http://localhost:8080/health >/dev/null 2>&1; then
    echo "MCP is healthy!"
    break
  fi
  if [ "$i" -eq 40 ]; then
    docker compose logs mcp
    exit 1
  fi
  sleep 5
done
```

Keep deployment gated on the full integration job.

- [ ] **Step 2: Add MCP unit-test and build steps**

Before integration tests:

```yaml
- name: Run MCP Unit Tests
  working-directory: ./mcp
  run: go test -race -count=1 ./...

- name: Build MCP Server
  working-directory: ./mcp
  run: go build ./cmd/server
```

- [ ] **Step 3: Format and tidy modules**

Run:

```bash
cd api
go mod tidy
go-fumpt -w pkg/auth pkg/middlewares pkg/requests pkg/validators pkg/handlers pkg/di
goimports -w pkg/auth pkg/middlewares pkg/requests pkg/validators pkg/handlers pkg/di

cd ../mcp
go mod tidy
go-fumpt -w .
goimports -w .

cd ../tests
go mod tidy
go-fumpt -w mcp_helpers_test.go mcp_integration_test.go
goimports -w mcp_helpers_test.go mcp_integration_test.go
```

Expected: all formatters and module tidies complete without errors.

- [ ] **Step 4: Run targeted unit suites**

Run:

```bash
cd api
go test ./pkg/auth ./pkg/middlewares ./pkg/requests ./pkg/validators ./pkg/handlers ./pkg/di

cd ../mcp
go test -race ./...
go build ./cmd/server
```

Expected: PASS.

- [ ] **Step 5: Run complete API tests**

Run: `cd api && go test ./...`

Expected: PASS.

- [ ] **Step 6: Run complete integration suite**

Run the Task 11 Docker Compose command.

Expected: PASS with both protocol versions and all seven tools covered.

- [ ] **Step 7: Inspect generated and deployment artifacts**

Run:

```bash
git diff --check
git status --short
grep -n '"/messages/incoming"' api/docs/swagger.json
grep -n '_SERVICE_NAME: http-sms-mcp' mcp/cloudbuild.yaml
grep -n 'github.com/modelcontextprotocol/go-sdk v1.7.0' mcp/go.mod tests/go.mod
```

Expected: no whitespace errors; generated Swagger, deployment service name,
and pinned SDK versions are present.

- [ ] **Step 8: Commit CI and final formatting**

```bash
git add .github/workflows/api.yml api mcp tests
git commit -m "ci(mcp): gate deploys on MCP tests"
```

- [ ] **Step 9: Review final history and worktree state**

Run:

```bash
git log --oneline main..HEAD
git status --short --branch
```

Expected: focused commits for API auth, incoming messages, MCP foundation,
OAuth, API client, tools, server, deployment, integration tests, and CI; the
worktree is clean.
