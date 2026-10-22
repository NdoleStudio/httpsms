package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingJWKSServer serves the JWKS document for kid/publicKey and counts every request it
// receives, so a test can assert exactly how many outbound fetches a verifier performed.
func countingJWKSServer(t *testing.T, kid string, publicKey *rsa.PublicKey, count *atomic.Int64) *httptest.Server {
	t.Helper()

	handler := testJWKSHandler(t, kid, publicKey)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	return server
}

// newBoundedTestVerifier builds a verifier whose JWKS refreshes are throttled to
// minRefreshInterval.
func newBoundedTestVerifier(t *testing.T, jwksURL string, jwksClient *http.Client, minRefreshInterval time.Duration) *MCPTokenVerifier {
	t.Helper()

	verifier, err := NewMCPTokenVerifier(MCPTokenVerifierConfig{
		Issuer:             "https://mcp.httpsms.com",
		Audience:           "https://api.httpsms.com",
		JWKSURL:            jwksURL,
		HTTPClient:         jwksClient,
		MinRefreshInterval: minRefreshInterval,
	})
	require.NoError(t, err)

	return verifier
}

// TestMCPTokenVerifierVerifyRequest_NonMCPBearerTokensPerformNoJWKSRequests proves the cheap
// unverified-issuer prefilter: the delegated MCP middleware runs before Firebase bearer
// authentication, so Firebase ID tokens (and any other non-MCP bearer value) reach this
// verifier first. None of them may cause a single outbound JWKS fetch.
func TestMCPTokenVerifierVerifyRequest_NonMCPBearerTokensPerformNoJWKSRequests(t *testing.T) {
	privateKey, publicKey := testRSAKey(t)

	var requestCount atomic.Int64
	jwks := countingJWKSServer(t, "test-key", publicKey, &requestCount)

	verifier := newBoundedTestVerifier(t, jwks.URL, jwks.Client(), time.Nanosecond)

	firebaseClaims := testMCPClaims("firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages")
	firebaseClaims.Issuer = "https://securetoken.google.com/httpsms-test"
	firebaseToken := signDelegatedToken(t, privateKey, "firebase-kid", firebaseClaims)

	noIssuerClaims := testMCPClaims("firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages")
	noIssuerClaims.Issuer = ""
	noIssuerToken := signDelegatedToken(t, privateKey, "unknown-kid", noIssuerClaims)

	tests := []struct {
		name string
		raw  string
	}{
		{name: "firebase id token", raw: firebaseToken},
		{name: "token without an issuer", raw: noIssuerToken},
		{name: "opaque api key", raw: "not-a-json-web-token"},
		{name: "empty token", raw: ""},
		{name: "malformed jwt", raw: "aaa.bbb.ccc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := verifier.VerifyRequest(context.Background(), tt.raw, http.MethodGet, "/v1/messages")

			require.Error(t, err)
			require.Nil(t, claims)
		})
	}

	assert.Equal(t, int64(0), requestCount.Load(), "non-MCP bearer tokens must never reach the JWKS endpoint")
}

// TestMCPTokenVerifierVerifyRequest_BoundsRefreshesAcrossManyUnknownKids proves that a flood of
// otherwise well-formed MCP-issuer tokens carrying random unknown "kid" headers cannot amplify
// into one outbound JWKS fetch per request.
func TestMCPTokenVerifierVerifyRequest_BoundsRefreshesAcrossManyUnknownKids(t *testing.T) {
	privateKey, publicKey := testRSAKey(t)

	var requestCount atomic.Int64
	jwks := countingJWKSServer(t, "test-key", publicKey, &requestCount)

	verifier := newBoundedTestVerifier(t, jwks.URL, jwks.Client(), time.Minute)

	for i := 0; i < 100; i++ {
		raw := signDelegatedToken(t, privateKey, fmt.Sprintf("unknown-kid-%d", i), testMCPClaims(
			"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
		))

		claims, err := verifier.VerifyRequest(context.Background(), raw, http.MethodGet, "/v1/messages")
		require.Error(t, err)
		require.Nil(t, claims)
	}

	assert.Equal(t, int64(1), requestCount.Load(), "unknown kids must cause at most one JWKS fetch per refresh interval")
}

// TestMCPJWKSCacheCollapsesConcurrentRefreshes proves that callers arriving while a refresh is
// already in flight share it instead of each starting their own fetch. Throttling is disabled
// here so the single fetch is attributable to collapsing alone.
func TestMCPJWKSCacheCollapsesConcurrentRefreshes(t *testing.T) {
	_, publicKey := testRSAKey(t)

	var requestCount atomic.Int64
	handler := testJWKSHandler(t, "test-key", publicKey)
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		// Hold the response open long enough for every concurrent caller to find the
		// refresh already in flight.
		time.Sleep(100 * time.Millisecond)
		handler(w, r)
	}))
	defer jwks.Close()

	cache := newMCPJWKSCache(jwks.URL, jwks.Client(), time.Minute, time.Nanosecond)

	const callers = 25
	start := make(chan struct{})
	var wg sync.WaitGroup
	keys := make([]*rsa.PublicKey, callers)
	errs := make([]error, callers)

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			keys[index], errs[index] = cache.key(context.Background(), "test-key")
		}(i)
	}

	close(start)
	wg.Wait()

	for i := 0; i < callers; i++ {
		require.NoError(t, errs[i])
		require.NotNil(t, keys[i])
	}
	assert.Equal(t, int64(1), requestCount.Load(), "concurrent JWKS cache misses must collapse into one fetch")
}

