package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
)

const (
	testResource   = "https://mcp.httpsms.com/mcp"
	testAPIAud     = "https://api.httpsms.com"
	testIssuer     = "https://mcp.httpsms.com"
	testClientID   = "test-client-id"
	testRedirect   = "https://client.example/callback"
	testFirebaseID = "firebase-uid"
	testUserEmail  = "user@example.com"
)

// stubVerifier is a test double for auth.IdentityVerifier.
type stubVerifier struct {
	principal auth.Principal
	err       error
}

func (v stubVerifier) Verify(context.Context, string) (auth.Principal, error) {
	return v.principal, v.err
}

// newTestServerConfig returns a valid ServerConfig for tests.
func newTestServerConfig() ServerConfig {
	return ServerConfig{
		Issuer:               testIssuer,
		Resource:             testResource,
		FirebaseAPIKey:       "test-firebase-api-key",
		FirebaseAuthDomain:   "httpsms-test.firebaseapp.com",
		AuthorizationCodeTTL: time.Minute,
		AccessTokenTTL:       15 * time.Minute,
		RefreshTokenTTL:      time.Hour,
	}
}

// newTestKeySet returns a KeySet configured for signing test MCP access
// tokens, independent of any other package's test key material.
func newTestKeySet(t *testing.T) *auth.KeySet {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	keys, err := auth.NewKeySet(keyPEM, "test-key-1")
	require.NoError(t, err)
	require.NoError(t, keys.Configure(testIssuer, testResource, testAPIAud))

	return keys
}

// newTestOAuthServer builds a Server wired to store, a ClientResolver over
// the same store (so a client registered through
// store.PutDynamicClient/registerTestClient resolves correctly), a fresh
// KeySet, and verifier.
func newTestOAuthServer(t *testing.T, store Store, verifier auth.IdentityVerifier) *Server {
	t.Helper()

	resolver := NewClientResolver(http.DefaultClient, store)
	keys := newTestKeySet(t)

	server, err := NewServer(store, resolver, keys, verifier, newTestServerConfig())
	require.NoError(t, err)

	return server
}

// approvingVerifier returns an auth.IdentityVerifier that always succeeds
// with a fixed test principal.
func approvingVerifier() stubVerifier {
	return stubVerifier{principal: auth.Principal{UserID: testFirebaseID, Email: testUserEmail}}
}

// registerTestClient stores a valid DCR-style Client record under clientID
// with the given redirect URIs.
func registerTestClient(t *testing.T, store Store, clientID string, redirectURIs []string) {
	t.Helper()

	require.NoError(t, store.PutDynamicClient(context.Background(), validTestClient(clientID, redirectURIs), dynamicClientTTL))
}

// pkceChallengeFor returns the RFC 7636 S256 code_challenge for verifier.
func pkceChallengeFor(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// validAuthorizeQuery returns a fully valid GET /oauth/authorize query
// string for testClientID/testRedirect, overridable per test via extra.
func validAuthorizeQuery(extra url.Values) string {
	query := url.Values{
		"client_id":             {testClientID},
		"redirect_uri":          {testRedirect},
		"response_type":         {"code"},
		"state":                 {"state-value"},
		"code_challenge":        {pkceChallengeFor("test-verifier")},
		"code_challenge_method": {"S256"},
		"resource":              {testResource},
		"scope":                 {"phones:read messages:send"},
	}
	for key, values := range extra {
		query[key] = values
	}
	return query.Encode()
}

var transactionIDPattern = regexp.MustCompile(`name="transaction_id" value="([^"]+)"`)

// extractTransactionID pulls the transaction_id hidden field out of a
// rendered authorize.html response body.
func extractTransactionID(t *testing.T, body string) string {
	t.Helper()

	matches := transactionIDPattern.FindStringSubmatch(body)
	require.Len(t, matches, 2, "response body must contain a transaction_id hidden field: %s", body)

	return matches[1]
}

func TestNewServerRequiresStore(t *testing.T) {
	store := newClientsTestStore(t)
	resolver := NewClientResolver(http.DefaultClient, store)
	keys := newTestKeySet(t)

	_, err := NewServer(nil, resolver, keys, approvingVerifier(), newTestServerConfig())
	require.Error(t, err)
}

func TestNewServerRequiresResolver(t *testing.T) {
	store := newClientsTestStore(t)
	keys := newTestKeySet(t)

	_, err := NewServer(store, nil, keys, approvingVerifier(), newTestServerConfig())
	require.Error(t, err)
}

func TestNewServerRequiresKeySet(t *testing.T) {
	store := newClientsTestStore(t)
	resolver := NewClientResolver(http.DefaultClient, store)

	_, err := NewServer(store, resolver, nil, approvingVerifier(), newTestServerConfig())
	require.Error(t, err)
}

func TestNewServerRequiresVerifier(t *testing.T) {
	store := newClientsTestStore(t)
	resolver := NewClientResolver(http.DefaultClient, store)
	keys := newTestKeySet(t)

	_, err := NewServer(store, resolver, keys, nil, newTestServerConfig())
	require.Error(t, err)
}

func TestNewServerRejectsIncompleteConfig(t *testing.T) {
	store := newClientsTestStore(t)
	resolver := NewClientResolver(http.DefaultClient, store)
	keys := newTestKeySet(t)

	testCases := map[string]func(*ServerConfig){
		"missing issuer":                 func(c *ServerConfig) { c.Issuer = "" },
		"missing resource":               func(c *ServerConfig) { c.Resource = "" },
		"missing firebase api key":       func(c *ServerConfig) { c.FirebaseAPIKey = "" },
		"missing firebase auth domain":   func(c *ServerConfig) { c.FirebaseAuthDomain = "" },
		"non-positive authz code ttl":    func(c *ServerConfig) { c.AuthorizationCodeTTL = 0 },
		"non-positive access token ttl":  func(c *ServerConfig) { c.AccessTokenTTL = 0 },
		"non-positive refresh token ttl": func(c *ServerConfig) { c.RefreshTokenTTL = 0 },
	}

	for name, mutate := range testCases {
		t.Run(name, func(t *testing.T) {
			config := newTestServerConfig()
			mutate(&config)

			_, err := NewServer(store, resolver, keys, approvingVerifier(), config)
			require.Error(t, err)
		})
	}
}

func TestHandleAuthorizeRejectsMissingClientID(t *testing.T) {
	store := newClientsTestStore(t)
	server := newTestOAuthServer(t, store, approvingVerifier())

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize", nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_request")
}

