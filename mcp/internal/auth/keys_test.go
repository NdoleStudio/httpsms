package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
)

const (
	testMCPIssuer      = "https://mcp.httpsms.com"
	testMCPAudience    = "https://mcp.httpsms.com/mcp"
	testAPIAudience    = "https://api.httpsms.com"
	testSigningKeyID   = "test-key-1"
	testFirebaseUserID = "user-id"
	testUserEmail      = "user@example.com"
)

// newTestPrivateKeyPEM generates a throwaway RSA private key of the given
// size encoded as PKCS#1 PEM, for use only in tests.
func newTestPrivateKeyPEM(t *testing.T, bits int) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// newTestPKCS8PrivateKeyPEM generates a throwaway 2048-bit RSA private key
// encoded as PKCS#8 PEM, for use only in tests.
func newTestPKCS8PrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	bytes, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: bytes})
}

// newTestKeySet builds a KeySet with test issuer/audiences already
// configured, as a production caller would after loading them from
// config.Config.
func newTestKeySet(t *testing.T) *auth.KeySet {
	t.Helper()

	keys, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 2048), testSigningKeyID)
	require.NoError(t, err)

	require.NoError(t, keys.Configure(testMCPIssuer, testMCPAudience, testAPIAudience))

	return keys
}

// parseTestClaims verifies raw against publicKey and returns its claims,
// failing the test if raw does not parse or verify.
func parseTestClaims(t *testing.T, raw string, publicKey *rsa.PublicKey) *auth.AccessClaims {
	t.Helper()

	claims := new(auth.AccessClaims)
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		return publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	require.NoError(t, err)
	require.True(t, token.Valid)

	return claims
}

func TestNewKeySetRejectsEmptyKeyID(t *testing.T) {
	_, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 2048), "")
	require.Error(t, err)
}

func TestNewKeySetRejectsInvalidPEM(t *testing.T) {
	_, err := auth.NewKeySet([]byte("not a pem block"), testSigningKeyID)
	require.Error(t, err)
}

func TestNewKeySetRejectsKeysSmallerThan2048Bits(t *testing.T) {
	_, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 1024), testSigningKeyID)
	require.ErrorContains(t, err, "2048")
}

func TestNewKeySetAcceptsPKCS1AndPKCS8Encodings(t *testing.T) {
	_, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 2048), testSigningKeyID)
	require.NoError(t, err)

	_, err = auth.NewKeySet(newTestPKCS8PrivateKeyPEM(t), testSigningKeyID)
	require.NoError(t, err)
}

func TestKeySetSigningFailsUntilConfigured(t *testing.T) {
	keys, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 2048), testSigningKeyID)
	require.NoError(t, err)

	_, err = keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUserID}, "client", []string{"phones:read"}, time.Minute)
	require.ErrorContains(t, err, "Configure")

	_, err = keys.SignAPIDelegationToken(auth.Principal{UserID: testFirebaseUserID}, []string{"phones:read"}, "GET", "/v1/phones", time.Minute)
	require.ErrorContains(t, err, "Configure")
}

func TestKeySetConfigureSucceedsOnce(t *testing.T) {
	keys, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 2048), testSigningKeyID)
	require.NoError(t, err)

	require.NoError(t, keys.Configure(testMCPIssuer, testMCPAudience, testAPIAudience))

	raw, err := keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUserID}, "client", []string{"phones:read"}, time.Minute)
	require.NoError(t, err)

	claims := parseTestClaims(t, raw, keys.PublicKey())
	assert.Equal(t, testMCPIssuer, claims.Issuer)
	assert.Equal(t, testMCPAudience, claims.Audience[0])
}

func TestKeySetConfigureRejectsEmptyValues(t *testing.T) {
	testCases := map[string]struct {
		issuer      string
		mcpAudience string
		apiAudience string
	}{
		"empty issuer":      {issuer: "", mcpAudience: testMCPAudience, apiAudience: testAPIAudience},
		"empty mcpAudience": {issuer: testMCPIssuer, mcpAudience: "", apiAudience: testAPIAudience},
		"empty apiAudience": {issuer: testMCPIssuer, mcpAudience: testMCPAudience, apiAudience: ""},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			keys, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 2048), testSigningKeyID)
			require.NoError(t, err)

			err = keys.Configure(tc.issuer, tc.mcpAudience, tc.apiAudience)
			require.Error(t, err)

			// A rejected Configure call must not leave the KeySet able to
			// sign, nor able to be configured again with valid values (an
			// empty-value call must not consume the one-shot slot).
			_, signErr := keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUserID}, "client", []string{"phones:read"}, time.Minute)
			require.Error(t, signErr)

			require.NoError(t, keys.Configure(testMCPIssuer, testMCPAudience, testAPIAudience))
		})
	}
}

