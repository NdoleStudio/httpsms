package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NdoleStudio/stacktrace"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRSAKey(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return privateKey, &privateKey.PublicKey
}

func testJWKSHandler(_ *testing.T, kid string, publicKey *rsa.PublicKey) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		set := mcpJWKSet{
			Keys: []mcpJWK{
				{
					Kty: "RSA",
					Kid: kid,
					N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
					E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(set)
	}
}

func signDelegatedToken(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims MCPClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	raw, err := token.SignedString(privateKey)
	require.NoError(t, err)

	return raw
}

func testMCPClaims(subject string, scopes []string, method string, path string) MCPClaims {
	now := time.Now()
	return MCPClaims{
		Scopes: scopes,
		Method: method,
		Path:   path,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://mcp.httpsms.com",
			Subject:   subject,
			Audience:  jwt.ClaimStrings{"https://api.httpsms.com"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
}

func newTestVerifier(t *testing.T, jwksURL string, jwksClient *http.Client) *MCPTokenVerifier {
	t.Helper()

	verifier, err := NewMCPTokenVerifier(MCPTokenVerifierConfig{
		Issuer:     "https://mcp.httpsms.com",
		Audience:   "https://api.httpsms.com",
		JWKSURL:    jwksURL,
		HTTPClient: jwksClient,
		// Refresh throttling is exercised on its own in mcp_jwks_test.go; the tests using
		// this helper assert unrelated verification behavior, so every unknown "kid" here
		// is allowed to refresh immediately.
		MinRefreshInterval: time.Nanosecond,
	})
	require.NoError(t, err)

	return verifier
}

func TestNewMCPTokenVerifier_RequiresIssuerAudienceAndJWKSURL(t *testing.T) {
	tests := []struct {
		name   string
		config MCPTokenVerifierConfig
	}{
		{name: "missing issuer", config: MCPTokenVerifierConfig{Audience: "aud", JWKSURL: "https://example.com"}},
		{name: "missing audience", config: MCPTokenVerifierConfig{Issuer: "iss", JWKSURL: "https://example.com"}},
		{name: "missing jwks url", config: MCPTokenVerifierConfig{Issuer: "iss", Audience: "aud"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMCPTokenVerifier(tt.config)
			require.Error(t, err)
			assert.Equal(t, ErrCodeInvalidToken, stacktrace.GetCode(err))
		})
	}
}

func TestMCPTokenVerifierVerifyRequest(t *testing.T) {
	privateKey, publicKey := testRSAKey(t)
	jwks := httptest.NewServer(testJWKSHandler(t, "test-key", publicKey))
	defer jwks.Close()

	verifier := newTestVerifier(t, jwks.URL, jwks.Client())

	raw := signDelegatedToken(t, privateKey, "test-key", testMCPClaims(
		"firebase-user-id",
		[]string{"messages:read"},
		http.MethodGet,
		"/v1/messages",
	))

	claims, err := verifier.VerifyRequest(context.Background(), raw, http.MethodGet, "/v1/messages")

	require.NoError(t, err)
	assert.Equal(t, "firebase-user-id", claims.Subject)
	assert.Equal(t, []string{"messages:read"}, claims.Scopes)
}

func TestMCPTokenVerifierVerifyRequest_RotateAPIKeyRouteMatchesUserIDWildcard(t *testing.T) {
	privateKey, publicKey := testRSAKey(t)
	jwks := httptest.NewServer(testJWKSHandler(t, "test-key", publicKey))
	defer jwks.Close()

	verifier := newTestVerifier(t, jwks.URL, jwks.Client())

	raw := signDelegatedToken(t, privateKey, "test-key", testMCPClaims(
		"firebase-user-id",
		[]string{"user-api-key:rotate"},
		http.MethodDelete,
		"/v1/users/firebase-user-id/api-keys",
	))

	claims, err := verifier.VerifyRequest(context.Background(), raw, http.MethodDelete, "/v1/users/firebase-user-id/api-keys")

	require.NoError(t, err)
	assert.Equal(t, "firebase-user-id", claims.Subject)
}

func TestMCPTokenVerifierVerifyRequest_Failures(t *testing.T) {
	privateKey, publicKey := testRSAKey(t)
	otherPrivateKey, _ := testRSAKey(t)
	jwks := httptest.NewServer(testJWKSHandler(t, "test-key", publicKey))
	defer jwks.Close()

	verifier := newTestVerifier(t, jwks.URL, jwks.Client())

	tests := []struct {
		name         string
		raw          string
		method       string
		path         string
		expectedCode stacktrace.ErrorCode
	}{
		{
			name: "wrong issuer",
			raw: func() string {
				claims := testMCPClaims("firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages")
				claims.Issuer = "https://not-mcp.httpsms.com"
				return signDelegatedToken(t, privateKey, "test-key", claims)
			}(),
			method:       http.MethodGet,
			path:         "/v1/messages",
			expectedCode: ErrCodeInvalidToken,
		},
		{
			name: "wrong audience",
			raw: func() string {
				claims := testMCPClaims("firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages")
				claims.Audience = jwt.ClaimStrings{"https://not-api.httpsms.com"}
				return signDelegatedToken(t, privateKey, "test-key", claims)
			}(),
			method:       http.MethodGet,
			path:         "/v1/messages",
			expectedCode: ErrCodeInvalidToken,
		},
		{
			name: "expired token",
			raw: func() string {
				claims := testMCPClaims("firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages")
				claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
				return signDelegatedToken(t, privateKey, "test-key", claims)
			}(),
			method:       http.MethodGet,
			path:         "/v1/messages",
			expectedCode: ErrCodeInvalidToken,
		},
		{
			name: "unknown kid",
			raw: signDelegatedToken(t, privateKey, "unknown-key", testMCPClaims(
				"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
			)),
			method:       http.MethodGet,
			path:         "/v1/messages",
			expectedCode: ErrCodeInvalidToken,
		},
		{
			name: "signed by wrong key",
			raw: signDelegatedToken(t, otherPrivateKey, "test-key", testMCPClaims(
				"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
			)),
			method:       http.MethodGet,
			path:         "/v1/messages",
			expectedCode: ErrCodeInvalidToken,
		},
		{
			name: "missing subject",
			raw: func() string {
				claims := testMCPClaims("", []string{"messages:read"}, http.MethodGet, "/v1/messages")
				return signDelegatedToken(t, privateKey, "test-key", claims)
			}(),
			method:       http.MethodGet,
			path:         "/v1/messages",
			expectedCode: ErrCodeInvalidToken,
		},
		{
			name: "missing required scope",
			raw: signDelegatedToken(t, privateKey, "test-key", testMCPClaims(
				"firebase-user-id", []string{"phones:read"}, http.MethodGet, "/v1/messages",
			)),
			method:       http.MethodGet,
			path:         "/v1/messages",
			expectedCode: ErrCodeInsufficientScope,
		},
		{
			name: "path does not match token binding",
			raw: signDelegatedToken(t, privateKey, "test-key", testMCPClaims(
				"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
			)),
			method:       http.MethodGet,
			path:         "/v1/message-threads",
			expectedCode: ErrCodeOperationDenied,
		},
		{
			name: "method does not match token binding",
			raw: signDelegatedToken(t, privateKey, "test-key", testMCPClaims(
				"firebase-user-id", []string{"messages:send"}, http.MethodPost, "/v1/messages/send",
			)),
			method:       http.MethodDelete,
			path:         "/v1/messages/send",
			expectedCode: ErrCodeOperationDenied,
		},
		{
			name: "route is not an approved MCP operation",
			raw: signDelegatedToken(t, privateKey, "test-key", testMCPClaims(
				"firebase-user-id", []string{"messages:read"}, http.MethodDelete, "/v1/users/firebase-user-id",
			)),
			method:       http.MethodDelete,
			path:         "/v1/users/firebase-user-id",
			expectedCode: ErrCodeOperationDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := verifier.VerifyRequest(context.Background(), tt.raw, tt.method, tt.path)

			require.Error(t, err)
			require.Nil(t, claims)
			assert.Equal(t, tt.expectedCode, stacktrace.GetCode(err))
		})
	}
}

func TestMCPTokenVerifierVerifyRequest_RefreshesJWKSOnceWhenKeyIsRotated(t *testing.T) {
	firstPrivateKey, firstPublicKey := testRSAKey(t)
	secondPrivateKey, secondPublicKey := testRSAKey(t)

	requestCount := 0
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			testJWKSHandler(t, "key-1", firstPublicKey)(w, r)
			return
		}
		testJWKSHandler(t, "key-2", secondPublicKey)(w, r)
	}))
	defer jwks.Close()

	verifier := newTestVerifier(t, jwks.URL, jwks.Client())

	firstToken := signDelegatedToken(t, firstPrivateKey, "key-1", testMCPClaims(
		"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
	))
	_, err := verifier.VerifyRequest(context.Background(), firstToken, http.MethodGet, "/v1/messages")
	require.NoError(t, err)
	assert.Equal(t, 1, requestCount)

	// The verifier's cache only has "key-1"; a token signed with the newly rotated "key-2"
	// forces exactly one additional JWKS refresh before it can be verified.
	secondToken := signDelegatedToken(t, secondPrivateKey, "key-2", testMCPClaims(
		"firebase-user-id", []string{"messages:read"}, http.MethodGet, "/v1/messages",
	))
	claims, err := verifier.VerifyRequest(context.Background(), secondToken, http.MethodGet, "/v1/messages")
	require.NoError(t, err)
	assert.Equal(t, "firebase-user-id", claims.Subject)
	assert.Equal(t, 2, requestCount)
}