func TestHandleAuthorizeRejectsUnresolvableClientID(t *testing.T) {
	store := newClientsTestStore(t)
	server := newTestOAuthServer(t, store, approvingVerifier())

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=never-registered", nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_client")
}

func TestHandleAuthorizeRejectsMissingRedirectURI(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id="+testClientID, nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleAuthorizeRejectsUnregisteredRedirectURI(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	query := url.Values{"client_id": {testClientID}, "redirect_uri": {"https://evil.example/callback"}}
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query.Encode(), nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestHandleAuthorizeRejectsMissingState asserts a request that has
// already been validated to have a known client and a registered
// redirect_uri, but is missing state, is rejected directly (not by
// redirecting -- there is no state to safely echo back).
func TestHandleAuthorizeRejectsMissingState(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	query := validAuthorizeQuery(url.Values{"state": {""}})
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_request")
}

// TestHandleAuthorizeRedirectsWithRFC9207IssuerOnMissingPKCE asserts a
// missing PKCE challenge is reported by redirecting back to the client
// (client_id/redirect_uri are already validated by this point) with
// error=invalid_request, the original state, and the RFC 9207 "iss"
// parameter.
func TestHandleAuthorizeRedirectsWithRFC9207IssuerOnMissingPKCE(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	query := validAuthorizeQuery(url.Values{"code_challenge": {""}})
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	location := parseLocation(t, rec)
	assert.Equal(t, "invalid_request", location.Query().Get("error"))
	assert.Equal(t, "state-value", location.Query().Get("state"))
	assert.Equal(t, testIssuer, location.Query().Get("iss"))
}

func TestHandleAuthorizeRedirectsInvalidTargetOnMissingResource(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	query := validAuthorizeQuery(url.Values{"resource": {""}})
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	location := parseLocation(t, rec)
	assert.Equal(t, "invalid_target", location.Query().Get("error"))
	assert.Equal(t, testIssuer, location.Query().Get("iss"))
}

func TestHandleAuthorizeRedirectsInvalidTargetOnWrongResource(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	query := validAuthorizeQuery(url.Values{"resource": {"https://not-mcp.example/mcp"}})
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "invalid_target", parseLocation(t, rec).Query().Get("error"))
}

func TestHandleAuthorizeRedirectsInvalidScopeOnUnknownScope(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	query := validAuthorizeQuery(url.Values{"scope": {"not-a-real-scope"}})
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "invalid_scope", parseLocation(t, rec).Query().Get("error"))
}

// TestHandleAuthorizeCreatesTransactionAndRendersFirebaseLoginPage covers
// the happy path: a fully valid request creates a persisted
// AuthorizationTransaction and renders the Firebase login page with the
// client name, requested scopes, and Firebase configuration -- and never
// puts anything sensitive in the page except the (non-secret) Firebase Web
// API key.
func TestHandleAuthorizeCreatesTransactionAndRendersFirebaseLoginPage(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+validAuthorizeQuery(nil), nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")

	body := rec.Body.String()
	assert.Contains(t, body, "Test Client")
	assert.Contains(t, body, "test-firebase-api-key")
	assert.Contains(t, body, "httpsms-test.firebaseapp.com")
	assert.Contains(t, body, "Send SMS messages on your behalf")

	transactionID := extractTransactionID(t, body)
	transaction, err := store.GetAuthorizationTransaction(context.Background(), transactionID)
	require.NoError(t, err)
	assert.Equal(t, testClientID, transaction.ClientID)
	assert.Equal(t, testRedirect, transaction.RedirectURI)
	assert.Equal(t, testResource, transaction.Resource)
	assert.Equal(t, "state-value", transaction.State)
	assert.Equal(t, []string{"phones:read", "messages:send"}, transaction.Scopes)
	assert.Equal(t, "S256", transaction.CodeChallengeMethod)
}

