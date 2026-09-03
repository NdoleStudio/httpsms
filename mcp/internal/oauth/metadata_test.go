package oauth_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
)

const testMCPBaseURL = "https://mcp.httpsms.com"

func TestProtectedResourceMetadataHandlerServesExactFields(t *testing.T) {
	handler := oauth.NewProtectedResourceMetadataHandler(testMCPBaseURL)

	req := httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	require.Equal(t, 200, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.Equal(t, "https://mcp.httpsms.com/mcp", body["resource"])
	assert.Equal(t, []any{"https://mcp.httpsms.com"}, body["authorization_servers"])
	assert.Equal(t, []any{
		"phones:read",
		"messages:read",
		"messages:send",
		"phone-api-keys:write",
		"user-api-key:rotate",
	}, body["scopes_supported"])

	// No other top-level fields are permitted by RFC 9728 for this
	// service's minimal, exact document.
	assert.Len(t, body, 3)
}

func TestProtectedResourceMetadataHandlerTrimsTrailingSlash(t *testing.T) {
	handler := oauth.NewProtectedResourceMetadataHandler(testMCPBaseURL + "/")

	req := httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "https://mcp.httpsms.com/mcp", body["resource"])
}

func TestAuthorizationServerMetadataHandlerServesExactFields(t *testing.T) {
	handler := oauth.NewAuthorizationServerMetadataHandler(testMCPBaseURL)

	req := httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	require.Equal(t, 200, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.Equal(t, "https://mcp.httpsms.com", body["issuer"])
	assert.Equal(t, "https://mcp.httpsms.com/oauth/authorize", body["authorization_endpoint"])
	assert.Equal(t, "https://mcp.httpsms.com/oauth/token", body["token_endpoint"])
	assert.Equal(t, "https://mcp.httpsms.com/oauth/register", body["registration_endpoint"])
	assert.Equal(t, "https://mcp.httpsms.com/.well-known/jwks.json", body["jwks_uri"])
	assert.Equal(t, []any{"code"}, body["response_types_supported"])
	assert.Equal(t, []any{"authorization_code", "refresh_token"}, body["grant_types_supported"])
	assert.Equal(t, []any{"S256"}, body["code_challenge_methods_supported"])
	assert.Equal(t, []any{
		"phones:read",
		"messages:read",
		"messages:send",
		"phone-api-keys:write",
		"user-api-key:rotate",
	}, body["scopes_supported"])
	assert.Equal(t, true, body["client_id_metadata_document_supported"])

	assert.Len(t, body, 10)
}

func TestScopesConstantOrderIsStable(t *testing.T) {
	// Both metadata documents and the future consent screen depend on this
	// exact, fixed order; a reordering would silently change the scopes
	// list presented to users.
	assert.Equal(t, []string{
		"phones:read",
		"messages:read",
		"messages:send",
		"phone-api-keys:write",
		"user-api-key:rotate",
	}, oauth.Scopes)
}