func TestKeySetConfigureRejectsSecondCall(t *testing.T) {
	keys, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 2048), testSigningKeyID)
	require.NoError(t, err)

	require.NoError(t, keys.Configure(testMCPIssuer, testMCPAudience, testAPIAudience))

	err = keys.Configure("https://other.example", "https://other.example/mcp", "https://other.example/api")
	require.ErrorContains(t, err, "already configured")

	// The rejected reconfiguration must not have overwritten the original
	// issuer/audiences.
	raw, err := keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUserID}, "client", []string{"phones:read"}, time.Minute)
	require.NoError(t, err)

	claims := parseTestClaims(t, raw, keys.PublicKey())
	assert.Equal(t, testMCPIssuer, claims.Issuer)
	assert.Equal(t, testMCPAudience, claims.Audience[0])
}

func TestKeySetConfigureIsRaceFreeUnderConcurrentCalls(t *testing.T) {
	keys, err := auth.NewKeySet(newTestPrivateKeyPEM(t, 2048), testSigningKeyID)
	require.NoError(t, err)

	const attempts = 16
	results := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			results <- keys.Configure(testMCPIssuer, testMCPAudience, testAPIAudience)
		}()
	}

	successes := 0
	for i := 0; i < attempts; i++ {
		if err := <-results; err == nil {
			successes++
		}
	}
	assert.Equal(t, 1, successes, "exactly one concurrent Configure call must succeed")

	raw, err := keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUserID}, "client", []string{"phones:read"}, time.Minute)
	require.NoError(t, err)
	claims := parseTestClaims(t, raw, keys.PublicKey())
	assert.Equal(t, testMCPIssuer, claims.Issuer)
}

func TestKeySetSignsAudienceBoundTokens(t *testing.T) {
	keys := newTestKeySet(t)

	raw, err := keys.SignMCPAccessToken(
		auth.Principal{UserID: testFirebaseUserID, Email: testUserEmail},
		"https://client.example/metadata.json",
		[]string{"messages:read"},
		15*time.Minute,
	)
	require.NoError(t, err)

	claims := parseTestClaims(t, raw, keys.PublicKey())
	require.Len(t, claims.Audience, 1)
	assert.Equal(t, testMCPAudience, claims.Audience[0])
	assert.Equal(t, testFirebaseUserID, claims.Subject)
	assert.Equal(t, testMCPIssuer, claims.Issuer)
	assert.Equal(t, []string{"messages:read"}, claims.Scopes)
	assert.Equal(t, "https://client.example/metadata.json", claims.ClientID)
	assert.Empty(t, claims.Method)
	assert.Empty(t, claims.Path)
}

func TestKeySetSignsAPIDelegationTokensBoundToOneOperation(t *testing.T) {
	keys := newTestKeySet(t)
	ttl := 2 * time.Minute

	before := time.Now()
	raw, err := keys.SignAPIDelegationToken(
		auth.Principal{UserID: testFirebaseUserID, Email: testUserEmail},
		[]string{"messages:send"},
		"POST",
		"/v1/messages/send",
		ttl,
	)
	require.NoError(t, err)

	claims := parseTestClaims(t, raw, keys.PublicKey())
	require.Len(t, claims.Audience, 1)
	assert.Equal(t, testAPIAudience, claims.Audience[0])
	assert.Equal(t, testMCPIssuer, claims.Issuer)
	assert.Equal(t, testFirebaseUserID, claims.Subject)
	assert.Equal(t, []string{"messages:send"}, claims.Scopes)
	assert.Equal(t, "POST", claims.Method)
	assert.Equal(t, "/v1/messages/send", claims.Path)
	assert.Empty(t, claims.ClientID)

	require.NotNil(t, claims.ExpiresAt)
	assert.WithinDuration(t, before.Add(ttl), claims.ExpiresAt.Time, 5*time.Second)
	assert.False(t, claims.ExpiresAt.Time.After(before.Add(ttl+5*time.Second)))
}

