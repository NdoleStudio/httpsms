package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/NdoleStudio/stacktrace"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// ErrCodeInvalidToken is thrown when a delegated MCP token cannot be verified.
	ErrCodeInvalidToken = stacktrace.ErrorCode(3000)

	// ErrCodeInsufficientScope is thrown when a delegated MCP token is valid but does not carry the required scope.
	ErrCodeInsufficientScope = stacktrace.ErrorCode(3001)

	// ErrCodeOperationDenied is thrown when a delegated MCP token is valid but is not bound to the requested operation.
	ErrCodeOperationDenied = stacktrace.ErrorCode(3002)
)

// mcpDelegatedRoute is an API operation that can be authorized with a delegated MCP token.
type mcpDelegatedRoute struct {
	method   string
	segments []string
	scope    string
}

// mcpDelegatedRoutes are the only API operations a delegated MCP token may authorize. Every
// entry corresponds to a tool in the MCP tool catalog. "*" matches any single path segment,
// which is required for the primary-API-key rotation route which is bound to the authenticated
// user's ID.
var mcpDelegatedRoutes = []mcpDelegatedRoute{
	{method: http.MethodGet, segments: []string{"v1", "phones"}, scope: "phones:read"},
	{method: http.MethodPost, segments: []string{"v1", "messages", "send"}, scope: "messages:send"},
	{method: http.MethodGet, segments: []string{"v1", "message-threads"}, scope: "messages:read"},
	{method: http.MethodGet, segments: []string{"v1", "messages"}, scope: "messages:read"},
	{method: http.MethodGet, segments: []string{"v1", "messages", "incoming"}, scope: "messages:read"},
	{method: http.MethodPost, segments: []string{"v1", "phone-api-keys"}, scope: "phone-api-keys:write"},
	{method: http.MethodDelete, segments: []string{"v1", "users", "*", "api-keys"}, scope: "user-api-key:rotate"},
}

// requiredMCPDelegatedScope returns the downstream API scope required to authorize method/path
// with a delegated MCP token, and whether method/path is an approved MCP API operation at all.
func requiredMCPDelegatedScope(method string, path string) (string, bool) {
	requestSegments := splitMCPPath(path)
	for _, route := range mcpDelegatedRoutes {
		if route.method != method {
			continue
		}
		if matchMCPPathSegments(route.segments, requestSegments) {
			return route.scope, true
		}
	}
	return "", false
}

func splitMCPPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func matchMCPPathSegments(pattern []string, actual []string) bool {
	if len(pattern) != len(actual) {
		return false
	}
	for i, segment := range pattern {
		if segment == "*" {
			continue
		}
		if segment != actual[i] {
			return false
		}
	}
	return true
}

// containsAllScopes returns true if every scope in required is present in granted.
func containsAllScopes(granted []string, required []string) bool {
	grantedSet := make(map[string]struct{}, len(granted))
	for _, scope := range granted {
		grantedSet[scope] = struct{}{}
	}
	for _, scope := range required {
		if _, ok := grantedSet[scope]; !ok {
			return false
		}
	}
	return true
}

// MCPTokenVerifierConfig configures a MCPTokenVerifier.
type MCPTokenVerifierConfig struct {
	// Issuer is the only issuer trusted for delegated MCP tokens.
	Issuer string

	// Audience is the audience delegated MCP tokens must carry.
	Audience string

	// JWKSURL is the JWKS endpoint used to verify delegated MCP token signatures.
	JWKSURL string

	// HTTPClient is used to fetch the JWKS document. http.DefaultClient is used when nil.
	HTTPClient *http.Client

	// CacheTTL is how long a fetched JWKS document is cached. Defaults to 15 minutes.
	CacheTTL time.Duration
}

// MCPTokenVerifier validates delegated MCP API JWTs minted by the hosted MCP service.
type MCPTokenVerifier struct {
	issuer   string
	audience string
	jwks     *mcpJWKSCache
}

// NewMCPTokenVerifier creates a new MCPTokenVerifier. It returns an error if config is missing
// any of the required Issuer, Audience, or JWKSURL values.
func NewMCPTokenVerifier(config MCPTokenVerifierConfig) (*MCPTokenVerifier, error) {
	if config.Issuer == "" || config.Audience == "" || config.JWKSURL == "" {
		return nil, stacktrace.NewErrorWithCodef(ErrCodeInvalidToken, "MCP token verifier requires an issuer, audience, and JWKS URL")
	}

	return &MCPTokenVerifier{
		issuer:   config.Issuer,
		audience: config.Audience,
		jwks:     newMCPJWKSCache(config.JWKSURL, config.HTTPClient, config.CacheTTL),
	}, nil
}

// VerifyRequest verifies that raw is a delegated MCP token that is valid, unexpired, issued by
// the configured issuer for the configured audience, and bound to the exact method and path of
// the current request with the scope that operation requires.
func (verifier *MCPTokenVerifier) VerifyRequest(ctx context.Context, raw string, method string, path string) (*MCPClaims, error) {
	claims := new(MCPClaims)
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		verifier.keyfunc(ctx),
		jwt.WithIssuer(verifier.issuer),
		jwt.WithAudience(verifier.audience),
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
		return nil, stacktrace.NewErrorWithCodef(ErrCodeOperationDenied, "MCP delegated token is not valid for this API operation")
	}

	if !containsAllScopes(claims.Scopes, []string{requiredScope}) {
		return nil, stacktrace.NewErrorWithCodef(ErrCodeInsufficientScope, "MCP delegated token has insufficient scope")
	}

	return claims, nil
}

// keyfunc returns a jwt.Keyfunc that resolves the RSA public key matching the token's "kid"
// header from the cached JWKS document.
func (verifier *MCPTokenVerifier) keyfunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, stacktrace.NewErrorWithCodef(ErrCodeInvalidToken, "unexpected MCP delegated token signing method [%v]", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, stacktrace.NewErrorWithCodef(ErrCodeInvalidToken, "MCP delegated token has no [kid] header")
		}

		return verifier.jwks.key(ctx, kid)
	}
}
