package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
)

const (
	testFirebaseProjectID = "httpsms-test"
	testFirebaseIssuer    = "https://securetoken.google.com/httpsms-test"
)

// firebaseTestClaims mirrors the unexported claims shape FirebaseVerifier
// decodes, so tests can build tokens with exactly the fields a real
// Firebase ID token carries without depending on any unexported type.
type firebaseTestClaims struct {
	Email    string           `json:"email,omitempty"`
	UserID   string           `json:"user_id,omitempty"`
	AuthTime *jwt.NumericDate `json:"auth_time,omitempty"`
	jwt.RegisteredClaims
}

// validFirebaseClaims returns a claim set that a genuine, current Firebase
// ID token for testFirebaseProjectID would carry.
func validFirebaseClaims() firebaseTestClaims {
	now := time.Now()
	return firebaseTestClaims{
		Email:    "user@example.com",
		UserID:   "user-id",
		AuthTime: jwt.NewNumericDate(now.Add(-time.Minute)),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testFirebaseIssuer,
			Subject:   "user-id",
			Audience:  jwt.ClaimStrings{testFirebaseProjectID},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
}

// testRSAKeyPair generates a throwaway 2048-bit RSA key, for use only in
// tests.
func testRSAKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return key
}

// selfSignedCertificatePEM returns a PEM-encoded self-signed X.509
// certificate for key, in the same shape Google's Firebase certificate
// endpoint serves ("kid" -> PEM certificate).
func selfSignedCertificatePEM(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "firebase-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// firebaseCertsHandler serves a Firebase-style certificate map response
// mapping "kid" to a PEM certificate, exactly as Google's endpoint does.
func firebaseCertsHandler(certs map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(certs)
	}
}

// signFirebaseToken signs claims as a Firebase-style RS256 ID token under
// kid.
func signFirebaseToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	raw, err := token.SignedString(key)
	require.NoError(t, err)

	return raw
}

// newTestVerifier builds a FirebaseVerifier pointed at a test certificate
// endpoint, with the production default (one minute) minimum refresh
// interval.
func newTestVerifier(t *testing.T, certsURL string, client *http.Client) *auth.FirebaseVerifier {
	t.Helper()

	verifier, err := auth.NewFirebaseVerifier(testFirebaseProjectID, certsURL, client, 0, 0)
	require.NoError(t, err)

	return verifier
}

func TestNewFirebaseVerifierRequiresProjectID(t *testing.T) {
	_, err := auth.NewFirebaseVerifier("", "https://example.com/certs", nil, 0, 0)
	require.Error(t, err)
}

func TestNewFirebaseVerifierRequiresCertsURL(t *testing.T) {
	_, err := auth.NewFirebaseVerifier("httpsms-test", "", nil, 0, 0)
	require.Error(t, err)
}

func TestFirebaseVerifierAcceptsValidToken(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)

	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())
	raw := signFirebaseToken(t, key, "firebase-test-key", validFirebaseClaims())

	principal, err := verifier.Verify(context.Background(), raw)
	require.NoError(t, err)
	assert.Equal(t, "user-id", principal.UserID)
	assert.Equal(t, "user@example.com", principal.Email)
}

func TestFirebaseVerifierRejectsWrongIssuer(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.Issuer = "https://securetoken.google.com/some-other-project"
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsWrongAudience(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.Audience = jwt.ClaimStrings{"some-other-project"}
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsExpiredToken(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsMissingExpiry(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.ExpiresAt = nil
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsUnknownKid(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"a-different-kid": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())
	raw := signFirebaseToken(t, key, "firebase-test-key", validFirebaseClaims())

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsWrongSigningMethod(t *testing.T) {
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, validFirebaseClaims())
	raw, err := token.SignedString([]byte("does-not-matter"))
	require.NoError(t, err)

	_, err = verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsMalformedToken(t *testing.T) {
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	_, err := verifier.Verify(context.Background(), "not-a-jwt")
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsTokenWhenCertsEndpointUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())
	raw := signFirebaseToken(t, testRSAKeyPair(t), "any-kid", validFirebaseClaims())

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

// TestFirebaseVerifierCachesCertificatesAndRefreshesOnRotation asserts the
// bounded cached-certificate-fetching behavior: a cache hit never refetches;
// a "kid" rotated in after the cache was populated triggers exactly one
// additional bounded fetch before the newly-signed token verifies, once the
// minimum refresh interval has elapsed.
func TestFirebaseVerifierCachesCertificatesAndRefreshesOnRotation(t *testing.T) {
	firstKey := testRSAKeyPair(t)
	secondKey := testRSAKeyPair(t)
	firstCert := selfSignedCertificatePEM(t, firstKey)
	secondCert := selfSignedCertificatePEM(t, secondKey)

	var requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount.Add(1) == 1 {
			firebaseCertsHandler(map[string]string{"key-1": firstCert})(w, r)
			return
		}
		firebaseCertsHandler(map[string]string{"key-2": secondCert})(w, r)
	}))
	defer server.Close()

	verifier, err := auth.NewFirebaseVerifier(testFirebaseProjectID, server.URL, server.Client(), 0, time.Millisecond)
	require.NoError(t, err)

	firstToken := signFirebaseToken(t, firstKey, "key-1", validFirebaseClaims())
	_, err = verifier.Verify(context.Background(), firstToken)
	require.NoError(t, err)
	assert.Equal(t, int64(1), requestCount.Load())

	// A cache hit for the already-known "key-1" must not trigger another
	// fetch.
	_, err = verifier.Verify(context.Background(), firstToken)
	require.NoError(t, err)
	assert.Equal(t, int64(1), requestCount.Load())

	// The cache only has "key-1"; a token signed with the newly rotated
	// "key-2" forces exactly one bounded refresh before it can verify --
	// legitimate rotation still works, it is only rate limited.
	time.Sleep(5 * time.Millisecond)
	secondToken := signFirebaseToken(t, secondKey, "key-2", validFirebaseClaims())
	principal, err := verifier.Verify(context.Background(), secondToken)
	require.NoError(t, err)
	assert.Equal(t, "user-id", principal.UserID)
	assert.Equal(t, int64(2), requestCount.Load())
}

// TestFirebaseVerifierRefreshesAfterCacheTTLExpires asserts the cache also
// refreshes on a plain TTL expiry, not only on a missing "kid".
func TestFirebaseVerifierRefreshesAfterCacheTTLExpires(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)

	var requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM})(w, r)
	}))
	defer server.Close()

	verifier, err := auth.NewFirebaseVerifier(testFirebaseProjectID, server.URL, server.Client(), 10*time.Millisecond, time.Millisecond)
	require.NoError(t, err)

	raw := signFirebaseToken(t, key, "firebase-test-key", validFirebaseClaims())

	_, err = verifier.Verify(context.Background(), raw)
	require.NoError(t, err)
	assert.Equal(t, int64(1), requestCount.Load())

	time.Sleep(20 * time.Millisecond)

	_, err = verifier.Verify(context.Background(), raw)
	require.NoError(t, err)
	assert.Equal(t, int64(2), requestCount.Load())
}

