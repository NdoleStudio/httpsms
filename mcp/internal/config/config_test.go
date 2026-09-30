package config_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/config"
)

// setValidEnv sets every environment variable Load requires to succeed, so
// individual tests can override or unset just the one setting under test.
func setValidEnv(t *testing.T) {
	t.Helper()

	privateKeyPEM := generatedTestPrivateKeyPEM(t)
	t.Setenv("ENV", "local")
	t.Setenv("MCP_BASE_URL", "https://mcp.httpsms.com")
	t.Setenv("HTTPSMS_API_URL", "https://api.httpsms.com")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("FIREBASE_PROJECT_ID", "httpsms")
	t.Setenv("FIREBASE_API_KEY", "test-firebase-api-key")
	t.Setenv("FIREBASE_AUTH_DOMAIN", "httpsms.firebaseapp.com")
	t.Setenv("MCP_SIGNING_PRIVATE_KEY", privateKeyPEM)
	t.Setenv("MCP_SIGNING_PRIVATE_KEY_FILE", "")
	t.Setenv("MCP_SIGNING_KEY_ID", "test-key-1")
}

func TestLoadSucceedsWithAValidEnvironment(t *testing.T) {
	setValidEnv(t)
	privateKeyPEM := generatedTestPrivateKeyPEM(t)

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "local", cfg.Environment)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "https://mcp.httpsms.com", cfg.BaseURL.String())
	assert.Equal(t, "https://api.httpsms.com", cfg.APIURL.String())
	assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
	assert.Equal(t, "httpsms", cfg.FirebaseProjectID)
	assert.Equal(t, "test-key-1", cfg.SigningKeyID)
	assert.Equal(t, []byte(privateKeyPEM), cfg.SigningPrivateKeyPEM)
	assert.Equal(t, "https://mcp.httpsms.com/mcp", cfg.MCPAudience)
	assert.Equal(t, "https://api.httpsms.com", cfg.APIAudience)
	assert.Equal(t, 15*time.Minute, cfg.AccessTokenTTL)
	assert.Equal(t, 2*time.Minute, cfg.APIDelegationTokenTTL)
	assert.Equal(t, 2*time.Minute, cfg.AuthorizationCodeTTL)
	assert.Equal(t, 30*24*time.Hour, cfg.RefreshTokenTTL)
	assert.Equal(t, 5*time.Minute, cfg.ConfirmationTTL)
	assert.Equal(t, 10*time.Second, cfg.HTTPTimeout)
	assert.Equal(t, 120, cfg.ReadToolsPerMinute)
	assert.Equal(t, 30, cfg.SendToolsPerMinute)
	assert.Equal(t, 10, cfg.KeyCreatesPerHour)
	assert.Equal(t, 3, cfg.KeyRotationsPerHour)
	assert.Equal(t, "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com", cfg.FirebaseCertsURL.String())
}

func TestLoadRejectsPartialConfiguration(t *testing.T) {
	setValidEnv(t)
	t.Setenv("HTTPSMS_API_URL", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "HTTPSMS_API_URL")
}

func TestLoadRejectsMissingBaseURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_BASE_URL", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "MCP_BASE_URL")
}

func TestLoadRejectsInvalidBaseURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_BASE_URL", "not-a-url")

	_, err := config.Load()

	require.ErrorContains(t, err, "MCP_BASE_URL")
}

func TestLoadRejectsMissingRedisURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("REDIS_URL", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "REDIS_URL")
}

func TestLoadRejectsMissingFirebaseProjectID(t *testing.T) {
	setValidEnv(t)
	t.Setenv("FIREBASE_PROJECT_ID", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "FIREBASE_PROJECT_ID")
}

func TestLoadRejectsMissingFirebaseAPIKey(t *testing.T) {
	setValidEnv(t)
	t.Setenv("FIREBASE_API_KEY", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "FIREBASE_API_KEY")
}

func TestLoadRejectsMissingFirebaseAuthDomain(t *testing.T) {
	setValidEnv(t)
	t.Setenv("FIREBASE_AUTH_DOMAIN", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "FIREBASE_AUTH_DOMAIN")
}

func TestLoadRejectsMissingSigningKeyID(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_SIGNING_KEY_ID", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "MCP_SIGNING_KEY_ID")
}

func TestLoadRejectsMissingSigningKeyMaterial(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_SIGNING_PRIVATE_KEY", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "MCP_SIGNING_PRIVATE_KEY")
}

func TestLoadRejectsBothSigningKeyEnvAndFileSet(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_SIGNING_PRIVATE_KEY_FILE", "some-file.pem")

	_, err := config.Load()

	require.ErrorContains(t, err, "not both")
}

