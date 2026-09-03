package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
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

// failThenSucceedVerifier fails the first Verify call and succeeds on every
// later one, modelling a user who mistypes a password (or whose ID token
// has just expired) and then signs in successfully in the same browser tab.
type failThenSucceedVerifier struct {
	calls int
}

func (v *failThenSucceedVerifier) Verify(context.Context, string) (auth.Principal, error) {
	v.calls++
	if v.calls == 1 {
		return auth.Principal{}, auth.ErrInvalidIdentityToken
	}
	return auth.Principal{UserID: testFirebaseID, Email: testUserEmail}, nil
}

// errStoreFailure is the stand-in for a Redis/infrastructure failure --
// deliberately not ErrNotFound, so it must never be reported to a client as
// an invalid grant or an invalid request.
var errStoreFailure = errors.New("oauth: redis unavailable")

// errorStore wraps a Store and forces selected methods to fail with
// errStoreFailure, so tests can distinguish "record is gone" (a client
// error) from "the store is broken" (a server error).
type errorStore struct {
	Store
	failGetTransaction     bool
	failConsumeTransaction bool
	failConsumeCode        bool
	failGetRefreshToken    bool
	failRotateRefreshToken bool
}

func (s *errorStore) GetAuthorizationTransaction(ctx context.Context, id string) (AuthorizationTransaction, error) {
	if s.failGetTransaction {
		return AuthorizationTransaction{}, errStoreFailure
	}
	return s.Store.GetAuthorizationTransaction(ctx, id)
}

func (s *errorStore) ConsumeAuthorizationTransaction(ctx context.Context, id string) (AuthorizationTransaction, error) {
	if s.failConsumeTransaction {
		return AuthorizationTransaction{}, errStoreFailure
	}
	return s.Store.ConsumeAuthorizationTransaction(ctx, id)
}

func (s *errorStore) ConsumeAuthorizationCode(ctx context.Context, code string) (AuthorizationCode, error) {
	if s.failConsumeCode {
		return AuthorizationCode{}, errStoreFailure
	}
	return s.Store.ConsumeAuthorizationCode(ctx, code)
}

func (s *errorStore) GetRefreshToken(ctx context.Context, token string) (RefreshGrant, error) {
	if s.failGetRefreshToken {
		return RefreshGrant{}, errStoreFailure
	}
	return s.Store.GetRefreshToken(ctx, token)
}

