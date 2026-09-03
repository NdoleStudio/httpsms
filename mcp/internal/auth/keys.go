package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// minRSAKeyBits is the minimum accepted RSA signing key size. Keys smaller
// than this are rejected by NewKeySet regardless of encoding.
const minRSAKeyBits = 2048

// JWK is a single RSA public key published in JWKS format. It never carries
// private key material.
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKS is a JSON Web Key Set document as published at the MCP service's
// JWKS endpoint for downstream verifiers (including the httpSMS API).
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// keySetConfig holds the deployment-derived issuer/audiences a KeySet signs
// with, published atomically as a single immutable value so a concurrent
// reader either sees no configuration or sees all three fields fully
// populated — never a partially-applied Configure call.
type keySetConfig struct {
	issuer      string
	mcpAudience string
	apiAudience string
}

// KeySet loads a single RSA signing key and mints/publishes the JWTs issued
// by the hosted MCP service. A KeySet never logs, and never exposes through
// any method, the private key it holds.
//
// issuer, mcpAudience, and apiAudience are deployment configuration (derived
// from config.Config) rather than key material, so NewKeySet returns a
// KeySet that cannot sign anything until the caller calls Configure exactly
// once. Configure is deliberately one-shot (not a plain setter) so a KeySet
// can safely be shared across goroutines without a data race: the
// issuer/audiences are stored behind a single atomic.Pointer swap, so
// Configure either fully publishes a complete, immutable *keySetConfig or
// does nothing, and every signing method only ever reads the published
// value through an atomic load — there is no window in which a concurrent
// reader can observe a partially-configured KeySet.
type KeySet struct {
	privateKey *rsa.PrivateKey
	keyID      string

	// config is nil until Configure succeeds, after which it is never
	// written again. atomic.Pointer.CompareAndSwap makes "claim the
	// one-shot slot" and "publish the fully-built value" a single atomic
	// step, so concurrent Configure calls race safely (exactly one wins)
	// and concurrent signing calls never observe a half-written config.
	config atomic.Pointer[keySetConfig]
}

// NewKeySet parses privateKeyPEM (PKCS#1 or PKCS#8, RSA only, at least
// minRSAKeyBits bits) and returns a KeySet that signs with it under keyID.
func NewKeySet(privateKeyPEM []byte, keyID string) (*KeySet, error) {
	if keyID == "" {
		return nil, errors.New("auth: signing key ID must not be empty")
	}

	privateKey, err := parseRSAPrivateKeyPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("auth: cannot load RSA signing key: %w", err)
	}

	if bits := privateKey.N.BitLen(); bits < minRSAKeyBits {
		return nil, fmt.Errorf("auth: RSA signing key has %d bits, must be at least %d", bits, minRSAKeyBits)
	}

	return &KeySet{privateKey: privateKey, keyID: keyID}, nil
}

// parseRSAPrivateKeyPEM decodes a single PEM block and parses it as either a
// PKCS#1 or PKCS#8 RSA private key. Any other key type is rejected.
func parseRSAPrivateKeyPEM(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("not a PKCS#1 or PKCS#8 private key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("PKCS#8 key is %T, not an RSA private key", key)
	}

	return rsaKey, nil
}

// Configure sets issuer, mcpAudience, and apiAudience exactly once. It must
// be called before any signing method and must not be called more than
// once; both are programmer errors and return an error rather than
// panicking, so callers (and their tests) can assert on them.
//
// Configure builds the complete configuration value first, then publishes
// it with a single atomic.Pointer.CompareAndSwap. This makes "claim the
// one-shot slot" and "make the new issuer/audiences visible" the same
// indivisible step, so KeySet is safe to share across goroutines: a
// concurrent signing call either reads nil (and fails closed) or reads a
// fully-populated *keySetConfig, never a partially-applied one.
func (keys *KeySet) Configure(issuer, mcpAudience, apiAudience string) error {
	if issuer == "" {
		return errors.New("auth: KeySet issuer must not be empty")
	}
	if mcpAudience == "" {
		return errors.New("auth: KeySet MCP audience must not be empty")
	}
	if apiAudience == "" {
		return errors.New("auth: KeySet API audience must not be empty")
	}

	cfg := &keySetConfig{issuer: issuer, mcpAudience: mcpAudience, apiAudience: apiAudience}
	if !keys.config.CompareAndSwap(nil, cfg) {
		return errors.New("auth: KeySet is already configured")
	}

	return nil
}

