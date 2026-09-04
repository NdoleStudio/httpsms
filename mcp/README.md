# httpSMS MCP Server

A standalone Model Context Protocol (MCP) server for httpSMS, deployed as its
own Cloud Run service and reachable at `https://mcp.httpsms.com/mcp`. It
implements MCP `2026-07-28` (with a temporary `2025-11-25` compatibility
path), authenticates users through existing Firebase accounts via OAuth 2.1,
and calls the httpSMS API (`api.httpsms.com`) on the user's behalf. It does
not access the httpSMS database directly and never stores a user's primary
API key or Firebase refresh token.

See `docs/superpowers/specs/2026-09-03-httpsms-mcp-server-design.md` at the
repository root for the full design.

## Environment variables

`internal/config.Load()` reads all configuration from the environment and
fails fast, naming every missing/invalid setting, if anything required is
absent. In `production` (`ENV=production`), every URL-valued setting must use
`https`.

### Required (no default)

| Variable | Purpose |
| --- | --- |
| `MCP_BASE_URL` | This service's own public base URL, e.g. `https://mcp.httpsms.com`. Used as the JWT issuer and to derive `MCP_AUDIENCE`. |
| `HTTPSMS_API_URL` | The httpSMS API base URL, e.g. `https://api.httpsms.com`. Used to derive `API_AUDIENCE`. |
| `REDIS_URL` | Connection string for OAuth/rate-limit/confirmation state. **Must** be a standalone Redis instance (see [Redis constraint](#redis-constraint-standalone-only)). |
| `FIREBASE_PROJECT_ID` | Firebase project used to verify ID tokens during login. |
| `FIREBASE_API_KEY` | Firebase Web API key used by the hosted login page's client-side SDK. |
| `FIREBASE_AUTH_DOMAIN` | Firebase Auth domain used by the hosted login page's client-side SDK. |
| `MCP_SIGNING_KEY_ID` | The `kid` this instance signs with and publishes at `/.well-known/jwks.json`. |
| `MCP_SIGNING_PRIVATE_KEY` **or** `MCP_SIGNING_PRIVATE_KEY_FILE` | PEM-encoded RSA private key (>=2048 bits, PKCS#1) used to sign MCP access tokens and API delegation tokens. Exactly one of these two must be set. |

### Optional (safe defaults)

| Variable | Default | Purpose |
| --- | --- | --- |
| `ENV` | `local` | Set to `production` in Cloud Run to enforce HTTPS on every configured URL. |
| `PORT` | `8080` | TCP port the HTTP server listens on. Cloud Run supplies this automatically; do not set it manually in Cloud Run. |
| `FIREBASE_CERTS_URL` | Google's public Firebase JWKS endpoint | Override only for local/test identity providers. |
| `MCP_AUDIENCE` | `<MCP_BASE_URL>/mcp` | Audience MCP access tokens are bound to. |
| `API_AUDIENCE` | `<HTTPSMS_API_URL>` | Audience API delegation tokens are bound to. **Must match** the API's `MCP_AUTH_AUDIENCE` (see [Coordinating with the API](#coordinating-with-the-api)). |
| `MCP_ACCESS_TOKEN_TTL` | `15m` | MCP access token lifetime. |
| `API_DELEGATION_TOKEN_TTL` | `2m` | Downstream API delegation token lifetime. |
| `AUTHORIZATION_CODE_TTL` | `2m` | OAuth authorization code lifetime. |
| `REFRESH_TOKEN_TTL` | `720h` (30 days) | OAuth refresh token lifetime. |
| `CONFIRMATION_TTL` | `5m` | Primary API-key-rotation confirmation handle lifetime. |
| `HTTP_TIMEOUT` | `10s` | Bounds every outbound call to the httpSMS API and to OAuth client metadata documents. |
| `READ_TOOLS_PER_MINUTE` | `120` | Per-user rate limit for read-only tools. |
| `SEND_TOOLS_PER_MINUTE` | `30` | Per-user rate limit for `send_sms`. |
| `KEY_CREATES_PER_HOUR` | `10` | Per-user rate limit for `create_phone_api_key`. |
| `KEY_ROTATIONS_PER_HOUR` | `3` | Per-user rate limit for `rotate_user_api_key`. |

None of the values above have a safe committed default for secrets: `.env`
files, `*.pem`/`*.key` files, and anything matching `mcp/.dockerignore` must
never be committed. See [Secrets and redaction](#secrets-and-redaction).

## Local startup

```bash
cd mcp
go run ./cmd/server
```

Required variables must be exported first, e.g. via a local (not committed)
`.env` loaded by your shell, or `export`. A quick way to generate a local
throwaway signing key:

```bash
mkdir -p .local && openssl genrsa -out .local/mcp-signing-key.pem 2048  # .local/ is gitignored; never commit
export MCP_SIGNING_PRIVATE_KEY_FILE=.local/mcp-signing-key.pem
export MCP_SIGNING_KEY_ID=local-dev-1
```

A local standalone Redis (`redis-server`, or the one already provided by
`tests/docker-compose.yml`) satisfies `REDIS_URL`.

## Health and MCP URLs

| Path | Purpose |
| --- | --- |
| `GET /healthz`, `GET /health` | Unauthenticated liveness/readiness check used by Cloud Run. Reports process readiness only; it does not probe Redis or the API, by design (see design doc, "Rate Limiting and Reliability"). |
| `POST /mcp` | The MCP Streamable HTTP endpoint. Requires a valid MCP access token. |
| `GET /.well-known/oauth-protected-resource`, `GET /.well-known/oauth-protected-resource/mcp` | OAuth protected-resource metadata. |
| `GET /.well-known/oauth-authorization-server` | OAuth authorization-server metadata. |
| `GET /.well-known/jwks.json` | This service's public signing key(s), consumed by MCP clients and by the httpSMS API's delegated-auth middleware. |
| `POST /oauth/register` | Legacy Dynamic Client Registration (compatibility only; Client ID Metadata Documents are preferred). |
| `GET /oauth/authorize`, `POST /oauth/firebase/complete`, `POST /oauth/token` | OAuth authorization-code + PKCE flow. |

Cloud Run supplies the listen port through `PORT`; the container's `EXPOSE
8080` documents the default used both locally and by Cloud Build's
`--port=8080` deploy flag. These must stay in sync: if `PORT` is ever
overridden in Cloud Run, `--port` in `cloudbuild.yaml` must change to match,
or the health check will fail and the revision will never become ready.

## Cloud Build invocation

`cloudbuild.yaml` mirrors `api/cloudbuild.yaml`: it builds `mcp/Dockerfile`
with Kaniko, publishes `:$SHORT_SHA` and `:latest` tags to
`us.gcr.io/$PROJECT_ID/http-sms-mcp`, and deploys the dedicated Cloud Run
service `http-sms-mcp` in `us-east1`.

Trigger manually with:

```bash
gcloud builds submit --config=mcp/cloudbuild.yaml .
```

or wire it to a Cloud Build trigger the same way `api/cloudbuild.yaml` is
wired, scoped to changes under `mcp/`.

Non-sensitive configuration (`ENV`, `MCP_BASE_URL`, `HTTPSMS_API_URL`,
`FIREBASE_PROJECT_ID`, `FIREBASE_AUTH_DOMAIN`) is passed with
`--set-env-vars` from `cloudbuild.yaml` substitutions. **Secrets are never
placed in `cloudbuild.yaml` or Cloud Build substitutions.** They are
referenced from Google Secret Manager with `--set-secrets`:

| Cloud Run env var | Secret Manager secret |
| --- | --- |
| `MCP_SIGNING_PRIVATE_KEY` | `mcp-signing-private-key` |
| `MCP_SIGNING_KEY_ID` | `mcp-signing-key-id` |
| `REDIS_URL` | `mcp-redis-url` |
| `FIREBASE_API_KEY` | `mcp-firebase-api-key` |

Create/update these once with, e.g.:

```bash
printf '%s' "$PRIVATE_KEY_PEM" | gcloud secrets create mcp-signing-private-key --data-file=-
printf '%s' "prod-mcp-key-1"   | gcloud secrets create mcp-signing-key-id --data-file=-
printf '%s' "$REDIS_URL"       | gcloud secrets create mcp-redis-url --data-file=-
printf '%s' "$FIREBASE_API_KEY" | gcloud secrets create mcp-firebase-api-key --data-file=-
```

The Cloud Run service's runtime service account needs
`roles/secretmanager.secretAccessor` on each secret.

## One-time custom-domain mapping

Ensure the Cloud project has verified ownership of `httpsms.com` in Google
Search Console before creating its first domain mapping. Existing
`app.httpsms.com` or `api.httpsms.com` mappings usually mean this prerequisite
is already satisfied.

`mcp.httpsms.com` is mapped once, not on every deploy:

```bash
gcloud run domain-mappings create \
  --service=http-sms-mcp \
  --domain=mcp.httpsms.com \
  --region=us-east1
```

Then add the DNS records the command prints (typically a `CNAME` to
`ghs.googlehosted.com` or the A/AAAA records Cloud Run reports) at the DNS
provider for `httpsms.com`. Verify with:

```bash
gcloud run domain-mappings describe \
  --domain=mcp.httpsms.com --region=us-east1 \
  --format="value(status.conditions)"
dig +short mcp.httpsms.com
curl -sf https://mcp.httpsms.com/healthz
```

`MCP_BASE_URL` must already be `https://mcp.httpsms.com` (see
`cloudbuild.yaml` substitutions) **before** mapping the domain: this service
signs every JWT with that value as the issuer, and OAuth/MCP discovery
documents publish it as the resource/issuer URL. Changing `MCP_BASE_URL`
after clients have discovered the old value invalidates every previously
issued MCP access and refresh token (see rotation notes below) and
re-triggers client-side discovery.

## Firebase authorized domains and providers

The hosted login page (served from this service, not from `web/`) uses the
Firebase Web SDK client-side with `FIREBASE_API_KEY` / `FIREBASE_AUTH_DOMAIN`
/ `FIREBASE_PROJECT_ID`. Firebase only allows sign-in from **authorized
domains** configured per-project in the Firebase console
(*Authentication → Settings → Authorized domains*):

- `mcp.httpsms.com` must be added there before the first real login, or every
  provider sign-in attempt served from `mcp.httpsms.com` will fail
  client-side with `auth/unauthorized-domain`.
- This is a **separate, one-time console step**, independent of DNS mapping
  and independent of `web/`'s own authorized-domain entry (`app.httpsms.com`
  or equivalent) — adding one does not add the other.
- Enable the same identity providers already enabled for `web/` (Email/
  Password, Google, GitHub) for this project; this service reuses the
  existing Firebase project (`FIREBASE_PROJECT_ID=httpsms-86c51`), it does
  not create a second one.

## Coordinating with the API

This service and `api/` share exactly one trust relationship: the API trusts
this service's JWKS to validate delegated API JWTs (see design doc,
"Delegated MCP-to-API Authentication"). The API side needs three variables
set (in `api`'s own deployment, not here):

| API variable | Must equal |
| --- | --- |
| `MCP_AUTH_ISSUER` | This service's `MCP_BASE_URL` (with no trailing slash). |
| `MCP_AUTH_AUDIENCE` | This service's `API_AUDIENCE` (defaults to `HTTPSMS_API_URL`). |
| `MCP_AUTH_JWKS_URL` | `https://mcp.httpsms.com/.well-known/jwks.json`. |

The API's JWKS cache (`api/pkg/auth/mcp_jwks.go`) refreshes automatically,
at most once per request, whenever it sees a `kid` it doesn't already have
cached, and otherwise every 15 minutes. **No manual API restart or cache
flush is required after this service rotates its signing key** — the API
self-heals on the next delegated request that carries the new `kid`.

## Signing-key rotation

This service holds exactly **one** active signing key/`kid` at a time (see
`internal/auth.KeySet`); it does not publish overlapping old+new keys in its
JWKS. Rotation therefore works like this, in order:

1. Generate a new RSA private key (>=2048 bits) and choose a **new, never
   reused** `kid` (e.g. append an incrementing suffix: `prod-mcp-key-2`).
2. Update the `mcp-signing-private-key` and `mcp-signing-key-id` secrets in
   Secret Manager with the new values (add new versions; do not delete the
   old versions until the rollout below is confirmed healthy, so you can roll
   back).
3. Redeploy this service (`gcloud builds submit --config=mcp/cloudbuild.yaml
   .`, or `gcloud run services update http-sms-mcp
   --update-secrets=...:latest`). The new revision immediately signs with,
   and publishes, only the new key.
4. Confirm `/.well-known/jwks.json` now serves the new `kid` and that the API
   picks it up (its cache self-heals; see above). Watch API logs for
   delegated-auth failures during the rollout window.
5. **Expected, safe side effect:** every previously issued MCP access token
   (`MCP_ACCESS_TOKEN_TTL`, default 15 minutes) can no longer be verified
   once the old key is gone from this service's in-memory `KeySet`, because
   verification here has no fallback to an old key. Connected MCP clients see
   a `401` and must use their (unaffected) refresh token to obtain a new
   access token signed with the new key — refresh tokens are opaque, hashed,
   Redis-stored values unrelated to the signing key, so they are **not**
   invalidated by rotation.
6. Once you've confirmed the new revision is healthy and no client-visible
   regressions appear, destroy the old secret versions (`gcloud secrets
   versions destroy`) if you want to fully retire the old key material.

Rotate on a schedule and immediately, out of band, if the private key or its
`.pem`/`.env` file is ever suspected compromised.

## Redis constraint: standalone only

`REDIS_URL` **must** point at a standalone Redis instance (e.g. Memorystore
Basic tier, or a single `redis-server`) — never a Redis Cluster or Ring
endpoint. `internal/oauth.NewRedisStore` **panics at startup** if given a
cluster/ring client, because refresh-token rotation runs a Lua script that
touches multiple related keys and cannot execute across cluster hash slots.
This is enforced in code, not just documentation, so a misconfigured cluster
URL fails fast at boot instead of silently corrupting refresh-token state.

## Removing `2025-11-25` compatibility

This service currently negotiates both MCP `2026-07-28` (primary) and
`2025-11-25` (temporary compatibility, per the design doc's rollout step 7).
When client telemetry confirms `2025-11-25` initialization requests have
stopped:

1. Remove the legacy negotiation path and its tests in
   `internal/server` (search for `2025-11-25`).
2. Update the protected-resource/authorization-server metadata and any
   client-facing documentation that still mentions the legacy version.
3. Deploy and confirm `TestLegacyInitializeNegotiates20251125`-equivalent
   coverage has been removed, not just skipped.

This is a follow-up code change, not a configuration flag — there is no
environment variable that disables `2025-11-25` today.

## Deployment and rollback

Deploy:

```bash
gcloud builds submit --config=mcp/cloudbuild.yaml .
```

Each deploy publishes both `:$SHORT_SHA` and `:latest` image tags. To roll
back to a previously known-good commit's image without rebuilding:

```bash
gcloud run deploy http-sms-mcp \
  --image=us.gcr.io/$PROJECT_ID/http-sms-mcp:<previous-short-sha> \
  --region=us-east1 --platform=managed --allow-unauthenticated --port=8080
```

Cloud Run also keeps prior revisions: you can shift traffic back instantly
without a new image, without touching secrets, via:

```bash
gcloud run services update-traffic http-sms-mcp \
  --region=us-east1 --to-revisions=<previous-revision-name>=100
```

Prefer `update-traffic` for a fast rollback (no rebuild, seconds) and the
`--image=<previous-sha>` redeploy when you also need to restore a previous
secret binding.

## Secrets and redaction

- Never commit `.env` files, `*.pem`/`*.key` files, or literal secret values
  into `cloudbuild.yaml`, `Dockerfile`, or any other tracked file (enforced
  by `mcp/.dockerignore` for image layers; enforce it for source control with
  your usual pre-commit/secret-scanning tooling).
- Secrets are injected only via Cloud Run `--set-secrets` from Google Secret
  Manager, per the [Cloud Build invocation](#cloud-build-invocation) table.
- The following must never appear in logs, traces, metrics, or error
  messages: MCP and downstream bearer tokens, Firebase ID tokens,
  authorization codes and refresh tokens, PKCE verifiers, primary and phone
  API keys, and SMS content/attachment payloads. Tool results that return a
  newly created or rotated key mark it as a sensitive, one-time value and
  this service never caches it.
