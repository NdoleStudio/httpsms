package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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

	// Replaying a consumed refresh token is reuse, not a harmless
	// mistake: the whole token family is revoked, so the replacement the
	// legitimate client is holding stops working too. This is the only
	// response that makes a stolen refresh token useless to an attacker
	// who also has to race the real client.
	rotatedAgain := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {secondBody.RefreshToken},
		"client_id":     {testClientID},
	})
	require.Equal(t, http.StatusBadRequest, rotatedAgain.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, rotatedAgain).Error)
}

// TestTokenEndpointRefreshRotatesRepeatedlyWithoutReplay asserts the happy
// path is unaffected by reuse detection: a client that always presents its
// most recent refresh token can rotate indefinitely.
func TestTokenEndpointRefreshRotatesRepeatedlyWithoutReplay(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	current := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	seen := map[string]bool{current.RefreshToken: true}
	for i := range 5 {
		response := postToken(t, server, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {current.RefreshToken},
			"client_id":     {testClientID},
		})
		require.Equal(t, http.StatusOK, response.Code, "rotation %d must succeed", i+1)

		current = decodeTokenResponse(t, response)
		require.NotEmpty(t, current.RefreshToken)
		assert.False(t, seen[current.RefreshToken], "every rotation must mint a fresh refresh token")
		seen[current.RefreshToken] = true
	}
}

// TestTokenEndpointRefreshReplayRevokesNewestFamilyMember asserts that a
// token replayed several rotations later still revokes whatever the
// family's active token is at that moment, not the token that directly
// succeeded the replayed one.
func TestTokenEndpointRefreshReplayRevokesNewestFamilyMember(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	current := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))
	leaked := current.RefreshToken

	for range 3 {
		response := postToken(t, server, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {current.RefreshToken},
			"client_id":     {testClientID},
		})
		require.Equal(t, http.StatusOK, response.Code)
		current = decodeTokenResponse(t, response)
	}

	// An attacker replays the very first refresh token of the lineage.
	replay := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {leaked},
		"client_id":     {testClientID},
	})
	require.Equal(t, http.StatusBadRequest, replay.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, replay).Error)

	// The legitimate client's newest token is now revoked.
	afterRevocation := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {current.RefreshToken},
		"client_id":     {testClientID},
	})
	require.Equal(t, http.StatusBadRequest, afterRevocation.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, afterRevocation).Error)
}

// TestTokenEndpointRefreshReuseIsIndistinguishableFromUnknownToken asserts
// the token endpoint never tells a caller whether it hit reuse detection
// or simply presented a token that was never issued: both are
// "invalid_grant" with the same description.
func TestTokenEndpointRefreshReuseIsIndistinguishableFromUnknownToken(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	first := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	require.Equal(t, http.StatusOK, postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
		"client_id":     {testClientID},
	}).Code)

	reuse := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
		"client_id":     {testClientID},
	})
	unknown := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"never-issued-refresh-token"},
		"client_id":     {testClientID},
	})

	require.Equal(t, unknown.Code, reuse.Code)
	assert.Equal(t, decodeOAuthError(t, unknown), decodeOAuthError(t, reuse))
}

// TestTokenEndpointRefreshReuseRevokesOnlyTheCompromisedFamily asserts a
// second, independent grant for the same user and client survives another
// family's revocation.
func TestTokenEndpointRefreshReuseRevokesOnlyTheCompromisedFamily(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	compromisedCode := issueTestAuthorizationCode(t, server, "verifier", nil)
	compromised := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(compromisedCode, "verifier")))

	healthyCode := issueTestAuthorizationCode(t, server, "verifier", nil)
	healthy := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(healthyCode, "verifier")))

	require.Equal(t, http.StatusOK, postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {compromised.RefreshToken},
		"client_id":     {testClientID},
	}).Code)

	// Replay the compromised family's consumed token.
	require.Equal(t, http.StatusBadRequest, postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {compromised.RefreshToken},
		"client_id":     {testClientID},
	}).Code)

	// The unrelated grant is untouched.
	require.Equal(t, http.StatusOK, postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {healthy.RefreshToken},
		"client_id":     {testClientID},
	}).Code)
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

// TestTokenEndpointRejectsUnsupportedContentType asserts the token
// endpoint only accepts form-encoded bodies (RFC 6749 Section 4.1.3), and
// reports anything else as "invalid_request".
func TestTokenEndpointRejectsUnsupportedContentType(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	body := authorizationCodeGrantValues(code, "verifier").Encode()

	for _, contentType := range []string{"application/json", "text/plain", "multipart/form-data; boundary=x"} {
		t.Run(contentType, func(t *testing.T) {
			rec := postBody(t, server.HandleToken, contentType, body)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			failure := decodeOAuthError(t, rec)
			assert.Equal(t, "invalid_request", failure.Error)
			assert.Contains(t, failure.ErrorDescription, "Content-Type must be application/x-www-form-urlencoded")
		})
	}

	// The rejected requests must not have consumed the code: the same body
	// still succeeds once it is correctly labelled.
	success := postBody(t, server.HandleToken, "application/x-www-form-urlencoded; charset=UTF-8", body)
	require.Equal(t, http.StatusOK, success.Code)
}

func TestTokenEndpointRejectsMissingContentType(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	rec := postBody(t, server.HandleToken, "", "grant_type=authorization_code")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	failure := decodeOAuthError(t, rec)
	assert.Equal(t, "invalid_request", failure.Error)
	assert.Contains(t, failure.ErrorDescription, "Content-Type must be application/x-www-form-urlencoded")
}

// TestTokenEndpointRejectsOversizeBody asserts a body larger than the
// 64 KiB bound is refused rather than buffered.
func TestTokenEndpointRejectsOversizeBody(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	values := authorizationCodeGrantValues(code, "verifier")
	values.Set("padding", strings.Repeat("a", maxFormBodyBytes+1))

	rec := postToken(t, server, values)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, rec).Error)
}

