package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Bounds applied to every fetch of Google's Firebase certificate endpoint.
const (
	firebaseCertsHTTPTimeout      = 2 * time.Second
	firebaseCertsMaxResponseBytes = 1 << 20 // 1 MiB
	firebaseCertsDefaultCacheTTL  = time.Hour

	// firebaseCertsDefaultMinRefreshInterval is the default minimum delay
	// between two outbound fetches of the certificate endpoint. It bounds
	// refresh amplification: without it, a flood of tokens carrying random
	// unknown "kid" headers would cause one outbound fetch per request.
	// Google publishes a rotated signing key well before it starts signing
	// with it, so a legitimate rotation is still picked up -- at worst one
	// interval late.
	firebaseCertsDefaultMinRefreshInterval = time.Minute

	// firebaseClockSkewLeeway is the tolerance applied to the "iat" and
	// "auth_time" claims, which are stamped by Google's clock and compared
	// against ours.
	firebaseClockSkewLeeway = time.Minute
)

// ErrInvalidIdentityToken is returned by IdentityVerifier.Verify for any
// identity token that does not parse, does not verify against a known
// signing certificate, or fails an issuer/audience/expiry/subject check. It
// deliberately does not distinguish the failure reason, so a caller can
// never learn from the error alone which specific check failed.
var ErrInvalidIdentityToken = errors.New("auth: invalid identity token")

// errFirebaseCertsRefreshThrottled reports that a certificate refresh was
// skipped because the minimum refresh interval has not elapsed yet.
var errFirebaseCertsRefreshThrottled = errors.New("auth: Firebase certificate refresh is rate limited")

// IdentityVerifier verifies a raw bearer identity token -- a Firebase ID
// token presented during the browser login step of the OAuth authorization
// flow -- and returns the Principal it identifies.
type IdentityVerifier interface {
	Verify(ctx context.Context, raw string) (Principal, error)
}

// firebaseClaims are the claims read from a Firebase ID token, beyond the
// registered claims already validated by the jwt.ParseWithClaims options in
// FirebaseVerifier.Verify.
type firebaseClaims struct {
	Email  string `json:"email,omitempty"`
	UserID string `json:"user_id,omitempty"`

	// AuthTime is the Firebase "auth_time" claim: when the user actually
	// authenticated. Firebase's own ID-token verification contract requires
	// it to be present and in the past.
	AuthTime *jwt.NumericDate `json:"auth_time,omitempty"`

	jwt.RegisteredClaims
}

// FirebaseVerifier verifies Firebase ID tokens for a single Firebase
// project directly against Google's public certificate endpoint. It never
// depends on the Firebase Admin SDK, and it never logs or returns the raw
// token it verifies.
type FirebaseVerifier struct {
	projectID string
	certs     *firebaseCertCache
}

// NewFirebaseVerifier returns a FirebaseVerifier for projectID, fetching
// signing certificates from certsURL (Google's
// "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com"
// endpoint in production) through httpClient.
//
// httpClient may be nil, in which case http.DefaultClient is used (its
// Transport, if any, is preserved so tests can point it at an httptest
// server; a bounded per-request timeout is always enforced regardless).
// cacheTTL may be <= 0, in which case a one-hour default is used.
// minRefreshInterval bounds how often an unknown "kid" (or an expired
// cache) may trigger an outbound fetch; it may be <= 0, in which case a
// one-minute default is used.
func NewFirebaseVerifier(projectID string, certsURL string, httpClient *http.Client, cacheTTL time.Duration, minRefreshInterval time.Duration) (*FirebaseVerifier, error) {
	if projectID == "" {
		return nil, errors.New("auth: Firebase project ID must not be empty")
	}
	if certsURL == "" {
		return nil, errors.New("auth: Firebase certificate URL must not be empty")
	}

	return &FirebaseVerifier{
		projectID: projectID,
		certs:     newFirebaseCertCache(certsURL, httpClient, cacheTTL, minRefreshInterval),
	}, nil
}

