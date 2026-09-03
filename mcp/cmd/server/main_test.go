package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/config"
)

// generateTestSigningKeyPEM returns a fresh PKCS#1-encoded RSA private key,
// suitable for MCP_SIGNING_PRIVATE_KEY in tests.
func generateTestSigningKeyPEM(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}

// setTestEnv sets every environment variable config.Load requires,
// pointing REDIS_URL at mr, and returns a cleanup func that restores every
// variable this func touched to its previous value.
func setTestEnv(t *testing.T, mr *miniredis.Miniredis) {
	t.Helper()

	env := map[string]string{
		"ENV":                     "test",
		"MCP_BASE_URL":            "https://mcp.httpsms.test",
		"HTTPSMS_API_URL":         "https://api.httpsms.test",
		"REDIS_URL":               "redis://" + mr.Addr(),
		"FIREBASE_PROJECT_ID":     "httpsms-test",
		"FIREBASE_API_KEY":        "test-firebase-api-key",
		"FIREBASE_AUTH_DOMAIN":    "httpsms-test.firebaseapp.com",
		"MCP_SIGNING_PRIVATE_KEY": generateTestSigningKeyPEM(t),
		"MCP_SIGNING_KEY_ID":      "test-key-1",
	}

	for key, value := range env {
		t.Setenv(key, value)
	}
	_ = os.Unsetenv("MCP_SIGNING_PRIVATE_KEY_FILE")
}

// TestBuildAssemblesAWorkingHandler is this package's local smoke test
// (brief Step 6): it loads configuration from environment variables set to
// point at an in-process miniredis instance, calls build, and exercises the
// resulting handler's health, metadata, and bearer-auth-rejection routes
// over real HTTP.
func TestBuildAssemblesAWorkingHandler(t *testing.T) {
	mr := miniredis.RunT(t)
	setTestEnv(t, mr)

	cfg, err := config.Load()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler, shutdown, err := build(ctx, cfg, "test")
	require.NoError(t, err)
	require.NotNil(t, handler)
	require.NotNil(t, shutdown)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		require.NoError(t, shutdown(shutdownCtx))
	}()

	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	resp, err := http.Get(httpServer.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp2, err := http.Get(httpServer.URL + "/.well-known/oauth-protected-resource")
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/mcp", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp3, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp3.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp3.StatusCode)
}

// TestBuildFailsFastOnInvalidConfiguration exercises build's own error
// path: an unparseable REDIS_URL is a build-time (not request-time) error.
func TestBuildFailsFastOnInvalidConfiguration(t *testing.T) {
	mr := miniredis.RunT(t)
	setTestEnv(t, mr)
	t.Setenv("REDIS_URL", "not-a-valid-redis-url")

	cfg, err := config.Load()
	require.NoError(t, err)

	handler, shutdown, err := build(context.Background(), cfg, "test")
	require.Error(t, err)
	require.Nil(t, handler)
	require.Nil(t, shutdown)
}
