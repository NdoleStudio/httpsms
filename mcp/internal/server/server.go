package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/config"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
	"github.com/NdoleStudio/httpsms/mcp/internal/tools"
)

// implementationName is this MCP server's own name, published in its MCP
// Implementation and in every discover/initialize response.
const implementationName = "httpSMS"

// maxMCPRequestBodyBytes bounds every request body the Streamable HTTP
// handler reads, per the approved design.
const maxMCPRequestBodyBytes = 1 << 20 // 1 MiB

// protectedResourceMetadataPath and authorizationServerMetadataPath are the
// well-known discovery routes RFC 9728 and RFC 8414 require.
const (
	protectedResourceMetadataPath = "/.well-known/oauth-protected-resource"
	// protectedResourceMetadataMCPPath is the path-suffixed alias RFC 9728
	// Section 3.1 actually prescribes for a protected resource whose
	// identifier carries a path component ("/mcp"): the well-known segment
	// is inserted between the host and that path, giving
	// "/.well-known/oauth-protected-resource/mcp". Clients differ in which
	// of the two they request (some, including the MCP SDKs, probe the
	// root form first), so both are mounted and serve the identical
	// document.
	protectedResourceMetadataMCPPath = protectedResourceMetadataPath + mcpPath
	authorizationServerMetadataPath  = "/.well-known/oauth-authorization-server"
	jwksPath                         = "/.well-known/jwks.json"
	mcpPath                          = "/mcp"
	registerPath                     = "/oauth/register"
	authorizePath                    = "/oauth/authorize"
	firebaseCompletePath             = "/oauth/firebase/complete"
	tokenPath                        = "/oauth/token"
	healthzPath                      = "/healthz"
	healthPath                       = "/health"
	requestIDHeaderName              = "X-Request-Id"
)

// maxRequestIDLength bounds how long an inbound X-Request-Id header may be
// before this service replaces it with one of its own. An unbounded,
// unvalidated upstream value would be echoed into every response header and
// every structured log line for the request, which is a log-injection and
// header-smuggling vector as well as an unbounded memory cost per request.
const maxRequestIDLength = 128

// requestIDExtraChars are the non-alphanumeric characters an inbound
// X-Request-Id may contain. The set covers the shapes real proxies emit
// (UUIDs, W3C trace IDs, base64url tokens) while excluding every control
// character, whitespace character, and non-ASCII byte.
const requestIDExtraChars = "-_.:+=/"

// Dependencies are the already-constructed components New wires into the
// httpSMS MCP service's HTTP surface. Every dependency is built and owned
// by the caller (see cmd/server/main.go's build function); New only
// assembles routes and middleware around them and never constructs,
// configures, or closes any of them itself.
type Dependencies struct {
	// Logger is used for structured request logging. It must never log a
	// bearer token, request body, cookie, or other secret.
	Logger zerolog.Logger

	// Keys signs and verifies every JWT this service mints, and publishes
	// this service's JWKS document. It must already be configured (see
	// auth.KeySet.Configure) before being passed here.
	Keys *auth.KeySet

	// OAuthServer implements the interactive OAuth endpoints: GET
	// /oauth/authorize, POST /oauth/firebase/complete, and POST
	// /oauth/token.
	OAuthServer *oauth.Server

	// OAuthServerConfig is the exact ServerConfig OAuthServer was built
	// with. New re-checks OAuthServerConfig.Resource against
	// config.Config.MCPAudience at assembly time (see the Task 5 ruling
	// this guards against): a mismatch here means the OAuth authorization
	// server would validate a "resource" value the MCP access-token
	// audience does not match, which would let a client obtain a token
	// this service's own bearer verifier can never accept, or worse, mint
	// tokens whose audience silently drifts from what was configured.
	OAuthServerConfig oauth.ServerConfig

	// OAuthStore backs Dynamic Client Registration (POST /oauth/register).
	OAuthStore oauth.Store

	// APIClient is the typed httpSMS API client every MCP tool calls
	// through a per-call delegation token.
	APIClient httpsms.Client

	// RedisClient backs the per-user/per-tool rate limiter. It must be a
	// standalone Redis client (redis.NewClient), never a cluster or ring
	// client.
	RedisClient redis.UniversalClient

	// APIDelegationTokenTTL bounds the lifetime of every delegation token
	// minted for a downstream httpSMS API call.
	APIDelegationTokenTTL time.Duration

	// ConfirmationTTL bounds the lifetime of a rotate_user_api_key
	// confirmation handle.
	ConfirmationTTL time.Duration

	// RateLimits configures the per-user/per-tool budgets enforced before
	// every tool call executes.
	RateLimits Limits

	// Version is this service's own build version, published in the MCP
	// Implementation.
	Version string
}