// PublicKey returns the RSA public key corresponding to the loaded signing
// key, for verifying tokens minted by this KeySet in tests and internal
// callers. It never exposes the private key.
func (keys *KeySet) PublicKey() *rsa.PublicKey {
	return &keys.privateKey.PublicKey
}

// KeyID returns the `kid` this KeySet signs with and publishes in its JWKS.
func (keys *KeySet) KeyID() string {
	return keys.keyID
}

// SignMCPAccessToken mints a short-lived MCP access token for principal,
// scoped to scopes and bound to the OAuth client identified by clientID. The
// token is audience-bound to the configured MCP audience and must never be
// accepted by the httpSMS API.
func (keys *KeySet) SignMCPAccessToken(principal Principal, clientID string, scopes []string, ttl time.Duration) (string, error) {
	cfg, err := keys.requireConfig()
	if err != nil {
		return "", err
	}

	claims, err := keys.baseClaims(cfg.issuer, principal, cfg.mcpAudience, scopes, ttl)
	if err != nil {
		return "", err
	}
	claims.ClientID = clientID

	return keys.sign(claims)
}

// SignAPIDelegationToken mints a short-lived downstream API delegation token
// for principal, scoped to scopes, and bound to exactly one API operation
// (method, path). The token is audience-bound to the configured API
// audience.
//
// The resulting JWT carries JSON fields `scopes`, `http_method`, and
// `http_path`; the configured issuer; the configured API audience; subject
// principal.UserID; is signed RS256; and carries a `kid` header. This is a
// wire contract with the httpSMS API's delegated MCP token verifier
// (api/pkg/auth.MCPClaims) and must not change independently of it.
func (keys *KeySet) SignAPIDelegationToken(principal Principal, scopes []string, method string, path string, ttl time.Duration) (string, error) {
	if method == "" || path == "" {
		return "", errors.New("auth: API delegation token requires a non-empty method and path")
	}

	cfg, err := keys.requireConfig()
	if err != nil {
		return "", err
	}

	claims, err := keys.baseClaims(cfg.issuer, principal, cfg.apiAudience, scopes, ttl)
	if err != nil {
		return "", err
	}
	claims.Method = method
	claims.Path = path

	return keys.sign(claims)
}

// requireConfig returns the KeySet's published configuration, or an error if
// Configure has not yet been called successfully.
func (keys *KeySet) requireConfig() (*keySetConfig, error) {
	cfg := keys.config.Load()
	if cfg == nil {
		return nil, errors.New("auth: KeySet.Configure must be called before signing tokens")
	}
	return cfg, nil
}

// baseClaims builds the claims shared by every token this KeySet mints.
func (keys *KeySet) baseClaims(issuer string, principal Principal, audience string, scopes []string, ttl time.Duration) (*AccessClaims, error) {
	if principal.UserID == "" {
		return nil, errors.New("auth: token subject (Firebase UID) must not be empty")
	}
	if ttl <= 0 {
		return nil, errors.New("auth: token TTL must be positive")
	}

	jti, err := newTokenID()
	if err != nil {
		return nil, fmt.Errorf("auth: cannot generate token ID: %w", err)
	}

	now := time.Now().UTC()
	return &AccessClaims{
		Email:  principal.Email,
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   principal.UserID,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        jti,
		},
	}, nil
}

// sign signs claims with keys.privateKey using RS256 and publishes keys.keyID
// as the token's `kid` header.
func (keys *KeySet) sign(claims *AccessClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keys.keyID

	raw, err := token.SignedString(keys.privateKey)
	if err != nil {
		return "", fmt.Errorf("auth: cannot sign token: %w", err)
	}

	return raw, nil
}

// newTokenID returns a random 128-bit token identifier encoded as hex, used
// as the JWT `jti` claim.
func newTokenID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// JWKS returns the JSON Web Key Set publishing keys.PublicKey() under
// keys.keyID. It never publishes private key material.
func (keys *KeySet) JWKS() JWKS {
	publicKey := keys.PublicKey()

	return JWKS{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Alg: "RS256",
				Kid: keys.keyID,
				N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(bigEndianBytes(publicKey.E)),
			},
		},
	}
}

// bigEndianBytes returns the minimal big-endian encoding of a positive int,
// as required for a JWK's "e" member.
func bigEndianBytes(n int) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(int64(n)))

	i := 0
	for i < len(buf)-1 && buf[i] == 0 {
		i++
	}
	return buf[i:]
}
