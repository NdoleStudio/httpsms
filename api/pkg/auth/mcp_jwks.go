package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/NdoleStudio/stacktrace"
)

const (
	// mcpJWKSDefaultCacheTTL is used when MCPTokenVerifierConfig.CacheTTL is not set.
	mcpJWKSDefaultCacheTTL = 15 * time.Minute

	// mcpJWKSHTTPTimeout bounds every HTTP call made to fetch the JWKS document.
	mcpJWKSHTTPTimeout = 2 * time.Second

	// mcpJWKSMaxResponseBytes bounds the size of the JWKS document read from the network.
	mcpJWKSMaxResponseBytes = 1 << 20 // 1 MiB
)

// mcpJWK is a single JSON Web Key as published by a JWKS endpoint. Only the
// fields required to build an RSA public key are decoded.
type mcpJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// mcpJWKSet is the JSON Web Key Set document shape.
type mcpJWKSet struct {
	Keys []mcpJWK `json:"keys"`
}

// mcpJWKSCache fetches and caches the RSA public keys published by a JWKS
// endpoint, keyed by "kid". It refreshes the cache once when a requested "kid"
// cannot be found, and otherwise refreshes only after CacheTTL has elapsed.
type mcpJWKSCache struct {
	url        string
	httpClient *http.Client
	cacheTTL   time.Duration

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// newMCPJWKSCache creates a new mcpJWKSCache for the given JWKS URL.
func newMCPJWKSCache(url string, httpClient *http.Client, cacheTTL time.Duration) *mcpJWKSCache {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// Reuse the caller's transport (important for tests using httptest
	// servers) but always enforce our own bounded timeout.
	client := &http.Client{
		Transport: httpClient.Transport,
		Timeout:   mcpJWKSHTTPTimeout,
	}

	if cacheTTL <= 0 {
		cacheTTL = mcpJWKSDefaultCacheTTL
	}

	return &mcpJWKSCache{
		url:        url,
		httpClient: client,
		cacheTTL:   cacheTTL,
		keys:       map[string]*rsa.PublicKey{},
	}
}

// key returns the cached RSA public key for kid, refreshing the JWKS document
// at most once per call when the cache is stale or the key is not yet known.
func (cache *mcpJWKSCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	cache.mu.Lock()
	key, ok := cache.keys[kid]
	expired := time.Since(cache.fetchedAt) >= cache.cacheTTL
	cache.mu.Unlock()

	if ok && !expired {
		return key, nil
	}

	if err := cache.refresh(ctx); err != nil {
		return nil, stacktrace.Propagatef(err, "cannot refresh MCP JWKS from [%s]", cache.url)
	}

	cache.mu.Lock()
	key, ok = cache.keys[kid]
	cache.mu.Unlock()
	if !ok {
		return nil, stacktrace.NewErrorWithCodef(ErrCodeInvalidToken, "MCP JWKS has no key with kid [%s]", kid)
	}

	return key, nil
}

// refresh fetches and replaces the cached JWKS key set.
func (cache *mcpJWKSCache) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cache.url, nil)
	if err != nil {
		return stacktrace.Propagatef(err, "cannot create request for MCP JWKS URL [%s]", cache.url)
	}

	resp, err := cache.httpClient.Do(req)
	if err != nil {
		return stacktrace.Propagatef(err, "cannot fetch MCP JWKS from [%s]", cache.url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return stacktrace.NewErrorf("MCP JWKS endpoint [%s] returned status code [%d]", cache.url, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, mcpJWKSMaxResponseBytes+1))
	if err != nil {
		return stacktrace.Propagatef(err, "cannot read response body from MCP JWKS URL [%s]", cache.url)
	}
	if len(body) > mcpJWKSMaxResponseBytes {
		return stacktrace.NewErrorf("MCP JWKS response from [%s] exceeds the [%d] byte limit", cache.url, mcpJWKSMaxResponseBytes)
	}

	var set mcpJWKSet
	if err = json.Unmarshal(body, &set); err != nil {
		return stacktrace.Propagatef(err, "cannot decode MCP JWKS response from [%s]", cache.url)
	}

	keys := map[string]*rsa.PublicKey{}
	for _, jwk := range set.Keys {
		if jwk.Kty != "RSA" || jwk.Kid == "" {
			continue
		}

		publicKey, err := rsaPublicKeyFromJWK(jwk)
		if err != nil {
			continue
		}

		keys[jwk.Kid] = publicKey
	}

	cache.mu.Lock()
	cache.keys = keys
	cache.fetchedAt = time.Now()
	cache.mu.Unlock()

	return nil
}

// rsaPublicKeyFromJWK constructs an *rsa.PublicKey from the modulus and
// exponent of a JSON Web Key.
func rsaPublicKeyFromJWK(jwk mcpJWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("cannot decode modulus for kid [%s]: %w", jwk.Kid, err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("cannot decode exponent for kid [%s]: %w", jwk.Kid, err)
	}

	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() {
		return nil, fmt.Errorf("exponent for kid [%s] is out of range", jwk.Kid)
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(e.Int64()),
	}, nil
}