func TestLoadRejectsMalformedSigningKeyPEM(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_SIGNING_PRIVATE_KEY", "not a pem block")

	_, err := config.Load()

	require.ErrorContains(t, err, "MCP_SIGNING_PRIVATE_KEY")
}

func TestLoadReadsSigningKeyFromFile(t *testing.T) {
	setValidEnv(t)
	privateKeyPEM := generatedTestPrivateKeyPEM(t)
	t.Setenv("MCP_SIGNING_PRIVATE_KEY", "")
	keyFile := writeTempKeyFile(t, privateKeyPEM)
	t.Setenv("MCP_SIGNING_PRIVATE_KEY_FILE", keyFile)

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, []byte(privateKeyPEM), cfg.SigningPrivateKeyPEM)
}

func TestLoadRejectsUnreadableSigningKeyFile(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_SIGNING_PRIVATE_KEY", "")
	t.Setenv("MCP_SIGNING_PRIVATE_KEY_FILE", "does-not-exist.pem")

	_, err := config.Load()

	require.ErrorContains(t, err, "MCP_SIGNING_PRIVATE_KEY_FILE")
}

func TestLoadRejectsInvalidAccessTokenTTL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_ACCESS_TOKEN_TTL", "not-a-duration")

	_, err := config.Load()

	require.ErrorContains(t, err, "MCP_ACCESS_TOKEN_TTL")
}

func TestLoadRejectsNonPositiveRefreshTokenTTL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("REFRESH_TOKEN_TTL", "0s")

	_, err := config.Load()

	require.ErrorContains(t, err, "REFRESH_TOKEN_TTL")
}

func TestLoadRejectsInvalidHTTPTimeout(t *testing.T) {
	setValidEnv(t)
	t.Setenv("HTTP_TIMEOUT", "-5s")

	_, err := config.Load()

	require.ErrorContains(t, err, "HTTP_TIMEOUT")
}

func TestLoadRequiresHTTPSForEveryURLInProduction(t *testing.T) {
	setValidEnv(t)
	t.Setenv("ENV", "production")
	t.Setenv("MCP_BASE_URL", "http://mcp.httpsms.com")

	_, err := config.Load()

	require.ErrorContains(t, err, "MCP_BASE_URL")
	require.ErrorContains(t, err, "https")
}

func TestLoadRequiresHTTPSForAPIURLInProduction(t *testing.T) {
	setValidEnv(t)
	t.Setenv("ENV", "production")
	t.Setenv("HTTPSMS_API_URL", "http://api.httpsms.com")

	_, err := config.Load()

	require.ErrorContains(t, err, "HTTPSMS_API_URL")
	require.ErrorContains(t, err, "https")
}

func TestLoadAllowsHTTPURLsOutsideProduction(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MCP_BASE_URL", "http://localhost:8090")
	t.Setenv("HTTPSMS_API_URL", "http://localhost:8000")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "http", cfg.BaseURL.Scheme)
	assert.Equal(t, "http", cfg.APIURL.Scheme)
}

func TestLoadOverridesRateLimitDefaults(t *testing.T) {
	setValidEnv(t)
	t.Setenv("READ_TOOLS_PER_MINUTE", "60")
	t.Setenv("SEND_TOOLS_PER_MINUTE", "15")
	t.Setenv("KEY_CREATES_PER_HOUR", "5")
	t.Setenv("KEY_ROTATIONS_PER_HOUR", "1")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, 60, cfg.ReadToolsPerMinute)
	assert.Equal(t, 15, cfg.SendToolsPerMinute)
	assert.Equal(t, 5, cfg.KeyCreatesPerHour)
	assert.Equal(t, 1, cfg.KeyRotationsPerHour)
}

func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	setValidEnv(t)
	t.Setenv("HTTPSMS_API_URL", "")
	t.Setenv("REDIS_URL", "")

	_, err := config.Load()

	require.ErrorContains(t, err, "HTTPSMS_API_URL")
	require.ErrorContains(t, err, "REDIS_URL")
}

var (
	testPrivateKeyOnce sync.Once
	testPrivateKeyPEM  string
	testPrivateKeyErr  error
)

func generatedTestPrivateKeyPEM(t *testing.T) string {
	t.Helper()

	testPrivateKeyOnce.Do(func() {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			testPrivateKeyErr = err
			return
		}

		testPrivateKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}))
	})

	require.NoError(t, testPrivateKeyErr)
	return testPrivateKeyPEM
}

// writeTempKeyFile writes contents to a new file inside t.TempDir() and
// returns its path.
func writeTempKeyFile(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := dir + "/signing-key.pem"

	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}