// validate returns an error naming the first missing or invalid field in
// deps, given cfg.
func (deps Dependencies) validate(cfg config.Config) error {
	switch {
	case cfg.BaseURL == nil:
		return errors.New("server: config.Config.BaseURL must not be nil")
	case deps.Keys == nil:
		return errors.New("server: Dependencies.Keys must not be nil")
	case deps.OAuthServer == nil:
		return errors.New("server: Dependencies.OAuthServer must not be nil")
	case deps.OAuthStore == nil:
		return errors.New("server: Dependencies.OAuthStore must not be nil")
	case deps.APIClient == nil:
		return errors.New("server: Dependencies.APIClient must not be nil")
	case deps.RedisClient == nil:
		return errors.New("server: Dependencies.RedisClient must not be nil")
	case deps.APIDelegationTokenTTL <= 0:
		return errors.New("server: Dependencies.APIDelegationTokenTTL must be positive")
	case deps.ConfirmationTTL <= 0:
		return errors.New("server: Dependencies.ConfirmationTTL must be positive")
	case deps.Version == "":
		return errors.New("server: Dependencies.Version must not be empty")
	case cfg.MCPAudience != canonicalMCPAudience(cfg):
		// The MCP audience is not a free-form label: it is the RFC 8707
		// resource identifier of this exact endpoint, it is what
		// RFC 9728 protected-resource metadata publishes as "resource",
		// and it is what every client sends as the "resource" parameter.
		// If a deployment overrode MCP_AUDIENCE to anything other than
		// this service's own /mcp URL, discovery would advertise one
		// value while tokens were minted for another, and no client
		// could ever obtain a token this service accepts. Fail at
		// startup rather than serve an unusable endpoint.
		return fmt.Errorf(
			"server: Config.MCPAudience %q must equal the canonical MCP endpoint URL %q (BaseURL + %q)",
			cfg.MCPAudience, canonicalMCPAudience(cfg), mcpPath,
		)
	case deps.OAuthServerConfig.Resource != cfg.MCPAudience:
		// Task 5's ruling: a wiring mismatch here must never mint
		// wrong-audience tokens. Fail fast at assembly time rather than
		// let it surface later as a confusing client-side "invalid_token"
		// rejection.
		return fmt.Errorf(
			"server: OAuth ServerConfig.Resource %q does not match Config.MCPAudience %q",
			deps.OAuthServerConfig.Resource, cfg.MCPAudience,
		)
	default:
		return deps.RateLimits.validate()
	}
}

// canonicalMCPAudience returns the only audience value this service's MCP
// endpoint may ever be configured with: its own public base URL plus the
// "/mcp" path it actually serves the endpoint on.
func canonicalMCPAudience(cfg config.Config) string {
	return strings.TrimRight(cfg.BaseURL.String(), "/") + mcpPath
}

