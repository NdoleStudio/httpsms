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
)

// ErrInvalidIdentityToken is returned by IdentityVerifier.Verify for any
// identity token that does not parse, does not verify against a known
// signing certificate, or fails an issuer/audience/expiry/subject check. It
// deliberately does not distinguish the failure reason, so a caller can
// never learn from the error alone which specific check failed.
var ErrInvalidIdentityToken = errors.New("auth: invalid identity token")

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
func NewFirebaseVerifier(projectID string, certsURL string, httpClient *http.Client, cacheTTL time.Duration) (*FirebaseVerifier, error) {
	if projectID == "" {
		return nil, errors.New("auth: Firebase project ID must not be empty")
	}
	if certsURL == "" {
		return nil, errors.New("auth: Firebase certificate URL must not be empty")
	}

	return &FirebaseVerifier{
		projectID: projectID,
		certs:     newFirebaseCertCache(certsURL, httpClient, cacheTTL),
	}, nil
}

// Verify implements IdentityVerifier. It requires raw to be signed RS256,
// issued by "https://securetoken.google.com/<projectID>", audienced to
// projectID, unexpired (with an expiry claim required to be present at
// all), and carrying a non-empty subject.
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
	if err != nil || !token.Valid || claims.Subject == "" {
		return Principal{}, ErrInvalidIdentityToken
	}

	return Principal{UserID: claims.Subject, Email: claims.Email}, nil
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
// from any generic JWKS/JWK cache. It refreshes at most once per call when
// a requested "kid" is not (or no longer) cached, and otherwise refreshes
// only after cacheTTL has elapsed since the last successful fetch -- the
// same bounded refresh-on-missing-kid behavior used by the httpSMS API's
// delegated MCP token verifier (api/pkg/auth's JWKS cache), applied here to
// Google's certificate-map response shape instead of a JWKS document.
type firebaseCertCache struct {
	url        string
	httpClient *http.Client
	cacheTTL   time.Duration

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// newFirebaseCertCache builds a firebaseCertCache for url.
func newFirebaseCertCache(url string, httpClient *http.Client, cacheTTL time.Duration) *firebaseCertCache {
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

	return &firebaseCertCache{
		url:        url,
		httpClient: client,
		cacheTTL:   cacheTTL,
		keys:       map[string]*rsa.PublicKey{},
	}
}

// key returns the cached RSA public key for kid, refreshing the
// certificate map at most once per call when the cache is stale or the key
// is not yet known.
func (cache *firebaseCertCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	cache.mu.Lock()
	key, ok := cache.keys[kid]
	expired := time.Since(cache.fetchedAt) >= cache.cacheTTL
	cache.mu.Unlock()

	if ok && !expired {
		return key, nil
	}

	if err := cache.refresh(ctx); err != nil {
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
