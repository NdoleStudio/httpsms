package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
)

// issueTestAuthorizationCode drives a full authorize -> Firebase-complete
// round trip and returns the resulting one-time authorization code, PKCE
// -bound to verifier and requesting/approving scopes (defaulting to
// "phones:read messages:send" when scopes is nil).
func issueTestAuthorizationCode(t *testing.T, server *Server, verifier string, scopes []string) string {
	t.Helper()

	if scopes == nil {
		scopes = []string{"phones:read", "messages:send"}
	}

	extra := url.Values{"code_challenge": {pkceChallengeFor(verifier)}}
	extra.Set("scope", strings.Join(scopes, " "))
	transactionID := startAuthorization(t, server, extra)

	values := url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"good-token"},
		"approved_scopes": scopes,
	}
	rec := postForm(t, server.HandleFirebaseComplete, values)
	require.Equal(t, http.StatusFound, rec.Code, "firebase complete must redirect with a code: %s", rec.Body.String())

	return parseLocation(t, rec).Query().Get("code")
}

// postToken posts values to server.HandleToken as an
// application/x-www-form-urlencoded request.
func postToken(t *testing.T, server *Server, values url.Values) *httptest.ResponseRecorder {
	t.Helper()

	return postForm(t, server.HandleToken, values)
}

// authorizationCodeGrantValues builds a valid POST /oauth/token
// authorization_code grant request body for code/verifier.
func authorizationCodeGrantValues(code, verifier string) url.Values {
	return url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {testClientID},
		"redirect_uri":  {testRedirect},
		"resource":      {testResource},
	}
}

func decodeTokenResponse(t *testing.T, rec *httptest.ResponseRecorder) tokenResponse {
	t.Helper()

	var body tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	return body
}

func decodeOAuthError(t *testing.T, rec *httptest.ResponseRecorder) oauthError {
	t.Helper()

	var body oauthError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	return body
}

// TestTokenEndpointConsumesCodeAndChecksPKCE is the literal scenario from
// the brief: a valid exchange succeeds exactly once, and replaying the
// same request afterward fails.
func TestTokenEndpointConsumesCodeAndChecksPKCE(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	values := authorizationCodeGrantValues(code, "verifier")

	response := postToken(t, server, values)
	require.Equal(t, http.StatusOK, response.Code)

	body := decodeTokenResponse(t, response)
	assert.NotEmpty(t, body.AccessToken)
	assert.Equal(t, "Bearer", body.TokenType)
	assert.NotEmpty(t, body.RefreshToken)
	assert.Equal(t, "phones:read messages:send", body.Scope)
	assert.Equal(t, int64(15*60), body.ExpiresIn)

	postAgain := postToken(t, server, values)
	require.Equal(t, http.StatusBadRequest, postAgain.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, postAgain).Error)
}