// New assembles the httpSMS MCP service's complete HTTP surface: OAuth
// discovery/authorization/token endpoints, the stateless MCP Streamable
// HTTP handler (bearer-authenticated and rate limited), and a health
// check, wrapped in a middleware chain of request ID, panic recovery,
// secure response headers, OpenTelemetry tracing, and redacted structured
// request logging.
func New(cfg config.Config, deps Dependencies) (http.Handler, error) {
	if err := deps.validate(cfg); err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(cfg.BaseURL.String(), "/")

	mux := http.NewServeMux()

	mux.HandleFunc("GET "+healthzPath, handleHealth)
	mux.HandleFunc("GET "+healthPath, handleHealth)

	// Every public metadata route is registered without a method pattern
	// so the CORS preflight (OPTIONS) actually reaches withPublicCORS: a
	// "GET /path" pattern matches only GET and HEAD, so an OPTIONS
	// preflight against it would be answered by the mux itself with a bare
	// 405 carrying no Access-Control-* headers, which a browser-based
	// client reads as "cross-origin fetch denied". withPublicCORS performs
	// the method dispatch (GET/HEAD, OPTIONS, everything else) itself.
	protectedResourceMetadata := withPublicCORS(oauth.NewProtectedResourceMetadataHandler(baseURL))
	mux.Handle(protectedResourceMetadataPath, protectedResourceMetadata)
	mux.Handle(protectedResourceMetadataMCPPath, protectedResourceMetadata)
	mux.Handle(authorizationServerMetadataPath, withPublicCORS(oauth.NewAuthorizationServerMetadataHandler(baseURL)))
	mux.Handle(jwksPath, withPublicCORS(jwksHandler(deps.Keys)))

	mux.Handle("POST "+registerPath, oauth.NewRegistrationHandler(deps.OAuthStore))
	mux.HandleFunc("GET "+authorizePath, deps.OAuthServer.HandleAuthorize)
	mux.HandleFunc("POST "+firebaseCompletePath, deps.OAuthServer.HandleFirebaseComplete)
	mux.HandleFunc("POST "+tokenPath, deps.OAuthServer.HandleToken)

	mux.Handle(mcpPath, withNoStore(protectedMCPHandler(deps)))

	return withBaseMiddleware(deps.Logger, cfg.Environment == config.EnvironmentProduction, mux), nil
}

// withBaseMiddleware wraps next in this service's standard middleware
// chain, outermost first: request ID, panic recovery, secure response
// headers, OpenTelemetry tracing, and redacted structured request logging.
// It is the single definition of that ordering, so a test can exercise the
// exact chain production traffic passes through.
func withBaseMiddleware(logger zerolog.Logger, production bool, next http.Handler) http.Handler {
	return requestIDMiddleware(
		recoveryMiddleware(logger)(
			secureHeadersMiddleware(production)(
				otelhttp.NewHandler(
					loggingMiddleware(logger)(next),
					"httpsms-mcp",
				),
			),
		),
	)
}

// protectedMCPHandler builds the /mcp handler: the official bearer-auth
// middleware wraps the stateless MCP Streamable HTTP handler, which in
// turn enforces per-user/per-tool rate limits on every tool call through
// an MCP receiving middleware (see rateLimitMiddleware). This ordering
// (auth, then rate limit, then dispatch) means an unauthenticated caller
// never consumes rate-limit budget, and a caller's identity for rate
// limiting always comes from a token this service has already verified.
func protectedMCPHandler(deps Dependencies) http.Handler {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: implementationName, Version: deps.Version}, &mcp.ServerOptions{})
	tools.Register(mcpServer, deps.Keys, deps.APIClient, deps.APIDelegationTokenTTL, deps.OAuthStore, deps.ConfirmationTTL)

	limiter := NewToolRateLimiter(deps.RedisClient, deps.RateLimits)
	// AddReceivingMiddleware(m1, m2) composes as m1(m2(handler)), so panic
	// recovery is the outermost receiving middleware: it covers the rate
	// limiter, every SDK middleware installed beneath it (including the
	// MRTR shim), and every tool handler. Without it a single panic
	// anywhere below would unwind through the SDK's JSON-RPC dispatch and
	// kill the whole process, dropping every other in-flight request.
	mcpServer.AddReceivingMiddleware(
		recoverPanicMiddleware(deps.Logger),
		rateLimitMiddleware(limiter),
	)

	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return mcpServer },
		&mcp.StreamableHTTPOptions{
			Stateless:                    true,
			JSONResponse:                 true,
			PropagateRequestCancellation: true,
			MaxRequestBodyBytes:          maxMCPRequestBodyBytes,
			Logger:                       mcpTransportLogger(),
		},
	)

	verifier := auth.NewVerifier(deps.Keys)
	resourceMetadataURL := strings.TrimRight(deps.OAuthServerConfig.Issuer, "/") + protectedResourceMetadataPath

	bearer := mcpauth.RequireBearerToken(verifier.VerifyMCPToken, &mcpauth.RequireBearerTokenOptions{
		ResourceMetadataURL: resourceMetadataURL,
	})

	return bearer(mcpHandler)
}

