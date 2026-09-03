package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
)

const testResourceMetadataURL = "https://mcp.httpsms.com/.well-known/oauth-protected-resource"

// newMiddlewareTestServer builds an http.Handler protected by
// mcpauth.RequireBearerToken(verifier.VerifyMCPToken, ...), whose inner
// handler reports the verified mcpauth.TokenInfo (or its absence) as JSON,
// so tests can assert on both the HTTP-level response and what ends up in
// the request context.
func newMiddlewareTestServer(t *testing.T, keys *auth.KeySet, requiredScopes []string) *httptest.Server {
	t.Helper()

	verifier := auth.NewVerifier(keys)
	middleware := mcpauth.RequireBearerToken(verifier.VerifyMCPToken, &mcpauth.RequireBearerTokenOptions{
		ResourceMetadataURL: testResourceMetadataURL,
		Scopes:              requiredScopes,
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := mcpauth.TokenInfoFromContext(r.Context())
		require.NotNil(t, info, "middleware must store TokenInfo in the request context on success")

		principal, ok := auth.PrincipalFromContext(r.Context())
		require.True(t, ok, "auth.PrincipalFromContext must find the principal the middleware stored")

		clientID, ok := auth.ClientIDFromContext(r.Context())
		require.True(t, ok, "auth.ClientIDFromContext must find the client ID the middleware stored")

		w.Header().Set("X-Test-User-ID", info.UserID)
		w.Header().Set("X-Test-Principal-Email", principal.Email)
		w.Header().Set("X-Test-Client-ID", clientID)
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(middleware(inner))
}

func TestRequireBearerTokenRejectsMissingToken(t *testing.T) {
	keys := newTestKeySet(t)
	server := newMiddlewareTestServer(t, keys, nil)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("WWW-Authenticate"), testResourceMetadataURL)
}

func TestRequireBearerTokenRejectsMalformedAuthorizationHeader(t *testing.T) {
	keys := newTestKeySet(t)
	server := newMiddlewareTestServer(t, keys, nil)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "not-a-bearer-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRequireBearerTokenRejectsExpiredToken(t *testing.T) {
	keys := newTestKeySet(t)
	server := newMiddlewareTestServer(t, keys, nil)
	defer server.Close()

	raw, err := keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUserID}, "client", []string{auth.ScopePhonesRead}, time.Nanosecond)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	resp := doBearerRequest(t, server.URL, raw)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("WWW-Authenticate"), testResourceMetadataURL)
}

func TestRequireBearerTokenRejectsWrongAudienceToken(t *testing.T) {
	keys := newTestKeySet(t)
	server := newMiddlewareTestServer(t, keys, nil)
	defer server.Close()

	// An API delegation token is audienced to the API, not the MCP
	// endpoint, and must never authenticate an MCP request.
	raw, err := keys.SignAPIDelegationToken(auth.Principal{UserID: testFirebaseUserID}, []string{auth.ScopePhonesRead}, http.MethodGet, "/v1/phones", time.Minute)
	require.NoError(t, err)

	resp := doBearerRequest(t, server.URL, raw)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRequireBearerTokenRejectsInvalidToken(t *testing.T) {
	keys := newTestKeySet(t)
	server := newMiddlewareTestServer(t, keys, nil)
	defer server.Close()

	resp := doBearerRequest(t, server.URL, "this-is-not-a-jwt")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("WWW-Authenticate"), testResourceMetadataURL)
}

func TestRequireBearerTokenRejectsInsufficientScope(t *testing.T) {
	keys := newTestKeySet(t)
	server := newMiddlewareTestServer(t, keys, []string{auth.ScopeMessagesSend})
	defer server.Close()

	raw, err := keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUserID}, "client", []string{auth.ScopePhonesRead}, time.Minute)
	require.NoError(t, err)

	resp := doBearerRequest(t, server.URL, raw)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRequireBearerTokenAcceptsValidTokenAndStoresTokenInfo(t *testing.T) {
	keys := newTestKeySet(t)
	server := newMiddlewareTestServer(t, keys, []string{auth.ScopePhonesRead})
	defer server.Close()

	raw, err := keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUserID, Email: testUserEmail}, "client", []string{auth.ScopePhonesRead, auth.ScopeMessagesRead}, time.Minute)
	require.NoError(t, err)

	resp := doBearerRequest(t, server.URL, raw)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, testFirebaseUserID, resp.Header.Get("X-Test-User-ID"))
	assert.Equal(t, testUserEmail, resp.Header.Get("X-Test-Principal-Email"))
	assert.Equal(t, "client", resp.Header.Get("X-Test-Client-ID"))
}

func doBearerRequest(t *testing.T, url string, token string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func TestPrincipalFromContextReturnsFalseWithoutToken(t *testing.T) {
	_, ok := auth.PrincipalFromContext(t.Context())
	assert.False(t, ok)
}

func TestClientIDFromContextReturnsFalseWithoutToken(t *testing.T) {
	_, ok := auth.ClientIDFromContext(t.Context())
	assert.False(t, ok)
}

func TestRequireScopeReturnsErrorWithoutToken(t *testing.T) {
	_, err := auth.RequireScope(t.Context(), auth.ScopePhonesRead)
	require.Error(t, err)
}