func (s *errorStore) RotateRefreshToken(ctx context.Context, oldToken string, grant RefreshGrant, ttl time.Duration) error {
	if s.failRotateRefreshToken {
		return errStoreFailure
	}
	return s.Store.RotateRefreshToken(ctx, oldToken, grant, ttl)
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

// TestHandleAuthorizeRejectsDuplicateParameters asserts a repeated
// authorization parameter is rejected outright rather than resolved by
// "first wins": two different consumers of the same URL picking different
// occurrences is a parameter-smuggling primitive.
func TestHandleAuthorizeRejectsDuplicateParameters(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	for _, name := range []string{"client_id", "redirect_uri", "state", "scope", "resource", "code_challenge"} {
		t.Run(name, func(t *testing.T) {
			query := validAuthorizeQuery(nil) + "&" + url.Values{name: {"duplicate-value"}}.Encode()
			req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
			rec := httptest.NewRecorder()

			server.HandleAuthorize(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "invalid_request")
			assert.Empty(t, rec.Header().Get("Location"), "a duplicated parameter must never produce a redirect")
		})
	}
}

// TestHandleAuthorizeRejectsMalformedCodeChallenge asserts only a
// syntactically valid S256 challenge (43 base64url characters) is accepted.
func TestHandleAuthorizeRejectsMalformedCodeChallenge(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	valid := pkceChallengeFor("test-verifier")
	require.Len(t, valid, 43)

	testCases := map[string]string{
		"empty":            "",
		"too short":        valid[:42],
		"too long":         valid + "A",
		"invalid alphabet": valid[:42] + "+",
		"padded base64":    valid[:42] + "=",
	}

	for name, challenge := range testCases {
		t.Run(name, func(t *testing.T) {
			query := validAuthorizeQuery(url.Values{"code_challenge": {challenge}})
			req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
			rec := httptest.NewRecorder()

			server.HandleAuthorize(rec, req)

			require.Equal(t, http.StatusFound, rec.Code)
			assert.Equal(t, "invalid_request", parseLocation(t, rec).Query().Get("error"))
		})
	}
}

// TestHandleAuthorizeDeduplicatesRequestedScopes asserts a repeated scope
// inside the single "scope" parameter is collapsed once, preserving the
// order the client asked in.
func TestHandleAuthorizeDeduplicatesRequestedScopes(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	query := validAuthorizeQuery(url.Values{"scope": {"messages:send phones:read messages:send phones:read"}})
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	transaction, err := store.GetAuthorizationTransaction(context.Background(), extractTransactionID(t, rec.Body.String()))
	require.NoError(t, err)
	assert.Equal(t, []string{"messages:send", "phones:read"}, transaction.Scopes)
}

// TestHandleAuthorizeSetsConsentPageProtections asserts the rendered
// consent page cannot be cached, framed, or leak its URL through a Referer
// header.
func TestHandleAuthorizeSetsConsentPageProtections(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+validAuthorizeQuery(nil), nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", rec.Header().Get("Pragma"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "frame-ancestors 'none'", rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
}

// TestHandleAuthorizeRendersEveryFirebaseProvider asserts the consent page
// offers all three identity providers the httpSMS web app supports --
// Google, GitHub, and email/password -- and that every path posts the ID
// token through the hidden form body, never a URL.
func TestHandleAuthorizeRendersEveryFirebaseProvider(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+validAuthorizeQuery(nil), nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "GoogleAuthProvider")
	assert.Contains(t, body, "GithubAuthProvider")
	assert.Contains(t, body, "signInWithEmailAndPassword")

	// Every provider funnels into the same hidden-field completion path.
	assert.Contains(t, body, `<input type="hidden" name="id_token" id="httpsms-id-token" value="">`)
	assert.Contains(t, body, `method="POST" action="/oauth/firebase/complete"`)
	assert.Contains(t, body, "completeWithUser")

	// The email/password credentials must not live inside the form that is
	// posted to this service.
	formStart := strings.Index(body, `<form id="httpsms-authorize-form"`)
	require.GreaterOrEqual(t, formStart, 0)
	formEnd := strings.Index(body[formStart:], "</form>")
	require.Greater(t, formEnd, 0)
	authorizeForm := body[formStart : formStart+formEnd]
	assert.NotContains(t, authorizeForm, `type="password"`)
	assert.NotContains(t, authorizeForm, `id="httpsms-email"`)

	// The consent page never carries a token in a URL.
	assert.NotContains(t, body, "id_token=")
}

// TestHandleAuthorizeRedirectsAreNotCacheable asserts an authorization
// error redirect -- which carries the client's state and the "iss"
// parameter -- is not storable by an intermediary or the browser cache.
func TestHandleAuthorizeRedirectsAreNotCacheable(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	query := validAuthorizeQuery(url.Values{"resource": {"https://not-mcp.example/mcp"}})
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query, nil)
	rec := httptest.NewRecorder()

	server.HandleAuthorize(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", rec.Header().Get("Pragma"))
}

// TestHandleFirebaseCompleteRejectsUnsupportedContentType asserts a body
// that is not form-encoded is rejected with the OAuth "invalid_request"
// error rather than silently parsed as an empty form.
func TestHandleFirebaseCompleteRejectsUnsupportedContentType(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	body := url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"good-token"},
		"approved_scopes": {"phones:read"},
	}.Encode()

	for _, contentType := range []string{"application/json", "text/plain", "multipart/form-data; boundary=x"} {
		t.Run(contentType, func(t *testing.T) {
			rec := postBody(t, server.HandleFirebaseComplete, contentType, body)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			failure := decodeOAuthError(t, rec)
			assert.Equal(t, "invalid_request", failure.Error)
			assert.Contains(t, failure.ErrorDescription, "Content-Type must be application/x-www-form-urlencoded")
		})
	}

	// The rejected requests never touched the transaction, and the same
	// body succeeds once it is correctly labelled.
	accepted := postBody(t, server.HandleFirebaseComplete, "application/x-www-form-urlencoded; charset=UTF-8", body)
	require.Equal(t, http.StatusFound, accepted.Code)
}

