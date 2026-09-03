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

// KeySet loads a single RSA signing key and mints/publishes the JWTs issued
// by the hosted MCP service. A KeySet never logs, and never exposes through
// any method, the private key it holds.
//
// Issuer, MCPAudience, and APIAudience are deployment configuration (derived
// from config.Config) rather than key material, so they are plain exported
// fields the caller sets after construction rather than constructor
// parameters. Signing methods use whatever value is set at call time.
type KeySet struct {
	// Issuer is used as the `iss` claim for every token this KeySet mints.
	// The wire contract with the httpSMS API requires this to be the MCP
	// service's base URL (MCP_BASE_URL).
	Issuer string

	// MCPAudience is the `aud` claim for MCP access tokens, e.g.
	// "https://mcp.httpsms.com/mcp".
	MCPAudience string

	// APIAudience is the `aud` claim for API delegation tokens, e.g.
	// "https://api.httpsms.com". This must match the httpSMS API's
	// configured API_AUDIENCE.
	APIAudience string

	privateKey *rsa.PrivateKey
	keyID      string
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
// token is audience-bound to keys.MCPAudience and must never be accepted by
// the httpSMS API.
func (keys *KeySet) SignMCPAccessToken(principal Principal, clientID string, scopes []string, ttl time.Duration) (string, error) {
	claims, err := keys.baseClaims(principal, keys.MCPAudience, scopes, ttl)
	if err != nil {
		return "", err
	}
	claims.ClientID = clientID

	return keys.sign(claims)
}

// SignAPIDelegationToken mints a short-lived downstream API delegation token
// for principal, scoped to scopes, and bound to exactly one API operation
// (method, path). The token is audience-bound to keys.APIAudience.
//
// The resulting JWT carries JSON fields `scopes`, `http_method`, and
// `http_path`; issuer keys.Issuer; audience keys.APIAudience; subject
// principal.UserID; is signed RS256; and carries a `kid` header. This is a
// wire contract with the httpSMS API's delegated MCP token verifier
// (api/pkg/auth.MCPClaims) and must not change independently of it.
func (keys *KeySet) SignAPIDelegationToken(principal Principal, scopes []string, method string, path string, ttl time.Duration) (string, error) {
	if method == "" || path == "" {
		return "", errors.New("auth: API delegation token requires a non-empty method and path")
	}

	claims, err := keys.baseClaims(principal, keys.APIAudience, scopes, ttl)
	if err != nil {
		return "", err
	}
	claims.Method = method
	claims.Path = path

	return keys.sign(claims)
}

// baseClaims builds the claims shared by every token this KeySet mints.
func (keys *KeySet) baseClaims(principal Principal, audience string, scopes []string, ttl time.Duration) (*AccessClaims, error) {
	if principal.UserID == "" {
		return nil, errors.New("auth: token subject (Firebase UID) must not be empty")
	}
	if keys.Issuer == "" {
		return nil, errors.New("auth: KeySet.Issuer must be set before signing tokens")
	}
	if audience == "" {
		return nil, errors.New("auth: token audience must not be empty")
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
			Issuer:    keys.Issuer,
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