func TestKeySetSignsOnlyRequestedScopes(t *testing.T) {
	keys := newTestKeySet(t)

	raw, err := keys.SignAPIDelegationToken(
		auth.Principal{UserID: testFirebaseUserID},
		[]string{"phones:read"},
		"GET",
		"/v1/phones",
		time.Minute,
	)
	require.NoError(t, err)

	claims := parseTestClaims(t, raw, keys.PublicKey())
	assert.Equal(t, []string{"phones:read"}, claims.Scopes)
	assert.NotContains(t, claims.Scopes, "messages:send")
}

func TestKeySetAPIDelegationTokenHasWireContractFields(t *testing.T) {
	keys := newTestKeySet(t)

	raw, err := keys.SignAPIDelegationToken(
		auth.Principal{UserID: testFirebaseUserID},
		[]string{"phone-api-keys:write"},
		"POST",
		"/v1/phone-api-keys",
		time.Minute,
	)
	require.NoError(t, err)

	// The wire contract with api/pkg/auth.MCPClaims requires exactly these
	// JSON field names: scopes, http_method, http_path.
	assert.True(t, strings.Contains(raw, ".")) // sanity: looks like a JWT

	token, _, err := jwt.NewParser().ParseUnverified(raw, jwt.MapClaims{})
	require.NoError(t, err)

	kid, ok := token.Header["kid"].(string)
	require.True(t, ok)
	assert.Equal(t, testSigningKeyID, kid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, []any{"phone-api-keys:write"}, claims["scopes"])
	assert.Equal(t, "POST", claims["http_method"])
	assert.Equal(t, "/v1/phone-api-keys", claims["http_path"])
	assert.Equal(t, []any{testAPIAudience}, claims["aud"])
	assert.Equal(t, testMCPIssuer, claims["iss"])
	assert.Equal(t, testFirebaseUserID, claims["sub"])
}

func TestKeySetSignMCPAccessTokenRejectsMissingSubject(t *testing.T) {
	keys := newTestKeySet(t)

	_, err := keys.SignMCPAccessToken(auth.Principal{}, "client", []string{"phones:read"}, time.Minute)
	require.Error(t, err)
}

func TestKeySetJWKSPublishesOnlyThePublicKey(t *testing.T) {
	keys := newTestKeySet(t)

	jwks := keys.JWKS()
	require.Len(t, jwks.Keys, 1)

	key := jwks.Keys[0]
	assert.Equal(t, testSigningKeyID, key.Kid)
	assert.Equal(t, "RSA", key.Kty)
	assert.Equal(t, "RS256", key.Alg)
	assert.NotEmpty(t, key.N)
	assert.NotEmpty(t, key.E)
}

func TestKeySetJWKSRoundTripsToAWorkingVerificationKey(t *testing.T) {
	keys := newTestKeySet(t)

	raw, err := keys.SignMCPAccessToken(
		auth.Principal{UserID: testFirebaseUserID},
		"client",
		[]string{"phones:read"},
		time.Minute,
	)
	require.NoError(t, err)

	jwk := keys.JWKS().Keys[0]
	publicKey := rsaPublicKeyFromJWK(t, jwk)

	claims := parseTestClaims(t, raw, publicKey)
	assert.Equal(t, testFirebaseUserID, claims.Subject)
}

// rsaPublicKeyFromJWK reconstructs an *rsa.PublicKey from a JWK's base64url
// modulus/exponent, independently of any production decoding code, so the
// round-trip test exercises exactly the bytes KeySet.JWKS() publishes.
func rsaPublicKeyFromJWK(t *testing.T, jwk auth.JWK) *rsa.PublicKey {
	t.Helper()

	nBytes := mustBase64URLDecode(t, jwk.N)
	eBytes := mustBase64URLDecode(t, jwk.E)

	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}

	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}
}

func mustBase64URLDecode(t *testing.T, s string) []byte {
	t.Helper()

	decoded, err := base64.RawURLEncoding.DecodeString(s)
	require.NoError(t, err)

	return decoded
}