// TestMCPJWKSCacheServesKnownKeyWhileRefreshIsThrottled proves that rate limiting never
// invalidates a key the cache already holds: an expired cache TTL combined with a throttled
// refresh still serves the previously published key.
func TestMCPJWKSCacheServesKnownKeyWhileRefreshIsThrottled(t *testing.T) {
	_, publicKey := testRSAKey(t)

	var requestCount atomic.Int64
	jwks := countingJWKSServer(t, "test-key", publicKey, &requestCount)

	// A one-nanosecond cache TTL makes every lookup consider the cache stale, so only the
	// refresh interval bounds the outbound fetches.
	cache := newMCPJWKSCache(jwks.URL, jwks.Client(), time.Nanosecond, time.Minute)

	for i := 0; i < 10; i++ {
		key, err := cache.key(context.Background(), "test-key")
		require.NoError(t, err)
		require.NotNil(t, key)
	}

	assert.Equal(t, int64(1), requestCount.Load())
}

// TestMCPTokenVerifierVerifyRequest_RefreshesRotatedKeyAfterMinRefreshInterval proves a real
// MCP signing-key rotation is still picked up: the refresh is throttled immediately after the
// previous attempt, and succeeds once the interval has elapsed.
func TestMCPTokenVerifierVerifyRequest_RefreshesRotatedKeyAfterMinRefreshInterval(t *testing.T) {
	firstPrivateKey, firstPublicKey := testRSAKey(t)
	secondPrivateKey, secondPublicKey := testRSAKey(t)

	var requestCount atomic.Int64
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount.Add(1) == 1 {
			testJWKSHandler(t, "key-1", firstPublicKey)(w, r)
			return
		}
		testJWKSHandler(t, "key-2", secondPublicKey)(w, r)
	}))
	defer jwks.Close()

	minRefreshInterval := 150 * time.Millisecond
	verifier := newBoundedTestVerifier(t, jwks.URL, jwks.Client(), minRefreshInterval)

	firstToken := signDelegatedToken(t, firstPrivateKey, "key-1", testMCPClaims(
		"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
	))
	_, err := verifier.VerifyRequest(context.Background(), firstToken, http.MethodGet, "/v1/messages")
	require.NoError(t, err)
	require.Equal(t, int64(1), requestCount.Load())

	secondToken := signDelegatedToken(t, secondPrivateKey, "key-2", testMCPClaims(
		"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
	))

	// Immediately after the first fetch the rotated key cannot be picked up yet: the
	// refresh is throttled rather than amplified into another fetch.
	_, err = verifier.VerifyRequest(context.Background(), secondToken, http.MethodGet, "/v1/messages")
	require.Error(t, err)
	require.Equal(t, int64(1), requestCount.Load())

	time.Sleep(minRefreshInterval + 50*time.Millisecond)

	claims, err := verifier.VerifyRequest(context.Background(), secondToken, http.MethodGet, "/v1/messages")
	require.NoError(t, err)
	assert.Equal(t, "firebase-user-id", claims.Subject)
	assert.Equal(t, int64(2), requestCount.Load())
}

// TestHasUnverifiedIssuer covers the prefilter in isolation, including that a matching
// unverified issuer is only ever a "maybe ours" signal.
func TestHasUnverifiedIssuer(t *testing.T) {
	privateKey, _ := testRSAKey(t)

	mcpToken := signDelegatedToken(t, privateKey, "test-key", testMCPClaims(
		"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
	))

	firebaseClaims := testMCPClaims("firebase-user-id", nil, http.MethodGet, "/v1/messages")
	firebaseClaims.Issuer = "https://securetoken.google.com/httpsms-test"
	firebaseToken := signDelegatedToken(t, privateKey, "test-key", firebaseClaims)

	unsignedToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, testMCPClaims(
		"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
	)).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	tests := []struct {
		name     string
		raw      string
		expected bool
	}{
		{name: "mcp token", raw: mcpToken, expected: true},
		{name: "firebase token", raw: firebaseToken, expected: false},
		{name: "opaque value", raw: "not-a-json-web-token", expected: false},
		{name: "empty value", raw: "", expected: false},
		{name: "unsigned token with the mcp issuer", raw: unsignedToken, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasUnverifiedIssuer(tt.raw, "https://mcp.httpsms.com"))
		})
	}
}