// mcpTransportLogger returns the *slog.Logger passed to the Streamable HTTP
// handler for its own internal transport diagnostics (connection setup
// failures, and similar). It is deliberately independent of this service's
// zerolog request logger and is bounded to level Warn, since the SDK's
// transport logger is not designed to redact request content the way this
// service's own request logging middleware is.
func mcpTransportLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

// rateLimitMiddleware returns an mcp.Middleware enforcing limiter's
// per-user/per-tool budgets before every "tools/call" request reaches its
// tool handler. Every other MCP method (tools/list, server/discover,
// initialize, ...) passes through untouched.
//
// The caller's identity comes from the MCP access token this request's
// bearer-auth middleware has already verified (auth.PrincipalFromContext),
// never from tool input, so a caller can never spend another user's
// budget or evade its own by claiming a different identity.
func rateLimitMiddleware(limiter *ToolRateLimiter) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}

			callReq, ok := req.(*mcp.CallToolRequest)
			if !ok || callReq.Params == nil {
				return next(ctx, method, req)
			}

			principal, ok := auth.PrincipalFromContext(ctx)
			if !ok {
				// No verified principal: the bearer-auth middleware
				// already rejected this request before it could reach
				// here, or this is a call made directly against the MCP
				// server without going through HTTP (e.g. an in-process
				// test). Either way, there is no user to rate limit.
				return next(ctx, method, req)
			}

			if !chargesRateLimit(callReq) {
				return next(ctx, method, req)
			}

			if err := limiter.Allow(ctx, principal.UserID, callReq.Params.Name); err != nil {
				var rateLimitErr *RateLimitError
				if errors.As(err, &rateLimitErr) {
					return nil, rateLimitJSONRPCError(rateLimitErr)
				}
				return nil, fmt.Errorf("server: cannot check rate limit: %w", err)
			}

			return next(ctx, method, req)
		}
	}
}

// rotateUserAPIKeyToolName is the one tool whose first call is, by design,
// never an execution: it only mints a confirmation handle and asks the
// caller to confirm.
const rotateUserAPIKeyToolName = "rotate_user_api_key"

// chargesRateLimit reports whether callReq should consume rate-limit
// budget.
//
// Every tool but rotate_user_api_key is charged on every call. A
// rotate_user_api_key call that presents no confirmation at all cannot
// rotate anything: it can only mint a confirmation handle and return an
// MRTR elicitation, so charging it would let a client burn the entire
// hourly rotation budget on prompts alone and leave the user unable to
// actually complete a rotation they had already been asked to confirm.
// Only a call that presents MRTR confirmation state (RequestState or
// InputResponses) or a legacy confirmation_handle argument -- that is, a
// call that may actually execute the rotation -- is charged.
//
// Note that a client which supports neither MRTR nor elicitation can never
// reach the execution path at all (the SDK's server-side MRTR shim needs a
// live client to elicit from), so this can never be used to rotate a key
// without spending budget.
func chargesRateLimit(callReq *mcp.CallToolRequest) bool {
	if callReq.Params.Name != rotateUserAPIKeyToolName {
		return true
	}
	if callReq.Params.RequestState != "" || len(callReq.Params.InputResponses) > 0 {
		return true
	}

	var arguments struct {
		ConfirmationHandle string `json:"confirmation_handle"`
	}
	if len(callReq.Params.Arguments) > 0 {
		// A malformed arguments payload is charged: it is rejected below
		// the middleware, and refusing to charge unparseable input would
		// hand a caller a free, unmetered path into the tool dispatcher.
		if err := json.Unmarshal(callReq.Params.Arguments, &arguments); err != nil {
			return true
		}
	}

	return arguments.ConfirmationHandle != ""
}

