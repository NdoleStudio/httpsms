package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
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

// TestServeReturnsAnErrorWhenTheListenerFails asserts a listener that can
// never start (here: an unparseable address, standing in for a port already
// in use or a PORT value Cloud Run could not bind) surfaces as a non-nil
// error from serve -- which is what makes main exit non-zero instead of
// logging and "shutting down" as if it had served successfully.
func TestServeReturnsAnErrorWhenTheListenerFails(t *testing.T) {
	shutdownCalls := 0
	shutdown := func(context.Context) error {
		shutdownCalls++
		return nil
	}

	err := serve(context.Background(), "not-an-address", http.NewServeMux(), shutdown)

	require.Error(t, err)
	require.Contains(t, err.Error(), "not-an-address")
	// Dependencies opened before serving must still be released.
	require.Equal(t, 1, shutdownCalls)
}

// TestServeReturnsNilOnGracefulShutdown asserts the normal Cloud Run path
// -- SIGTERM cancels the context -- drains and returns nil, so main exits
// zero.
func TestServeReturnsNilOnGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	shutdownCalls := 0
	shutdown := func(context.Context) error {
		shutdownCalls++
		return nil
	}

	done := make(chan error, 1)
	go func() { done <- serve(ctx, "127.0.0.1:0", mux, shutdown) }()

	// Give the listener a moment to start, then ask for shutdown.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("serve did not return after its context was cancelled")
	}

	require.Equal(t, 1, shutdownCalls)
}

// TestServeReportsShutdownFailures asserts a dependency that fails to close
// is also a non-zero exit, not a silent log line.
func TestServeReportsShutdownFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := serve(ctx, "127.0.0.1:0", http.NewServeMux(), func(context.Context) error {
		return errors.New("redis close failed")
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "redis close failed")
}

// TestRunFailsWhenConfigurationIsInvalid asserts run surfaces a
// configuration error as an error return (a non-zero exit) rather than
// exiting deep inside the call stack.
func TestRunFailsWhenConfigurationIsInvalid(t *testing.T) {
	t.Setenv("MCP_BASE_URL", "")
	t.Setenv("REDIS_URL", "")

	err := run(context.Background(), "test")

	require.Error(t, err)
	require.Contains(t, err.Error(), "load configuration")
}