func TestTokenEndpointRejectsWrongVerifier(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "correct-verifier", nil)

	response := postToken(t, server, authorizationCodeGrantValues(code, "wrong-verifier"))
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRejectsWrongClientID(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)

	values := authorizationCodeGrantValues(code, "verifier")
	values.Set("client_id", "some-other-client")

	response := postToken(t, server, values)
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRejectsWrongRedirectURI(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)

	values := authorizationCodeGrantValues(code, "verifier")
	values.Set("redirect_uri", "https://client.example/other-callback")

	response := postToken(t, server, values)
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRejectsWrongResource(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)

	values := authorizationCodeGrantValues(code, "verifier")
	values.Set("resource", "https://not-mcp.example/mcp")

	response := postToken(t, server, values)
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRejectsMissingResource(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)

	values := authorizationCodeGrantValues(code, "verifier")
	values.Set("resource", "")

	response := postToken(t, server, values)
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRejectsMissingGrantType(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	response := postToken(t, server, url.Values{})
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRejectsUnsupportedGrantType(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	response := postToken(t, server, url.Values{"grant_type": {"client_credentials"}})
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "unsupported_grant_type", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRejectsNonPOST(t *testing.T) {
	store := newClientsTestStore(t)
	server := newTestOAuthServer(t, store, approvingVerifier())

	req := httptest.NewRequest(http.MethodGet, "/oauth/token", nil)
	rec := httptest.NewRecorder()

	server.HandleToken(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// TestTokenEndpointMintsAudienceBoundAccessTokenWithGrantedScopes verifies
// the minted access token is a real, verifiable JWT audience-bound to the
// configured MCP resource and carrying exactly the granted scopes and
// subject.
func TestTokenEndpointMintsAudienceBoundAccessTokenWithGrantedScopes(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", []string{"phones:read"})
	response := postToken(t, server, authorizationCodeGrantValues(code, "verifier"))
	require.Equal(t, http.StatusOK, response.Code)

	body := decodeTokenResponse(t, response)

	claims := new(auth.AccessClaims)
	token, err := jwt.ParseWithClaims(body.AccessToken, claims, func(*jwt.Token) (any, error) {
		return server.keys.PublicKey(), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	assert.Equal(t, testFirebaseID, claims.Subject)
	assert.Equal(t, []string{testResource}, []string(claims.Audience))
	assert.Equal(t, testIssuer, claims.Issuer)
	assert.Equal(t, []string{"phones:read"}, claims.Scopes)
	assert.Equal(t, testClientID, claims.ClientID)
}

func TestTokenEndpointRefreshRotatesTokenAndRejectsReplayOfOldToken(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	first := postToken(t, server, authorizationCodeGrantValues(code, "verifier"))
	require.Equal(t, http.StatusOK, first.Code)
	firstBody := decodeTokenResponse(t, first)

	refreshValues := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {firstBody.RefreshToken},
		"client_id":     {testClientID},
	}

	second := postToken(t, server, refreshValues)
	require.Equal(t, http.StatusOK, second.Code)
	secondBody := decodeTokenResponse(t, second)
	assert.NotEqual(t, firstBody.RefreshToken, secondBody.RefreshToken)
	assert.NotEmpty(t, secondBody.AccessToken)

	// The old refresh token must not be usable again.
	replay := postToken(t, server, refreshValues)
	require.Equal(t, http.StatusBadRequest, replay.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, replay).Error)

	// The newly rotated refresh token, however, must still work.
	rotatedAgain := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {secondBody.RefreshToken},
		"client_id":     {testClientID},
	})
	require.Equal(t, http.StatusOK, rotatedAgain.Code)
}

func TestTokenEndpointRefreshRejectsWrongClientID(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	first := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	response := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
		"client_id":     {"some-other-client"},
	})
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRefreshRejectsUnknownToken(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	response := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"never-issued"},
		"client_id":     {testClientID},
	})
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRefreshRejectsWrongResource(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	first := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	response := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
		"client_id":     {testClientID},
		"resource":      {"https://not-mcp.example/mcp"},
	})
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_target", decodeOAuthError(t, response).Error)
}

// TestTokenEndpointRefreshAllowsScopeNarrowing asserts a refresh request
// may ask for a strict subset of the originally granted scopes.
func TestTokenEndpointRefreshAllowsScopeNarrowing(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", []string{"phones:read", "messages:send"})
	first := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	response := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
		"client_id":     {testClientID},
		"scope":         {"phones:read"},
	})
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "phones:read", decodeTokenResponse(t, response).Scope)
}

// TestTokenEndpointRefreshRejectsScopeExpansion asserts a refresh request
// can never be granted a scope beyond what was originally issued.
func TestTokenEndpointRefreshRejectsScopeExpansion(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", []string{"phones:read"})
	first := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	response := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
		"client_id":     {testClientID},
		"scope":         {"phones:read messages:send"},
	})
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_scope", decodeOAuthError(t, response).Error)
}

func TestTokenEndpointRefreshRejectsMissingClientID(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	first := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	response := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
	})
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, response).Error)
}

// TestVerifyPKCERejectsNonS256Method documents that only "S256" is ever
// accepted, never "plain".
func TestVerifyPKCERejectsNonS256Method(t *testing.T) {
	assert.False(t, verifyPKCE("challenge", "plain", "challenge"))
}