// codeInternalError is the JSON-RPC 2.0 reserved code for an internal
// server error, returned in place of a panic that recoverPanicMiddleware
// caught.
const codeInternalError = -32603

// recoverPanicMiddleware returns an mcp.Middleware that recovers a panic
// raised anywhere beneath it -- the rate limiter, the SDK's own
// middleware, or a tool handler -- converts it into a generic JSON-RPC
// internal error, and logs it.
//
// The log line carries only the JSON-RPC method, the tool name when the
// request is a tools/call (tool names are a fixed, public set, never user
// data), the panic value, and the stack. It never carries tool arguments,
// the request body, or any header, so a panic can never leak a bearer
// token, a phone number, message content, or a freshly minted API key into
// the logs. The client is told nothing beyond "internal error".
func recoverPanicMiddleware(logger zerolog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				event := logger.Error().
					Str("mcp_method", method).
					Interface("panic", rec).
					Str("stack", string(debug.Stack()))
				if callReq, ok := req.(*mcp.CallToolRequest); ok && callReq.Params != nil {
					event = event.Str("tool", callReq.Params.Name)
				}
				event.Msg("recovered from panic in MCP method handler")

				result = nil
				err = &jsonrpc.Error{Code: codeInternalError, Message: "internal error"}
			}()

			return next(ctx, method, req)
		}
	}
}

// codeRateLimited is this service's JSON-RPC error code for a rate-limit
// rejection, drawn from the "-32000 to -32099" range JSON-RPC 2.0 reserves
// for implementation-defined server errors.
const codeRateLimited = -32029

// rateLimitErrorData is the structured "data" payload of a rate-limit
// JSON-RPC error, carrying enough for a well-behaved client to back off
// and retry automatically.
type rateLimitErrorData struct {
	Tool              string `json:"tool"`
	RetryAfterSeconds int    `json:"retry_after_seconds"`
}

// rateLimitJSONRPCError converts err into a structured MCP/JSON-RPC error
// carrying a retry-after duration, per the approved design.
func rateLimitJSONRPCError(err *RateLimitError) error {
	retryAfterSeconds := int(err.RetryAfter.Round(time.Second) / time.Second)
	if retryAfterSeconds < 1 {
		retryAfterSeconds = 1
	}

	data, marshalErr := json.Marshal(rateLimitErrorData{Tool: err.Tool, RetryAfterSeconds: retryAfterSeconds})
	if marshalErr != nil {
		data = nil
	}

	return &jsonrpc.Error{
		Code:    codeRateLimited,
		Message: err.Error(),
		Data:    data,
	}
}

// jwksHandler returns an http.HandlerFunc serving keys' JSON Web Key Set.
func jwksHandler(keys *auth.KeySet) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keys.JWKS())
	}
}