// Verify implements IdentityVerifier. It requires raw to be signed RS256,
// issued by "https://securetoken.google.com/<projectID>", audienced to
// projectID, unexpired (with an expiry claim required to be present at
// all), carrying "iat" and "auth_time" claims that are not in the future,
// and carrying a non-empty subject.
func (v *FirebaseVerifier) Verify(ctx context.Context, raw string) (Principal, error) {
	claims := new(firebaseClaims)
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		v.keyfunc(ctx),
		jwt.WithIssuer("https://securetoken.google.com/"+v.projectID),
		jwt.WithAudience(v.projectID),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	if err != nil || !token.Valid || claims.Subject == "" || !hasValidFirebaseIssueTimes(claims) {
		return Principal{}, ErrInvalidIdentityToken
	}

	return Principal{UserID: claims.Subject, Email: claims.Email}, nil
}

// hasValidFirebaseIssueTimes reports whether the token's "iat" and
// "auth_time" claims are both present and not in the future (allowing for
// firebaseClockSkewLeeway). Firebase's documented ID-token verification
// contract requires both, and neither is validated by the registered-claim
// options passed to jwt.ParseWithClaims.
func hasValidFirebaseIssueTimes(claims *firebaseClaims) bool {
	if claims.IssuedAt == nil || claims.AuthTime == nil {
		return false
	}

	latest := time.Now().Add(firebaseClockSkewLeeway)

	return !claims.IssuedAt.After(latest) && !claims.AuthTime.After(latest)
}

// keyfunc returns a jwt.Keyfunc that resolves the RSA public key matching
// the token's "kid" header from the cached Firebase certificate map.
func (v *FirebaseVerifier) keyfunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidIdentityToken
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, ErrInvalidIdentityToken
		}

		return v.certs.key(ctx, kid)
	}
}

// firebaseCertCache fetches and caches the RSA public keys published by
// Google's Firebase certificate endpoint, keyed by "kid". Google's endpoint
// serves a flat JSON object mapping key ID to a PEM-encoded X.509
// certificate (not a JWKS document), so this cache is deliberately separate
// from any generic JWKS/JWK cache.
//
// Two bounds keep an attacker from turning a stream of tokens carrying
// random unknown "kid" headers into a stream of outbound fetches:
//
//   - concurrent refreshes are collapsed into a single in-flight fetch that
//     every waiting caller shares, and
//   - a new fetch is never started until minRefreshInterval has elapsed
//     since the previous attempt (successful or not); until then, callers
//     either reuse the cached key or fail closed.
//
// A legitimate key rotation is still picked up: Google publishes a rotated
// certificate before signing with it, and a missing "kid" triggers a real
// refresh as soon as the interval has elapsed.
type firebaseCertCache struct {
	url                string
	httpClient         *http.Client
	cacheTTL           time.Duration
	minRefreshInterval time.Duration

	mu            sync.Mutex
	keys          map[string]*rsa.PublicKey
	fetchedAt     time.Time
	lastAttemptAt time.Time
	inflight      *firebaseCertRefresh
}

// firebaseCertRefresh is a single in-flight certificate refresh shared by
// every caller that arrives while it is running. err is written before done
// is closed, so a waiter that observes done may safely read it.
type firebaseCertRefresh struct {
	done chan struct{}
	err  error
}

// newFirebaseCertCache builds a firebaseCertCache for url.
func newFirebaseCertCache(url string, httpClient *http.Client, cacheTTL time.Duration, minRefreshInterval time.Duration) *firebaseCertCache {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// Reuse the caller's transport (important for tests using httptest
	// servers) but always enforce our own bounded timeout.
	client := &http.Client{
		Transport: httpClient.Transport,
		Timeout:   firebaseCertsHTTPTimeout,
	}

	if cacheTTL <= 0 {
		cacheTTL = firebaseCertsDefaultCacheTTL
	}
	if minRefreshInterval <= 0 {
		minRefreshInterval = firebaseCertsDefaultMinRefreshInterval
	}

	return &firebaseCertCache{
		url:                url,
		httpClient:         client,
		cacheTTL:           cacheTTL,
		minRefreshInterval: minRefreshInterval,
		keys:               map[string]*rsa.PublicKey{},
	}
}