func TestHandleFirebaseCompleteRejectsMissingContentType(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())

	rec := postBody(t, server.HandleFirebaseComplete, "", "transaction_id=x")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	failure := decodeOAuthError(t, rec)
	assert.Equal(t, "invalid_request", failure.Error)
	assert.Contains(t, failure.ErrorDescription, "Content-Type must be application/x-www-form-urlencoded")
}

// TestHandleFirebaseCompleteRejectsOversizeBody asserts a body larger than
// the 64 KiB bound is refused instead of being buffered into memory.
func TestHandleFirebaseCompleteRejectsOversizeBody(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	oversize := url.Values{
		"transaction_id": {transactionID},
		"id_token":       {"good-token"},
		"padding":        {strings.Repeat("a", maxFormBodyBytes+1)},
	}

	rec := postForm(t, server.HandleFirebaseComplete, oversize)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, rec).Error)
}

// TestHandleFirebaseCompleteRejectsDuplicateParameters asserts a repeated
// single-valued parameter (here, two transaction IDs) is refused.
func TestHandleFirebaseCompleteRejectsDuplicateParameters(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id": {transactionID, "another-transaction"},
		"id_token":       {"good-token"},
	})

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, rec).Error)
}

// TestHandleFirebaseCompleteConsumesTransactionOnSuccess asserts a
// completed authorization is one-time: replaying the exact same consent
// POST cannot mint a second authorization code.
func TestHandleFirebaseCompleteConsumesTransactionOnSuccess(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	values := url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"good-token"},
		"approved_scopes": {"phones:read", "messages:send"},
	}

	first := postForm(t, server.HandleFirebaseComplete, values)
	require.Equal(t, http.StatusFound, first.Code)
	require.NotEmpty(t, parseLocation(t, first).Query().Get("code"))
	assert.Equal(t, "no-store", first.Header().Get("Cache-Control"))

	replay := postForm(t, server.HandleFirebaseComplete, values)
	require.Equal(t, http.StatusBadRequest, replay.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, replay).Error)

	_, err := store.GetAuthorizationTransaction(context.Background(), transactionID)
	require.ErrorIs(t, err, ErrNotFound)
}

// TestHandleFirebaseCompleteConsumesTransactionOnDenial asserts a denial
// is equally final: the denied transaction cannot then be approved.
func TestHandleFirebaseCompleteConsumesTransactionOnDenial(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	denial := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id": {transactionID},
		"denied":         {"1"},
	})
	require.Equal(t, http.StatusFound, denial.Code)
	assert.Equal(t, "access_denied", parseLocation(t, denial).Query().Get("error"))
	assert.Equal(t, "no-store", denial.Header().Get("Cache-Control"))

	approval := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"good-token"},
		"approved_scopes": {"phones:read"},
	})
	require.Equal(t, http.StatusBadRequest, approval.Code)
	assert.Equal(t, "invalid_request", decodeOAuthError(t, approval).Error)
}

// TestHandleFirebaseCompleteIsAtomicUnderConcurrentCompletions asserts
// that when the same consent POST arrives many times at once, exactly one
// of them can consume the transaction and issue a code -- the
// one-time-use guarantee is enforced by the store's atomic consume, not by
// request ordering.
func TestHandleFirebaseCompleteIsAtomicUnderConcurrentCompletions(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	values := url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"good-token"},
		"approved_scopes": {"phones:read", "messages:send"},
	}

	const callers = 12
	var wg sync.WaitGroup
	codes := make([]string, callers)
	statuses := make([]int, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			server.HandleFirebaseComplete(rec, req)

			statuses[index] = rec.Code
			if location, err := url.Parse(rec.Header().Get("Location")); err == nil {
				codes[index] = location.Query().Get("code")
			}
		}(i)
	}
	wg.Wait()

	issued := 0
	for index, status := range statuses {
		if status == http.StatusFound && codes[index] != "" {
			issued++
			continue
		}
		assert.Equal(t, http.StatusBadRequest, status, "a losing completion must fail closed")
	}
	assert.Equal(t, 1, issued, "exactly one concurrent completion may issue an authorization code")
}