// handleHealth is this service's liveness/readiness check: a stateless MCP
// service has no per-instance state to report on, so "the process is
// serving HTTP" is a sufficient readiness signal for Cloud Run.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// requestIDMiddleware assigns every request a random request ID (reusing
// one already set by an upstream proxy, if it is well-formed and bounded),
// publishes it on the response and request context, so every later
// middleware and handler can correlate its own log lines to the same
// request.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := sanitizeRequestID(r.Header.Get(requestIDHeaderName))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		w.Header().Set(requestIDHeaderName, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sanitizeRequestID returns value when it is a safe, bounded request
// identifier this service is willing to echo back and log, or "" when the
// caller must be given a freshly generated one instead.
//
// An inbound X-Request-Id is entirely attacker-controlled: it is written
// verbatim into a response header and into every structured log line for
// the request. An oversized value is an unbounded per-request cost, and a
// value carrying control characters (CR/LF in particular) is a
// header-injection and log-injection vector. Rather than truncate or strip
// -- which would silently corrupt a legitimate upstream correlation ID --
// anything outside the accepted shape is replaced wholesale.
func sanitizeRequestID(value string) string {
	if value == "" || len(value) > maxRequestIDLength {
		return ""
	}

	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte(requestIDExtraChars, c) >= 0:
		default:
			return ""
		}
	}

	return value
}

// requestIDContextKey is the context key requestIDMiddleware publishes the
// per-request ID under.
type requestIDContextKey struct{}

// requestIDFromContext returns the request ID requestIDMiddleware
// published on ctx, or "" if none.
func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey{}).(string)
	return id
}

// recoveryMiddleware returns middleware that recovers a panic from any
// later handler, logs it (never including the request body or any
// header), and responds 500. Without this, a single handler panic would
// crash the whole process and drop every other in-flight request.
func recoveryMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error().
						Str("request_id", requestIDFromContext(r.Context())).
						Interface("panic", rec).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Msg("recovered from panic")
					w.Header().Set("Cache-Control", "no-store")
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// hstsMaxAgeSeconds is the HTTP Strict-Transport-Security max-age this
// service advertises in production: two years, the value the HSTS preload
// list requires and long enough that a returning client is never exposed
// to a downgrade window.
const hstsMaxAgeSeconds = 63072000

// secureHeadersMiddleware returns middleware setting a baseline of
// defensive HTTP response headers on every response, regardless of route.
//
// In production it additionally sets Strict-Transport-Security: this
// service mints and accepts bearer tokens, so a single plaintext request
// (a user typing the host without a scheme, a stale http:// link in a
// client's configuration) is enough to expose one. HSTS is deliberately
// not sent outside production, where the service is reached over plain
// http on localhost and a cached HSTS entry for "localhost" would break
// every other local service on that host.
func secureHeadersMiddleware(production bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			if production {
				w.Header().Set("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", hstsMaxAgeSeconds))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// loggingMiddleware returns middleware that logs one structured line per
// request: method, path, status, duration, and request ID. It never logs
// a request/response body, query string, or any header (in particular,
// never Authorization), so it can never leak a bearer token, authorization
// code, refresh token, or PKCE verifier.
func loggingMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusCapturingWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			logger.Info().
				Str("request_id", requestIDFromContext(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", sw.status).
				Dur("duration", time.Since(start)).
				Msg("http request")
		})
	}
}