// parseLocation parses the Location header of a redirect response.
func parseLocation(t *testing.T, rec *httptest.ResponseRecorder) *url.URL {
	t.Helper()

	location, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)

	return location
}

// startAuthorization drives a full GET /oauth/authorize happy path and
// returns the created transaction ID, for tests of
// HandleFirebaseComplete/HandleToken that need a real, store-backed
// transaction.
func startAuthorization(t *testing.T, server *Server, extra url.Values) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+validAuthorizeQuery(extra), nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	return extractTransactionID(t, rec.Body.String())
}

func TestHandleFirebaseCompleteRejectsMissingTransactionID(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	req := httptest.NewRequest(http.MethodPost, "/oauth/firebase/complete", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.HandleFirebaseComplete(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleFirebaseCompleteRejectsUnknownTransaction(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id": {"never-issued"},
		"id_token":       {"some-token"},
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleFirebaseCompleteRedirectsAccessDeniedOnDenial(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id": {transactionID},
		"denied":         {"1"},
	})

	require.Equal(t, http.StatusFound, rec.Code)
	location := parseLocation(t, rec)
	assert.Equal(t, "access_denied", location.Query().Get("error"))
	assert.Equal(t, "state-value", location.Query().Get("state"))
	assert.Equal(t, testIssuer, location.Query().Get("iss"))
}

// TestHandleFirebaseCompleteRejectsBadIdentityToken covers Step 3 of the
// brief: a bad identity token must be rejected, and rejected directly (not
// via a client redirect) since the transaction's authenticity has not yet
// been established.
func TestHandleFirebaseCompleteRejectsBadIdentityToken(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, stubVerifier{err: auth.ErrInvalidIdentityToken})
	transactionID := startAuthorization(t, server, nil)

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"bad-token"},
		"approved_scopes": {"phones:read", "messages:send"},
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleFirebaseCompleteRejectsMissingIdentityToken(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id": {transactionID},
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestHandleFirebaseCompleteIssuesOneTimeCodeAndRedirects covers a valid
// token and approved scopes issuing a one-time code redirect, carrying
// state and the RFC 9207 "iss" parameter, and the code being redeemable
// exactly once against the Store.
func TestHandleFirebaseCompleteIssuesOneTimeCodeAndRedirects(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"good-token"},
		"approved_scopes": {"phones:read", "messages:send"},
	})

	require.Equal(t, http.StatusFound, rec.Code)
	location := parseLocation(t, rec)
	assert.Equal(t, "state-value", location.Query().Get("state"))
	assert.Equal(t, testIssuer, location.Query().Get("iss"))
	code := location.Query().Get("code")
	require.NotEmpty(t, code)

	record, err := store.ConsumeAuthorizationCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, testFirebaseID, record.UserID)
	assert.Equal(t, testUserEmail, record.Email)
	assert.Equal(t, []string{"phones:read", "messages:send"}, record.Scopes)
	assert.Equal(t, testResource, record.Resource)

	_, err = store.ConsumeAuthorizationCode(context.Background(), code)
	require.ErrorIs(t, err, ErrNotFound)
}

// TestHandleFirebaseCompleteNarrowsToApprovedScopesOnly asserts a user
// approving fewer scopes than requested results in a code bound to only
// the approved subset -- and that approving a scope outside what was
// requested cannot expand it.
func TestHandleFirebaseCompleteNarrowsToApprovedScopesOnly(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"good-token"},
		"approved_scopes": {"phones:read", "user-api-key:rotate"}, // "user-api-key:rotate" was never requested
	})

	require.Equal(t, http.StatusFound, rec.Code)
	code := parseLocation(t, rec).Query().Get("code")

	record, err := store.ConsumeAuthorizationCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, []string{"phones:read"}, record.Scopes)
}

func TestHandleFirebaseCompleteRedirectsAccessDeniedWhenNoScopeApproved(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id": {transactionID},
		"id_token":       {"good-token"},
	})

	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "access_denied", parseLocation(t, rec).Query().Get("error"))
}

// postForm posts values to handler as an application/x-www-form-urlencoded
// request and returns the recorded response.
func postForm(t *testing.T, handler http.HandlerFunc, values url.Values) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.PostForm = values
	req.Form = values
	rec := httptest.NewRecorder()

	handler(rec, req)

	return rec
}