// TestHandleFirebaseCompleteAllowsRetryAfterFailedVerification asserts a
// failed Firebase sign-in does not burn the transaction: nothing was
// issued, so the user may simply try again in the same browser tab.
func TestHandleFirebaseCompleteAllowsRetryAfterFailedVerification(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, &failThenSucceedVerifier{})
	transactionID := startAuthorization(t, server, nil)

	values := url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"first-attempt"},
		"approved_scopes": {"phones:read", "messages:send"},
	}

	failed := postForm(t, server.HandleFirebaseComplete, values)
	require.Equal(t, http.StatusUnauthorized, failed.Code)

	retry := postForm(t, server.HandleFirebaseComplete, values)
	require.Equal(t, http.StatusFound, retry.Code)
	assert.NotEmpty(t, parseLocation(t, retry).Query().Get("code"))
}

// TestHandleFirebaseCompleteRejectsEmptyVerifiedSubject asserts a verifier
// that returns success but no user ID can never lead to a code bound to an
// empty subject.
func TestHandleFirebaseCompleteRejectsEmptyVerifiedSubject(t *testing.T) {
	store := newClientsTestStore(t)
	registerTestClient(t, store, testClientID, []string{testRedirect})
	server := newTestOAuthServer(t, store, stubVerifier{principal: auth.Principal{Email: testUserEmail}})
	transactionID := startAuthorization(t, server, nil)

	rec := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"subjectless-token"},
		"approved_scopes": {"phones:read"},
	})

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "access_denied", decodeOAuthError(t, rec).Error)

	// No code was issued, and the transaction was not consumed.
	_, err := store.GetAuthorizationTransaction(context.Background(), transactionID)
	require.NoError(t, err)
}

// TestHandleFirebaseCompleteReturnsServerErrorOnStoreFailure asserts an
// infrastructure failure is reported as a 500 "server_error", never as a
// client-side "invalid_request".
func TestHandleFirebaseCompleteReturnsServerErrorOnStoreFailure(t *testing.T) {
	base := newClientsTestStore(t)
	registerTestClient(t, base, testClientID, []string{testRedirect})
	failing := &errorStore{Store: base}
	server := newTestOAuthServer(t, failing, approvingVerifier())
	transactionID := startAuthorization(t, server, nil)

	failing.failGetTransaction = true
	lookupFailure := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id": {transactionID},
		"id_token":       {"good-token"},
	})
	require.Equal(t, http.StatusInternalServerError, lookupFailure.Code)
	assert.Equal(t, "server_error", decodeOAuthError(t, lookupFailure).Error)

	failing.failGetTransaction = false
	failing.failConsumeTransaction = true
	consumeFailure := postForm(t, server.HandleFirebaseComplete, url.Values{
		"transaction_id":  {transactionID},
		"id_token":        {"good-token"},
		"approved_scopes": {"phones:read"},
	})
	require.Equal(t, http.StatusInternalServerError, consumeFailure.Code)
	assert.Equal(t, "server_error", decodeOAuthError(t, consumeFailure).Error)
}

// postForm posts values to handler as a real application/x-www-form-urlencoded
// request: the body is encoded on the wire and parsed by the handler's own
// ParseForm call, so body-size and Content-Type enforcement are exercised
// exactly as they are in production (never by pre-populating PostForm).
func postForm(t *testing.T, handler http.HandlerFunc, values url.Values) *httptest.ResponseRecorder {
	t.Helper()

	return postBody(t, handler, "application/x-www-form-urlencoded", values.Encode())
}

// postBody posts a raw body with an explicit Content-Type to handler.
func postBody(t *testing.T, handler http.HandlerFunc, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Del("Content-Type")
	}
	rec := httptest.NewRecorder()

	handler(rec, req)

	return rec
}