// TestTokenEndpointRejectsDuplicateParameters asserts a repeated token
// parameter is refused instead of resolved by "first wins".
func TestTokenEndpointRejectsDuplicateParameters(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	values := authorizationCodeGrantValues(code, "verifier")
	values["client_id"] = []string{testClientID, "some-other-client"}

	rec := postToken(t, server, values)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, rec).Error)
}

// TestTokenEndpointReturnsServerErrorWhenCodeLookupFails asserts a Redis
// failure while consuming an authorization code is a 500 "server_error",
// not an "invalid_grant" that would make a client discard a valid code.
func TestTokenEndpointReturnsServerErrorWhenCodeLookupFails(t *testing.T) {
	base := newClientsTestStore(t)
	registerTestClient(t, base, testClientID, []string{testRedirect})
	failing := &errorStore{Store: base}
	server := newTestOAuthServer(t, failing, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	failing.failConsumeCode = true

	rec := postToken(t, server, authorizationCodeGrantValues(code, "verifier"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "server_error", decodeOAuthError(t, rec).Error)
}

// TestTokenEndpointRefreshDistinguishesMissingTokenFromStoreFailure
// asserts an unknown/expired refresh token is "invalid_grant" (400), while
// a Redis failure looking one up is "server_error" (500).
func TestTokenEndpointRefreshDistinguishesMissingTokenFromStoreFailure(t *testing.T) {
	base := newClientsTestStore(t)
	registerTestClient(t, base, testClientID, []string{testRedirect})
	failing := &errorStore{Store: base}
	server := newTestOAuthServer(t, failing, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	first := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	refreshValues := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
		"client_id":     {testClientID},
	}

	failing.failGetRefreshToken = true
	lookupFailure := postToken(t, server, refreshValues)
	require.Equal(t, http.StatusInternalServerError, lookupFailure.Code)
	assert.Equal(t, "server_error", decodeOAuthError(t, lookupFailure).Error)

	failing.failGetRefreshToken = false
	failing.failRotateRefreshToken = true
	rotateFailure := postToken(t, server, refreshValues)
	require.Equal(t, http.StatusInternalServerError, rotateFailure.Code)
	assert.Equal(t, "server_error", decodeOAuthError(t, rotateFailure).Error)

	// A genuinely unknown token remains a client error.
	failing.failRotateRefreshToken = false
	unknown := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"never-issued"},
		"client_id":     {testClientID},
	})
	require.Equal(t, http.StatusBadRequest, unknown.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, unknown).Error)
}

// TestTokenEndpointRefreshReuseCheckFailureIsServerError asserts a Redis
// failure during reuse detection is reported as "server_error" (500)
// rather than being swallowed into an "invalid_grant": the server could
// not determine whether a family needed revoking, so it must not pretend
// the outcome is the client's fault.
func TestTokenEndpointRefreshReuseCheckFailureIsServerError(t *testing.T) {
	base := newClientsTestStore(t)
	registerTestClient(t, base, testClientID, []string{testRedirect})
	failing := &errorStore{Store: base}
	server := newTestOAuthServer(t, failing, approvingVerifier())

	failing.failDetectReuse = true

	response := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"never-issued"},
		"client_id":     {testClientID},
	})

	require.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Equal(t, "server_error", decodeOAuthError(t, response).Error)
}

// TestTokenEndpointRefreshConcurrentRotationRevokesFamily asserts a lost
// rotation race is treated as reuse: two parties presented the same
// refresh token, so the winner's replacement is revoked too.
func TestTokenEndpointRefreshConcurrentRotationRevokesFamily(t *testing.T) {
	base := newClientsTestStore(t)
	registerTestClient(t, base, testClientID, []string{testRedirect})
	racing := &raceRotateStore{Store: base}
	server := newTestOAuthServer(t, racing, approvingVerifier())

	code := issueTestAuthorizationCode(t, server, "verifier", nil)
	first := decodeTokenResponse(t, postToken(t, server, authorizationCodeGrantValues(code, "verifier")))

	refreshValues := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first.RefreshToken},
		"client_id":     {testClientID},
	}

	// racing.stealRotation makes the very next rotation lose the race by
	// consuming the old token itself first, exactly as a concurrent
	// legitimate request would.
	racing.stealRotation = true
	lost := postToken(t, server, refreshValues)
	require.Equal(t, http.StatusBadRequest, lost.Code)
	assert.Equal(t, "invalid_grant", decodeOAuthError(t, lost).Error)

	// The token the racing winner minted is revoked along with the family.
	racing.stealRotation = false
	afterRevocation := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {racing.stolenToken},
		"client_id":     {testClientID},
	})
	require.Equal(t, http.StatusBadRequest, afterRevocation.Code)
}

// raceRotateStore simulates a concurrent rotation: when stealRotation is
// set, the next RotateRefreshToken call first performs its own rotation of
// the same old token (recording the replacement in stolenToken) and then
// lets the real call run, which now loses the race and sees ErrNotFound.
type raceRotateStore struct {
	Store
	stealRotation bool
	stolenToken   string
}

func (s *raceRotateStore) RotateRefreshToken(ctx context.Context, oldToken string, grant RefreshGrant, ttl time.Duration) error {
	if s.stealRotation {
		s.stealRotation = false
		s.stolenToken = "stolen-" + grant.Token
		stolen := grant
		stolen.Token = s.stolenToken
		if err := s.Store.RotateRefreshToken(ctx, oldToken, stolen, ttl); err != nil {
			return err
		}
	}
	return s.Store.RotateRefreshToken(ctx, oldToken, grant, ttl)
}
