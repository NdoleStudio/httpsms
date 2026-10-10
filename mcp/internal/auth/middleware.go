package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

// OAuth scopes issued by this service and required by MCP tools. These are
// the same wire values published in oauth.Scopes (the source of truth for
// values presented in OAuth discovery metadata and the consent screen);
// both lists must be kept in sync.
const (
	ScopePhonesRead        = "phones:read"
	ScopeMessagesRead      = "messages:read"
	ScopeMessagesSend      = "messages:send"
	ScopePhoneAPIKeysWrite = "phone-api-keys:write"
	ScopeUserAPIKeyRotate  = "user-api-key:rotate"
)

// tokenInfoPrincipalKey and tokenInfoClientIDKey are the mcpauth.TokenInfo
// Extra map keys Verifier.VerifyMCPToken populates. They are internal to
// this package: callers must use PrincipalFromContext and RequireScope
// rather than reading mcpauth.TokenInfo.Extra directly.
const (
	tokenInfoPrincipalKey = "principal"
	tokenInfoClientIDKey  = "client_id"
)

// Verifier authenticates MCP bearer tokens presented to this service's own
// `/mcp` endpoint. Every such token is an MCP access token minted by this
// same service's KeySet (see KeySet.SignMCPAccessToken); Verifier never
// authenticates a Firebase ID token or a downstream API delegation token.
type Verifier struct {
	keys *KeySet
}

// NewVerifier returns a Verifier that authenticates MCP bearer tokens
// against keys' own signing key, issuer, and MCP audience.
func NewVerifier(keys *KeySet) *Verifier {
	return &Verifier{keys: keys}
}

// VerifyMCPToken implements mcpauth.TokenVerifier for use with
// mcpauth.RequireBearerToken. It never logs or returns raw, and the
// mcpauth.TokenInfo it returns never carries raw or any other secret
// material -- only the claims already present in an MCP access token
// (subject, scopes, expiry, client, email).
func (v *Verifier) VerifyMCPToken(_ context.Context, raw string, _ *http.Request) (*mcpauth.TokenInfo, error) {
	claims, err := v.keys.VerifyAccessToken(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid access token", mcpauth.ErrInvalidToken)
	}

	var expiration time.Time
	if claims.ExpiresAt != nil {
		expiration = claims.ExpiresAt.Time
	}

	return &mcpauth.TokenInfo{
		UserID:     claims.Subject,
		Scopes:     claims.Scopes,
		Expiration: expiration,
		Extra: map[string]any{
			tokenInfoPrincipalKey: Principal{UserID: claims.Subject, Email: claims.Email},
			tokenInfoClientIDKey:  claims.ClientID,
		},
	}, nil
}

// PrincipalFromContext returns the Principal carried by the MCP access
// token that mcpauth.RequireBearerToken (configured with a Verifier's
// VerifyMCPToken) has already validated for the current request, or false
// if ctx carries no verified token.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	info := mcpauth.TokenInfoFromContext(ctx)
	if info == nil {
		return Principal{}, false
	}

	principal, ok := info.Extra[tokenInfoPrincipalKey].(Principal)
	return principal, ok
}

// ClientIDFromContext returns the OAuth client ID carried by the MCP access
// token that mcpauth.RequireBearerToken (configured with a Verifier's
// VerifyMCPToken) has already validated for the current request, or false
// if ctx carries no verified token. Tools use this to bind sensitive
// confirmation state (see the rotate_user_api_key tool) to the exact OAuth
// client that requested the operation, not just the authenticated user.
func ClientIDFromContext(ctx context.Context) (string, bool) {
	info := mcpauth.TokenInfoFromContext(ctx)
	if info == nil {
		return "", false
	}

	clientID, ok := info.Extra[tokenInfoClientIDKey].(string)
	return clientID, ok
}

// RequireScope returns the Principal carried by ctx's already-validated MCP
// access token, or an error if ctx carries no verified token or the token's
// scopes do not include scope. It never calls the httpSMS API and never
// mints a token itself; callers use the returned Principal to mint their
// own scope-bound API delegation token for the single downstream operation
// they are about to perform.
func RequireScope(ctx context.Context, scope string) (Principal, error) {
	info := mcpauth.TokenInfoFromContext(ctx)
	if info == nil {
		return Principal{}, errors.New("auth: request has no verified MCP bearer token")
	}

	if !slices.Contains(info.Scopes, scope) {
		return Principal{}, fmt.Errorf("auth: this operation requires the %q scope", scope)
	}

	principal, ok := info.Extra[tokenInfoPrincipalKey].(Principal)
	if !ok {
		return Principal{}, errors.New("auth: verified MCP bearer token is missing its principal")
	}

	return principal, nil
}