// TestFirebaseVerifierRateLimitsUnknownKidRefreshes asserts an attacker
// cannot amplify a flood of tokens carrying random unknown "kid" headers
// into one outbound certificate fetch per request: after the first fetch,
// no further fetch happens until the minimum refresh interval elapses.
//
// No token, certificate, or key material is logged by this test; only the
// outbound request count is asserted.
func TestFirebaseVerifierRateLimitsUnknownKidRefreshes(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)

	var requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM})(w, r)
	}))
	defer server.Close()

	// A one-minute minimum refresh interval, i.e. the production default.
	verifier := newTestVerifier(t, server.URL, server.Client())

	valid := signFirebaseToken(t, key, "firebase-test-key", validFirebaseClaims())
	_, err := verifier.Verify(context.Background(), valid)
	require.NoError(t, err)
	require.Equal(t, int64(1), requestCount.Load())

	for i := 0; i < 200; i++ {
		unknown := signFirebaseToken(t, key, fmt.Sprintf("random-kid-%d", i), validFirebaseClaims())
		_, err := verifier.Verify(context.Background(), unknown)
		require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
	}

	assert.Equal(t, int64(1), requestCount.Load(), "unknown kids must not cause one outbound fetch per request")

	// The still-cached, still-published key keeps verifying throughout.
	_, err = verifier.Verify(context.Background(), valid)
	require.NoError(t, err)
	assert.Equal(t, int64(1), requestCount.Load())
}

// TestFirebaseVerifierCollapsesConcurrentRefreshes asserts that a burst of
// concurrent verifications arriving against a cold cache shares a single
// outbound fetch instead of issuing one per goroutine.
func TestFirebaseVerifierCollapsesConcurrentRefreshes(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)

	var requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		// Hold the fetch open long enough for every caller to pile up
		// behind the single in-flight refresh.
		time.Sleep(50 * time.Millisecond)
		firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM})(w, r)
	}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())
	raw := signFirebaseToken(t, key, "firebase-test-key", validFirebaseClaims())

	const callers = 25
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := verifier.Verify(context.Background(), raw)
			errs[index] = err
		}(i)
	}
	wg.Wait()

	for index, err := range errs {
		require.NoError(t, err, "caller %d must verify against the shared refresh", index)
	}
	assert.Equal(t, int64(1), requestCount.Load(), "concurrent refreshes must be collapsed into one fetch")
}

func TestFirebaseVerifierRejectsMissingIssuedAt(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.IssuedAt = nil
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsFutureIssuedAt(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.IssuedAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsMissingAuthTime(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.AuthTime = nil
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsFutureAuthTime(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.AuthTime = jwt.NewNumericDate(time.Now().Add(time.Hour))
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}

func TestFirebaseVerifierRejectsEmptySubject(t *testing.T) {
	key := testRSAKeyPair(t)
	certPEM := selfSignedCertificatePEM(t, key)
	server := httptest.NewServer(firebaseCertsHandler(map[string]string{"firebase-test-key": certPEM}))
	defer server.Close()

	verifier := newTestVerifier(t, server.URL, server.Client())

	claims := validFirebaseClaims()
	claims.Subject = ""
	raw := signFirebaseToken(t, key, "firebase-test-key", claims)

	_, err := verifier.Verify(context.Background(), raw)
	require.ErrorIs(t, err, auth.ErrInvalidIdentityToken)
}