// key returns the cached RSA public key for kid, refreshing the
// certificate map when the cache is stale or the key is not yet known --
// subject to the collapsing and rate limiting described on
// firebaseCertCache.
func (cache *firebaseCertCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	cache.mu.Lock()
	key, ok := cache.keys[kid]
	expired := time.Since(cache.fetchedAt) >= cache.cacheTTL
	cache.mu.Unlock()

	if ok && !expired {
		return key, nil
	}

	if err := cache.refreshOnce(ctx); err != nil {
		// A rate-limited refresh must not invalidate a key we already
		// hold: serving the (stale but still published) cached key is
		// strictly better than failing a legitimate login because the
		// cache TTL elapsed moments after the last fetch attempt.
		if errors.Is(err, errFirebaseCertsRefreshThrottled) && ok {
			return key, nil
		}
		return nil, fmt.Errorf("auth: cannot refresh Firebase certificates: %w", err)
	}

	cache.mu.Lock()
	key, ok = cache.keys[kid]
	cache.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("%w: no certificate for kid %q", ErrInvalidIdentityToken, kid)
	}

	return key, nil
}

// refreshOnce performs at most one outbound certificate fetch on behalf of
// every caller that needs one at the same time, and refuses to start a new
// fetch until minRefreshInterval has elapsed since the previous attempt.
func (cache *firebaseCertCache) refreshOnce(ctx context.Context) error {
	cache.mu.Lock()

	if inflight := cache.inflight; inflight != nil {
		cache.mu.Unlock()
		select {
		case <-inflight.done:
			return inflight.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if !cache.lastAttemptAt.IsZero() && time.Since(cache.lastAttemptAt) < cache.minRefreshInterval {
		cache.mu.Unlock()
		return errFirebaseCertsRefreshThrottled
	}

	inflight := &firebaseCertRefresh{done: make(chan struct{})}
	cache.inflight = inflight
	cache.lastAttemptAt = time.Now()
	cache.mu.Unlock()

	err := cache.refresh(ctx)
	inflight.err = err

	cache.mu.Lock()
	cache.inflight = nil
	cache.mu.Unlock()
	close(inflight.done)

	return err
}

// refresh fetches and replaces the cached certificate map.
func (cache *firebaseCertCache) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cache.url, nil)
	if err != nil {
		return fmt.Errorf("auth: cannot create request for Firebase certificate URL %q: %w", cache.url, err)
	}

	resp, err := cache.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth: cannot fetch Firebase certificates from %q: %w", cache.url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: Firebase certificate endpoint %q returned status %d", cache.url, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, firebaseCertsMaxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("auth: cannot read Firebase certificate response from %q: %w", cache.url, err)
	}
	if len(body) > firebaseCertsMaxResponseBytes {
		return fmt.Errorf("auth: Firebase certificate response from %q exceeds the %d byte limit", cache.url, firebaseCertsMaxResponseBytes)
	}

	var certs map[string]string
	if err := json.Unmarshal(body, &certs); err != nil {
		return fmt.Errorf("auth: cannot decode Firebase certificate response from %q: %w", cache.url, err)
	}

	keys := make(map[string]*rsa.PublicKey, len(certs))
	for kid, certPEM := range certs {
		publicKey, err := rsaPublicKeyFromCertificatePEM(certPEM)
		if err != nil {
			// Skip a single malformed entry rather than failing the whole
			// refresh; an unusable "kid" simply remains unresolvable.
			continue
		}
		keys[kid] = publicKey
	}

	cache.mu.Lock()
	cache.keys = keys
	cache.fetchedAt = time.Now()
	cache.mu.Unlock()

	return nil
}

// rsaPublicKeyFromCertificatePEM decodes a single PEM-encoded X.509
// certificate and returns its RSA public key.
func rsaPublicKeyFromCertificatePEM(certPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("auth: not a PEM-encoded certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("auth: cannot parse X.509 certificate: %w", err)
	}

	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("auth: certificate public key is %T, not RSA", cert.PublicKey)
	}

	return publicKey, nil
}