// statusCapturingWriter wraps an http.ResponseWriter to record the status
// code written, for logging.
type statusCapturingWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Unwrap returns the wrapped http.ResponseWriter so
// http.NewResponseController (which the MCP SDK uses to flush every SSE
// event) can reach the real writer's optional interfaces --
// Flush/FlushError, SetReadDeadline, SetWriteDeadline, Hijack -- through
// this wrapper instead of failing with http.ErrNotSupported.
func (w *statusCapturingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Flush implements http.Flusher so a streaming (SSE) response through this
// wrapper is delivered to the client as each event is written, rather than
// buffered until the whole response completes.
func (w *statusCapturingWriter) Flush() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	//nolint:errcheck // flushing is best-effort, matching the MCP SDK.
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// withPublicCORS wraps next with a permissive but non-credentialed CORS
// policy suitable only for public discovery metadata (OAuth protected
// resource/authorization server metadata, JWKS): these documents carry no
// per-caller secret, and a client-side OAuth/MCP SDK must be able to fetch
// them cross-origin from a browser. It never sets
// Access-Control-Allow-Credentials, so this must never be applied to any
// route that reads a cookie or returns caller-specific data.
//
// It performs its own method dispatch rather than relying on the mux's
// method patterns, because a "GET /path" pattern matches only GET and HEAD:
// an OPTIONS preflight would never reach this wrapper and would be answered
// with a bare 405 carrying no Access-Control-* headers at all.
func withPublicCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writePublicCORSHeaders(w)

		switch r.Method {
		case http.MethodGet, http.MethodHead:
			next.ServeHTTP(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Max-Age", publicCORSMaxAgeSeconds)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Header().Set("Allow", publicCORSAllowedMethods)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

const (
	// publicCORSAllowedMethods are the only methods a public metadata
	// document supports: it is a static, read-only JSON document.
	publicCORSAllowedMethods = "GET, HEAD, OPTIONS"

	// publicCORSAllowedHeaders are the request headers a browser-based
	// client may send when fetching public metadata. Content-Type and
	// MCP-Protocol-Version are both non-safelisted (or, for Content-Type,
	// safelisted only for a narrow set of values), so a fetch that sets
	// either triggers a preflight that must see them echoed here or the
	// browser blocks the request outright. Authorization is included
	// because MCP clients commonly reuse one configured fetch wrapper for
	// every request to the server, including discovery -- the metadata
	// handlers themselves ignore it entirely.
	publicCORSAllowedHeaders = "Authorization, Content-Type, MCP-Protocol-Version"

	// publicCORSMaxAgeSeconds lets a browser cache the preflight result
	// for these immutable documents for 10 minutes.
	publicCORSMaxAgeSeconds = "600"
)

// writePublicCORSHeaders sets the non-credentialed CORS headers every
// public metadata response (preflight or not) carries.
func writePublicCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", publicCORSAllowedMethods)
	w.Header().Set("Access-Control-Allow-Headers", publicCORSAllowedHeaders)
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type, MCP-Protocol-Version")
	w.Header().Add("Vary", "Origin")
	w.Header().Add("Vary", "Access-Control-Request-Headers")
}

// withNoStore wraps next so every response (success or error) carries
// Cache-Control: no-store, overriding any Cache-Control value next itself
// sets (the MCP Streamable HTTP handler sets its own "no-cache,
// no-transform" value, which is not strict enough for a response that may
// carry a one-time secret such as a freshly minted phone API key or
// rotated user API key). noStoreWriter enforces this by rewriting the
// header immediately before the response is actually flushed, which is the
// only point by which every handler (including one that sets
// Cache-Control late, right before writing) has had its say.
func withNoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&noStoreWriter{ResponseWriter: w}, r)
	})
}

// noStoreWriter forces the Cache-Control/Pragma no-store headers right
// before the response's headers are actually sent, so it always wins over
// any value an inner handler set earlier.
type noStoreWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *noStoreWriter) setNoStore() {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func (w *noStoreWriter) WriteHeader(status int) {
	w.setNoStore()
	w.ResponseWriter.WriteHeader(status)
}

func (w *noStoreWriter) Write(b []byte) (int, error) {
	w.setNoStore()
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher so streaming (SSE) responses through the
// MCP handler keep working when wrapped by withNoStore. It resolves the
// flusher through http.NewResponseController rather than a direct
// http.Flusher type assertion, so it keeps working when the writer beneath
// it is itself a wrapper that only exposes Unwrap.
func (w *noStoreWriter) Flush() {
	w.setNoStore()
	//nolint:errcheck // flushing is best-effort, matching the MCP SDK.
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap returns the wrapped http.ResponseWriter so
// http.NewResponseController can reach the real writer's optional
// interfaces through this wrapper. Note that the MCP SDK's own SSE writes
// go through this wrapper's Flush (a *noStoreWriter is an http.Flusher, so
// the controller stops here), which is what guarantees the no-store header
// is set before the first event is flushed.
func (w *noStoreWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
