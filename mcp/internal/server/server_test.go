package server_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/config"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
	"github.com/NdoleStudio/httpsms/mcp/internal/server"
)

const (
	testIssuer      = "https://mcp.httpsms.test"
	testAPIAud      = "https://api.httpsms.test"
	testFirebaseUID = "firebase-uid-1"
)

// approvingVerifier is a fixed-response auth.IdentityVerifier test double.
type approvingVerifier struct{}

func (approvingVerifier) Verify(context.Context, string) (auth.Principal, error) {
	return auth.Principal{UserID: testFirebaseUID, Email: "user@example.com"}, nil
}

// stubAPIClient is a no-op httpsms.Client test double. Every server_test.go
// test exercises protocol-level behavior (metadata, auth, protocol
// negotiation, tools/list) that never actually invokes a tool handler, so
// every method here is unreachable in practice; they exist only to satisfy
// the httpsms.Client interface.
type stubAPIClient struct{}

func (stubAPIClient) ListPhones(context.Context, string, httpsms.ListPhonesParams) ([]httpsms.Phone, error) {
	return nil, nil
}

func (stubAPIClient) SendSMS(context.Context, string, httpsms.SendSMSParams) (httpsms.Message, error) {
	return httpsms.Message{}, nil
}

func (stubAPIClient) ListMessageThreads(context.Context, string, httpsms.ListMessageThreadsParams) ([]httpsms.MessageThread, error) {
	return nil, nil
}

func (stubAPIClient) ListThreadMessages(context.Context, string, httpsms.ListThreadMessagesParams) ([]httpsms.Message, error) {
	return nil, nil
}

func (stubAPIClient) ListIncomingMessages(context.Context, string, httpsms.ListIncomingMessagesParams) ([]httpsms.Message, error) {
	return nil, nil
}

func (stubAPIClient) CreatePhoneAPIKey(context.Context, string, httpsms.CreatePhoneAPIKeyParams) (httpsms.PhoneAPIKey, error) {
	return httpsms.PhoneAPIKey{}, nil
}

func (stubAPIClient) RotateUserAPIKey(context.Context, string, string) (httpsms.User, error) {
	return httpsms.User{}, nil
}

var _ httpsms.Client = stubAPIClient{}

// newTestConfig returns a valid config.Config for tests, backed by mr's
// address as its Redis URL.
func newTestConfig(t *testing.T, mr *miniredis.Miniredis) config.Config {
	t.Helper()

	baseURL, err := url.Parse(testIssuer)
	require.NoError(t, err)
	apiURL, err := url.Parse(testAPIAud)
	require.NoError(t, err)

	return config.Config{
		Environment:           "test",
		Port:                  "0",
		BaseURL:               baseURL,
		APIURL:                apiURL,
		RedisURL:              "redis://" + mr.Addr(),
		MCPAudience:           testIssuer + "/mcp",
		APIAudience:           testAPIAud,
		AccessTokenTTL:        15 * time.Minute,
		APIDelegationTokenTTL: 2 * time.Minute,
		AuthorizationCodeTTL:  2 * time.Minute,
		RefreshTokenTTL:       30 * 24 * time.Hour,
		ConfirmationTTL:       5 * time.Minute,
		ReadToolsPerMinute:    120,
		SendToolsPerMinute:    30,
		KeyCreatesPerHour:     10,
		KeyRotationsPerHour:   3,
	}
}

// newTestKeys returns a KeySet configured against cfg's issuer and
// audiences.
func newTestKeys(t *testing.T, cfg config.Config) *auth.KeySet {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	keys, err := auth.NewKeySet(keyPEM, "test-key-1")
	require.NoError(t, err)
	require.NoError(t, keys.Configure(strings.TrimRight(cfg.BaseURL.String(), "/"), cfg.MCPAudience, cfg.APIAudience))

	return keys
}

// testHarness bundles every dependency server.New needs plus a running
// httptest.Server exposing the assembled handler.
type testHarness struct {
	httpServer *httptest.Server
	keys       *auth.KeySet
	cfg        config.Config
}

// newTestHarness assembles server.New's dependencies against a fresh
// miniredis instance and starts an httptest.Server serving the result.
func newTestHarness(t *testing.T, mutate ...func(*config.Config)) *testHarness {
	t.Helper()

	return newTestHarnessWithAPIClient(t, stubAPIClient{}, mutate...)
}

// newTestHarnessWithAPIClient is newTestHarness with an explicit
// httpsms.Client, for tests that need the client to misbehave (see the
// panic-recovery tests).
func newTestHarnessWithAPIClient(t *testing.T, apiClient httpsms.Client, mutate ...func(*config.Config)) *testHarness {
	t.Helper()

	mr := miniredis.RunT(t)
	cfg := newTestConfig(t, mr)
	for _, m := range mutate {
		m(&cfg)
	}
	keys := newTestKeys(t, cfg)

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	store := oauth.NewRedisStore(redisClient)
	resolver := oauth.NewClientResolver(http.DefaultClient, store)

	issuer := strings.TrimRight(cfg.BaseURL.String(), "/")
	oauthServerConfig := oauth.ServerConfig{
		Issuer:               issuer,
		Resource:             cfg.MCPAudience,
		FirebaseAPIKey:       "test-firebase-api-key",
		FirebaseAuthDomain:   "httpsms-test.firebaseapp.com",
		AuthorizationCodeTTL: cfg.AuthorizationCodeTTL,
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
	}

	oauthServer, err := oauth.NewServer(store, resolver, keys, approvingVerifier{}, oauthServerConfig)
	require.NoError(t, err)

	handler, err := server.New(cfg, server.Dependencies{
		Logger:                zerolog.Nop(),
		Keys:                  keys,
		OAuthServer:           oauthServer,
		OAuthServerConfig:     oauthServerConfig,
		OAuthStore:            store,
		APIClient:             apiClient,
		RedisClient:           redisClient,
		APIDelegationTokenTTL: cfg.APIDelegationTokenTTL,
		ConfirmationTTL:       cfg.ConfirmationTTL,
		RateLimits: server.Limits{
			ReadPerMinute:       cfg.ReadToolsPerMinute,
			SendPerMinute:       cfg.SendToolsPerMinute,
			KeyCreatesPerHour:   cfg.KeyCreatesPerHour,
			KeyRotationsPerHour: cfg.KeyRotationsPerHour,
		},
		Version: "test",
	})
	require.NoError(t, err)

	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	return &testHarness{httpServer: httpServer, keys: keys, cfg: cfg}
}

// mintToken mints a fixed-scope MCP access token for the harness's test
// principal.
func (h *testHarness) mintToken(t *testing.T, scopes ...string) string {
	t.Helper()

	token, err := h.keys.SignMCPAccessToken(auth.Principal{UserID: testFirebaseUID, Email: "user@example.com"}, "test-client", scopes, 15*time.Minute)
	require.NoError(t, err)
	return token
}

var allScopes = []string{
	auth.ScopePhonesRead,
	auth.ScopeMessagesRead,
	auth.ScopeMessagesSend,
	auth.ScopePhoneAPIKeysWrite,
	auth.ScopeUserAPIKeyRotate,
}

// --- Step 1: route and protocol tests -------------------------------------

func TestHealthRoutesReturn200(t *testing.T) {
	h := newTestHarness(t)

	for _, path := range []string{"/healthz", "/health"} {
		resp, err := http.Get(h.httpServer.URL + path)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equalf(t, http.StatusOK, resp.StatusCode, "GET %s", path)
	}
}

func TestMetadataJWKSAndRegistrationRoutesAreMounted(t *testing.T) {
	h := newTestHarness(t)

	resp, err := http.Get(h.httpServer.URL + "/.well-known/oauth-protected-resource")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var prm struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&prm))
	require.Equal(t, testIssuer+"/mcp", prm.Resource)

	resp2, err := http.Get(h.httpServer.URL + "/.well-known/oauth-authorization-server")
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	var asm struct {
		Issuer               string `json:"issuer"`
		TokenEndpoint        string `json:"token_endpoint"`
		RegistrationEndpoint string `json:"registration_endpoint"`
	}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&asm))
	require.Equal(t, testIssuer, asm.Issuer)
	require.Equal(t, testIssuer+"/oauth/token", asm.TokenEndpoint)
	require.Equal(t, testIssuer+"/oauth/register", asm.RegistrationEndpoint)

	resp3, err := http.Get(h.httpServer.URL + "/.well-known/jwks.json")
	require.NoError(t, err)
	defer resp3.Body.Close()
	require.Equal(t, http.StatusOK, resp3.StatusCode)
	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	require.NoError(t, json.NewDecoder(resp3.Body).Decode(&jwks))
	require.Len(t, jwks.Keys, 1)

	registrationBody := `{
		"client_name": "test-client",
		"redirect_uris": ["https://client.example/callback"],
		"grant_types": ["authorization_code", "refresh_token"],
		"response_types": ["code"],
		"token_endpoint_auth_method": "none"
	}`
	resp4, err := http.Post(h.httpServer.URL+"/oauth/register", "application/json", strings.NewReader(registrationBody))
	require.NoError(t, err)
	defer resp4.Body.Close()
	require.Equal(t, http.StatusCreated, resp4.StatusCode)
}

func TestAuthorizeTokenAndFirebaseCompleteRoutesAreMounted(t *testing.T) {
	h := newTestHarness(t)

	// A minimal (invalid) authorize request is still routed to
	// oauth.Server.HandleAuthorize -- a 404 here would mean the route is
	// not mounted at all, which is what this test guards against.
	resp, err := http.Get(h.httpServer.URL + "/oauth/authorize")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NotEqual(t, http.StatusNotFound, resp.StatusCode)

	resp2, err := http.PostForm(h.httpServer.URL+"/oauth/token", url.Values{})
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.NotEqual(t, http.StatusNotFound, resp2.StatusCode)
	require.Equal(t, "no-store", resp2.Header.Get("Cache-Control"))

	// POST /oauth/firebase/complete is mounted too; a 404 here would mean
	// the route itself is missing (a malformed/empty form body still
	// reaches oauth.Server.HandleFirebaseComplete and is rejected with a
	// client error, never a 404).
	resp3, err := http.PostForm(h.httpServer.URL+"/oauth/firebase/complete", url.Values{})
	require.NoError(t, err)
	defer resp3.Body.Close()
	require.NotEqual(t, http.StatusNotFound, resp3.StatusCode)
}

func TestUnauthenticatedMCPReturns401AndProtectedResourceMetadata(t *testing.T) {
	h := newTestHarness(t)

	req, err := http.NewRequest(http.MethodPost, h.httpServer.URL+"/mcp", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	wwwAuthenticate := resp.Header.Get("WWW-Authenticate")
	require.Contains(t, wwwAuthenticate, "Bearer")
	require.Contains(t, wwwAuthenticate, "resource_metadata=")
	require.Contains(t, wwwAuthenticate, "/.well-known/oauth-protected-resource")
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
}

// postMCP issues an authenticated POST /mcp request carrying body, with the
// given extra headers set in addition to Content-Type and Accept.
func postMCP(t *testing.T, h *testHarness, token string, body string, headers map[string]string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, h.httpServer.URL+"/mcp", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func TestAuthenticatedServerDiscoverNegotiatesLatestProtocolVersion(t *testing.T) {
	h := newTestHarness(t)
	token := h.mintToken(t, allScopes...)

	body := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{` +
		`"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
		`"io.modelcontextprotocol/clientCapabilities":{}` +
		`}}}`

	resp := postMCP(t, h, token, body, map[string]string{
		"Mcp-Protocol-Version": "2026-07-28",
		"Mcp-Method":           "server/discover",
	})
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "body: %s", respBody)

	var decoded struct {
		Result struct {
			SupportedVersions []string `json:"supportedVersions"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(respBody, &decoded))
	require.NotEmpty(t, decoded.Result.SupportedVersions)
	require.Equal(t, "2026-07-28", decoded.Result.SupportedVersions[0])
	require.Contains(t, decoded.Result.SupportedVersions, "2025-11-25")
}

func TestLegacyInitializeNegotiates20251125(t *testing.T) {
	h := newTestHarness(t)
	token := h.mintToken(t, allScopes...)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{` +
		`"protocolVersion":"2025-11-25",` +
		`"capabilities":{},` +
		`"clientInfo":{"name":"legacy-test-client","version":"1.0"}` +
		`}}`

	resp := postMCP(t, h, token, body, nil)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "body: %s", respBody)

	var decoded struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(respBody, &decoded))
	require.Equal(t, "2025-11-25", decoded.Result.ProtocolVersion)
}

func TestToolsListOrderIsDeterministic(t *testing.T) {
	h := newTestHarness(t)
	token := h.mintToken(t, allScopes...)

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`

	resp := postMCP(t, h, token, body, nil)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "body: %s", respBody)

	var decoded struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(respBody, &decoded))

	names := make([]string, len(decoded.Result.Tools))
	for i, tool := range decoded.Result.Tools {
		names[i] = tool.Name
	}

	// The SDK's tool set iterates in sorted-by-name order (see
	// mcp.featureSet); asserting the exact order here means a future SDK
	// upgrade that changed this iteration order would be caught here
	// rather than surfacing as a confusing client-side ordering bug.
	require.Equal(t, []string{
		"create_phone_api_key",
		"list_incoming_messages",
		"list_message_threads",
		"list_phones",
		"list_thread_messages",
		"rotate_user_api_key",
		"send_sms",
	}, names)
}

func TestGetAndDeleteMCPAreRejectedInStatelessMode(t *testing.T) {
	h := newTestHarness(t)
	token := h.mintToken(t, allScopes...)

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		req, err := http.NewRequest(method, h.httpServer.URL+"/mcp", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equalf(t, http.StatusMethodNotAllowed, resp.StatusCode, "method %s", method)
	}
}

func TestPublicMetadataRoutesSetPermissiveNonCredentialedCORS(t *testing.T) {
	h := newTestHarness(t)

	for _, path := range []string{
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-authorization-server",
		"/.well-known/jwks.json",
	} {
		resp, err := http.Get(h.httpServer.URL + path)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equalf(t, "*", resp.Header.Get("Access-Control-Allow-Origin"), "path %s", path)
		require.Emptyf(t, resp.Header.Get("Access-Control-Allow-Credentials"), "path %s", path)
	}
}

func TestSecretResultAndErrorResponsesOnMCPAreNeverCached(t *testing.T) {
	h := newTestHarness(t)

	// Unauthenticated (error) response.
	req, err := http.NewRequest(http.MethodPost, h.httpServer.URL+"/mcp", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

	// Authenticated (success) response.
	token := h.mintToken(t, allScopes...)
	resp2 := postMCP(t, h, token, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`, nil)
	defer resp2.Body.Close()
	require.Equal(t, "no-store", resp2.Header.Get("Cache-Control"))
}

func TestToolRateLimitIsEnforcedBeforeToolExecution(t *testing.T) {
	h := newTestHarness(t, func(cfg *config.Config) {
		cfg.ReadToolsPerMinute = 1
	})
	token := h.mintToken(t, allScopes...)

	callListPhones := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_phones","arguments":{}}}`

	resp1 := postMCP(t, h, token, callListPhones, nil)
	defer resp1.Body.Close()
	body1, err := io.ReadAll(resp1.Body)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp1.StatusCode, "body: %s", body1)

	resp2 := postMCP(t, h, token, callListPhones, nil)
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)

	var decoded struct {
		Error *struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body2, &decoded))
	require.NotNilf(t, decoded.Error, "expected the second call to be rate limited, body: %s", body2)
	require.Contains(t, strings.ToLower(decoded.Error.Message), "rate limit")

	var data struct {
		Tool              string `json:"tool"`
		RetryAfterSeconds int    `json:"retry_after_seconds"`
	}
	require.NoError(t, json.Unmarshal(decoded.Error.Data, &data))
	require.Equal(t, "list_phones", data.Tool)
	require.GreaterOrEqual(t, data.RetryAfterSeconds, 1)
}

// --- Dependency validation / Task 5 audience-consistency ruling ----------

func TestNewRejectsMismatchedOAuthResourceAndConfigAudience(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := newTestConfig(t, mr)
	keys := newTestKeys(t, cfg)

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	store := oauth.NewRedisStore(redisClient)
	resolver := oauth.NewClientResolver(http.DefaultClient, store)

	mismatchedConfig := oauth.ServerConfig{
		Issuer:               testIssuer,
		Resource:             "https://mcp.httpsms.test/wrong-resource",
		FirebaseAPIKey:       "test-firebase-api-key",
		FirebaseAuthDomain:   "httpsms-test.firebaseapp.com",
		AuthorizationCodeTTL: cfg.AuthorizationCodeTTL,
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
	}
	oauthServer, err := oauth.NewServer(store, resolver, keys, approvingVerifier{}, mismatchedConfig)
	require.NoError(t, err)

	_, err = server.New(cfg, server.Dependencies{
		Logger:                zerolog.Nop(),
		Keys:                  keys,
		OAuthServer:           oauthServer,
		OAuthServerConfig:     mismatchedConfig,
		OAuthStore:            store,
		APIClient:             stubAPIClient{},
		RedisClient:           redisClient,
		APIDelegationTokenTTL: cfg.APIDelegationTokenTTL,
		ConfirmationTTL:       cfg.ConfirmationTTL,
		RateLimits:            newTestLimits(cfg),
		Version:               "test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Resource")
	require.Contains(t, err.Error(), "MCPAudience")
}

// newTestLimits returns the server.Limits matching cfg's configured
// budgets.
func newTestLimits(cfg config.Config) server.Limits {
	return server.Limits{
		ReadPerMinute:       cfg.ReadToolsPerMinute,
		SendPerMinute:       cfg.SendToolsPerMinute,
		KeyCreatesPerHour:   cfg.KeyCreatesPerHour,
		KeyRotationsPerHour: cfg.KeyRotationsPerHour,
	}
}

func TestNewRejectsIncompleteDependencies(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := newTestConfig(t, mr)
	keys := newTestKeys(t, cfg)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()
	store := oauth.NewRedisStore(redisClient)
	resolver := oauth.NewClientResolver(http.DefaultClient, store)
	oauthServerConfig := oauth.ServerConfig{
		Issuer:               testIssuer,
		Resource:             cfg.MCPAudience,
		FirebaseAPIKey:       "test-firebase-api-key",
		FirebaseAuthDomain:   "httpsms-test.firebaseapp.com",
		AuthorizationCodeTTL: cfg.AuthorizationCodeTTL,
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
	}
	oauthServer, err := oauth.NewServer(store, resolver, keys, approvingVerifier{}, oauthServerConfig)
	require.NoError(t, err)

	complete := server.Dependencies{
		Logger:                zerolog.Nop(),
		Keys:                  keys,
		OAuthServer:           oauthServer,
		OAuthServerConfig:     oauthServerConfig,
		OAuthStore:            store,
		APIClient:             stubAPIClient{},
		RedisClient:           redisClient,
		APIDelegationTokenTTL: cfg.APIDelegationTokenTTL,
		ConfirmationTTL:       cfg.ConfirmationTTL,
		RateLimits:            newTestLimits(cfg),
		Version:               "test",
	}

	// The complete set must itself be accepted, or every case below would
	// pass for the wrong reason.
	_, err = server.New(cfg, complete)
	require.NoError(t, err)

	tests := []struct {
		name   string
		mutate func(*server.Dependencies)
	}{
		{"nil Keys", func(d *server.Dependencies) { d.Keys = nil }},
		{"nil OAuthServer", func(d *server.Dependencies) { d.OAuthServer = nil }},
		{"nil OAuthStore", func(d *server.Dependencies) { d.OAuthStore = nil }},
		{"nil APIClient", func(d *server.Dependencies) { d.APIClient = nil }},
		{"nil RedisClient", func(d *server.Dependencies) { d.RedisClient = nil }},
		{"zero APIDelegationTokenTTL", func(d *server.Dependencies) { d.APIDelegationTokenTTL = 0 }},
		{"zero ConfirmationTTL", func(d *server.Dependencies) { d.ConfirmationTTL = 0 }},
		{"empty Version", func(d *server.Dependencies) { d.Version = "" }},
		// Every rate-limit budget must be positive: ToolRateLimiter treats
		// a non-positive budget as "this tool is not rate limited at all",
		// so a forgotten or mis-wired budget would silently remove the
		// limit instead of failing.
		{"zero ReadPerMinute", func(d *server.Dependencies) { d.RateLimits.ReadPerMinute = 0 }},
		{"negative ReadPerMinute", func(d *server.Dependencies) { d.RateLimits.ReadPerMinute = -1 }},
		{"zero SendPerMinute", func(d *server.Dependencies) { d.RateLimits.SendPerMinute = 0 }},
		{"zero KeyCreatesPerHour", func(d *server.Dependencies) { d.RateLimits.KeyCreatesPerHour = 0 }},
		{"zero KeyRotationsPerHour", func(d *server.Dependencies) { d.RateLimits.KeyRotationsPerHour = 0 }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := complete
			test.mutate(&deps)
			_, err := server.New(cfg, deps)
			require.Error(t, err)
		})
	}
}

// --- Fix round 1: hardened assembly ---------------------------------------

// TestPublicMetadataRoutesAnswerCORSPreflight asserts a browser CORS
// preflight against every public discovery document is actually answered by
// the CORS wrapper -- not by the mux with a bare 405 -- and is permissive
// enough for the request headers an MCP/OAuth client sends.
func TestPublicMetadataRoutesAnswerCORSPreflight(t *testing.T) {
	h := newTestHarness(t)

	for _, path := range []string{
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-protected-resource/mcp",
		"/.well-known/oauth-authorization-server",
		"/.well-known/jwks.json",
	} {
		req, err := http.NewRequest(http.MethodOptions, h.httpServer.URL+path, nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "https://client.example")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		req.Header.Set("Access-Control-Request-Headers", "content-type, mcp-protocol-version")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equalf(t, http.StatusNoContent, resp.StatusCode, "OPTIONS %s", path)
		require.Equalf(t, "*", resp.Header.Get("Access-Control-Allow-Origin"), "OPTIONS %s", path)
		require.Emptyf(t, resp.Header.Get("Access-Control-Allow-Credentials"), "OPTIONS %s", path)

		allowedMethods := resp.Header.Get("Access-Control-Allow-Methods")
		require.Containsf(t, allowedMethods, http.MethodGet, "OPTIONS %s", path)
		require.Containsf(t, allowedMethods, http.MethodOptions, "OPTIONS %s", path)

		allowedHeaders := strings.ToLower(resp.Header.Get("Access-Control-Allow-Headers"))
		require.Containsf(t, allowedHeaders, "content-type", "OPTIONS %s", path)
		require.Containsf(t, allowedHeaders, "mcp-protocol-version", "OPTIONS %s", path)
	}
}

// TestPublicMetadataRoutesStillServeGETAndHEAD asserts adding the preflight
// path did not change the documents themselves: GET still serves the JSON,
// and HEAD still succeeds with the same headers and no body.
func TestPublicMetadataRoutesStillServeGETAndHEAD(t *testing.T) {
	h := newTestHarness(t)

	for _, path := range []string{
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-protected-resource/mcp",
		"/.well-known/oauth-authorization-server",
		"/.well-known/jwks.json",
	} {
		resp, err := http.Get(h.httpServer.URL + path)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equalf(t, http.StatusOK, resp.StatusCode, "GET %s", path)
		require.Equalf(t, "application/json", resp.Header.Get("Content-Type"), "GET %s", path)
		require.Equalf(t, "*", resp.Header.Get("Access-Control-Allow-Origin"), "GET %s", path)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NotEmptyf(t, body, "GET %s", path)

		headResp, err := http.Head(h.httpServer.URL + path)
		require.NoError(t, err)
		defer headResp.Body.Close()
		require.Equalf(t, http.StatusOK, headResp.StatusCode, "HEAD %s", path)
		require.Equalf(t, "*", headResp.Header.Get("Access-Control-Allow-Origin"), "HEAD %s", path)
	}
}

// TestPublicMetadataRoutesRejectUnsupportedMethods asserts a write method
// against a read-only document is refused with an Allow header rather than
// reaching the document handler.
func TestPublicMetadataRoutesRejectUnsupportedMethods(t *testing.T) {
	h := newTestHarness(t)

	req, err := http.NewRequest(http.MethodPost, h.httpServer.URL+"/.well-known/oauth-protected-resource", strings.NewReader("{}"))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Allow"), http.MethodGet)
}

// TestProtectedResourceMetadataIsServedAtBothRFC9728Paths asserts the
// path-suffixed alias RFC 9728 prescribes for a resource with a path
// component ("/.well-known/oauth-protected-resource/mcp") serves the exact
// same document as the root well-known path.
func TestProtectedResourceMetadataIsServedAtBothRFC9728Paths(t *testing.T) {
	h := newTestHarness(t)

	read := func(path string) map[string]any {
		resp, err := http.Get(h.httpServer.URL + path)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equalf(t, http.StatusOK, resp.StatusCode, "GET %s", path)

		var document map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&document))
		return document
	}

	root := read("/.well-known/oauth-protected-resource")
	suffixed := read("/.well-known/oauth-protected-resource/mcp")

	require.Equal(t, testIssuer+"/mcp", root["resource"])
	require.Equal(t, root, suffixed)
}

// TestHSTSIsSetOnlyInProduction asserts Strict-Transport-Security is sent
// in production (where every request is HTTPS and a downgraded request
// could expose a bearer token) and never outside it (where a cached HSTS
// entry for "localhost" would break unrelated local services).
func TestHSTSIsSetOnlyInProduction(t *testing.T) {
	local := newTestHarness(t)
	resp, err := http.Get(local.httpServer.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Empty(t, resp.Header.Get("Strict-Transport-Security"))

	production := newTestHarness(t, func(cfg *config.Config) {
		cfg.Environment = config.EnvironmentProduction
	})
	resp2, err := http.Get(production.httpServer.URL + "/healthz")
	require.NoError(t, err)
	defer resp2.Body.Close()

	hsts := resp2.Header.Get("Strict-Transport-Security")
	require.Contains(t, hsts, "max-age=")
	require.Contains(t, hsts, "includeSubDomains")
	require.NotContains(t, hsts, "max-age=0")
}

// TestInboundRequestIDIsBoundedAndValidated asserts a well-formed upstream
// correlation ID is preserved, while an oversized or malformed one (which
// would otherwise be echoed into a response header and every log line) is
// replaced with a freshly generated ID.
func TestInboundRequestIDIsBoundedAndValidated(t *testing.T) {
	h := newTestHarness(t)

	tests := []struct {
		name      string
		requestID string
		preserved bool
	}{
		{"uuid is preserved", "6f1b7f3e-6d2a-4f0f-9f39-0f6c4c6f39ab", true},
		{"trace-id shape is preserved", "trace-id:abc123/def-456_78=", true},
		{"empty is generated", "", false},
		{"oversized is replaced", strings.Repeat("a", 129), false},
		{"control character is replaced", "abc\tdef", false},
		{"whitespace is replaced", "abc def", false},
		{"non-ascii is replaced", "abc\u00e9", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, h.httpServer.URL+"/healthz", nil)
			require.NoError(t, err)
			if test.requestID != "" {
				// Set the raw value directly, bypassing the client's own
				// header validation: the point of the test is what this
				// service does with a hostile value, not what net/http
				// refuses to send.
				req.Header["X-Request-Id"] = []string{test.requestID}
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			echoed := resp.Header.Get("X-Request-Id")
			require.NotEmpty(t, echoed)

			if test.preserved {
				require.Equal(t, test.requestID, echoed)
				return
			}

			require.NotEqual(t, test.requestID, echoed)
			require.LessOrEqual(t, len(echoed), 128)
		})
	}
}

// panickingAPIClient is an httpsms.Client whose every call panics, standing
// in for a latent nil-dereference or index-out-of-range bug anywhere below
// the MCP dispatcher.
type panickingAPIClient struct{ stubAPIClient }

func (panickingAPIClient) ListPhones(context.Context, string, httpsms.ListPhonesParams) ([]httpsms.Phone, error) {
	panic("boom: simulated tool handler panic")
}

// TestToolHandlerPanicIsRecoveredAsJSONRPCInternalError asserts a panic
// inside a tool handler is converted into a JSON-RPC internal error rather
// than killing the process, and that the server keeps serving afterwards.
func TestToolHandlerPanicIsRecoveredAsJSONRPCInternalError(t *testing.T) {
	h := newTestHarnessWithAPIClient(t, panickingAPIClient{})
	token := h.mintToken(t, allScopes...)

	resp := postMCP(t, h, token, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_phones","arguments":{}}}`, nil)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var decoded struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.NotNilf(t, decoded.Error, "expected a JSON-RPC error, body: %s", body)
	require.Equal(t, -32603, decoded.Error.Code)
	// The client is told nothing about the panic itself.
	require.NotContains(t, strings.ToLower(decoded.Error.Message), "boom")
	require.NotContains(t, strings.ToLower(decoded.Error.Message), "panic")

	// The process survived and the server still serves other requests.
	resp2 := postMCP(t, h, token, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`, nil)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
}

// TestRotateUserAPIKeyConfirmationPromptIsNotRateLimited asserts a
// rotate_user_api_key call that presents no confirmation -- which can only
// mint a confirmation handle, never rotate anything -- does not consume the
// hourly rotation budget, while a call presenting a confirmation handle
// (which may execute) does.
func TestRotateUserAPIKeyConfirmationPromptIsNotRateLimited(t *testing.T) {
	h := newTestHarness(t, func(cfg *config.Config) {
		cfg.KeyRotationsPerHour = 1
	})
	token := h.mintToken(t, allScopes...)

	callError := func(t *testing.T, body string) string {
		t.Helper()

		resp := postMCP(t, h, token, body, nil)
		defer resp.Body.Close()

		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var decoded struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(raw, &decoded))
		if decoded.Error == nil {
			return ""
		}
		return strings.ToLower(decoded.Error.Message)
	}

	unconfirmed := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"rotate_user_api_key","arguments":{}}}`
	withHandle := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"rotate_user_api_key","arguments":{"confirmation_handle":"not-a-real-handle"}}}`

	// Several confirmation-only calls in a row must never exhaust a budget
	// of one rotation per hour.
	for i := 0; i < 4; i++ {
		require.NotContainsf(t, callError(t, unconfirmed), "rate limit", "confirmation-only call %d", i+1)
	}

	// The first call that presents a confirmation handle spends the single
	// available rotation...
	require.NotContains(t, callError(t, withHandle), "rate limit")
	// ... so the next one is rejected by the limiter.
	require.Contains(t, callError(t, withHandle), "rate limit")

	// A confirmation-only call is still refused once the budget is spent
	// only because it is charged -- it is not, so it still gets through.
	require.NotContains(t, callError(t, unconfirmed), "rate limit")
}

// TestRotateUserAPIKeyConfirmationPromptBudgetIsExhausted asserts the
// confirmation-prompt bucket -- distinct from the execution bucket -- has
// its own finite budget (KeyRotationsPerHour * 5) rather than being
// unlimited: a client that mints confirmation handles without ever
// confirming eventually gets rate limited too, and the structured error it
// receives names the confirmation bucket rather than the execution one.
func TestRotateUserAPIKeyConfirmationPromptBudgetIsExhausted(t *testing.T) {
	h := newTestHarness(t, func(cfg *config.Config) {
		cfg.KeyRotationsPerHour = 1
	})
	token := h.mintToken(t, allScopes...)

	unconfirmed := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"rotate_user_api_key","arguments":{}}}`

	// The confirmation-prompt budget is KeyRotationsPerHour * 5 = 5: the
	// first five confirmation-only calls must all succeed.
	for i := 0; i < 5; i++ {
		resp := postMCP(t, h, token, unconfirmed, nil)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)
		require.NotContainsf(t, strings.ToLower(string(body)), "rate limit", "confirmation-only call %d, body: %s", i+1, body)
	}

	// The sixth confirmation-only call in the same hour is rejected.
	resp := postMCP(t, h, token, unconfirmed, nil)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var decoded struct {
		Error *struct {
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.NotNilf(t, decoded.Error, "expected the sixth confirmation-only call to be rate limited, body: %s", body)
	require.Contains(t, strings.ToLower(decoded.Error.Message), "rate limit")

	var data struct {
		Tool string `json:"tool"`
	}
	require.NoError(t, json.Unmarshal(decoded.Error.Data, &data))
	// The error names the confirmation bucket, not the execution tool
	// name, so a client (and an operator reading logs) can tell the two
	// budgets apart.
	require.Equal(t, "rotate_user_api_key:confirm", data.Tool)
}

// TestRotateUserAPIKeyConfirmationAndExecutionBucketsAreIndependentOverHTTP
// asserts, through the fully assembled HTTP handler, that exhausting the
// confirmation-prompt bucket never spends any of the execution bucket's
// budget, and vice versa -- the same independent-buckets guarantee
// TestToolRateLimiterRotationConfirmationBucketIsIndependentAndDerived
// asserts at the unit level, confirmed here to hold end to end.
func TestRotateUserAPIKeyConfirmationAndExecutionBucketsAreIndependentOverHTTP(t *testing.T) {
	h := newTestHarness(t, func(cfg *config.Config) {
		cfg.KeyRotationsPerHour = 1
	})
	token := h.mintToken(t, allScopes...)

	unconfirmed := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"rotate_user_api_key","arguments":{}}}`
	withHandle := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"rotate_user_api_key","arguments":{"confirmation_handle":"not-a-real-handle"}}}`

	callBody := func(t *testing.T, body string) string {
		t.Helper()
		resp := postMCP(t, h, token, body, nil)
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return strings.ToLower(string(raw))
	}

	// Exhaust the confirmation-prompt bucket (budget 5).
	for i := 0; i < 5; i++ {
		require.NotContainsf(t, callBody(t, unconfirmed), "rate limit", "confirmation-only call %d", i+1)
	}
	require.Contains(t, callBody(t, unconfirmed), "rate limit")

	// The execution bucket (budget 1) is untouched: the single
	// confirmation-handle call still succeeds.
	require.NotContains(t, callBody(t, withHandle), "rate limit")
}

// TestNewRejectsMCPAudienceThatIsNotTheCanonicalEndpointURL asserts an
// MCP_AUDIENCE override that does not equal BaseURL + "/mcp" fails startup:
// discovery metadata always publishes the canonical value, so any other
// audience yields tokens no client could ever successfully present.
func TestNewRejectsMCPAudienceThatIsNotTheCanonicalEndpointURL(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := newTestConfig(t, mr)
	cfg.MCPAudience = testIssuer + "/not-mcp"

	keys := newTestKeys(t, cfg)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	store := oauth.NewRedisStore(redisClient)
	resolver := oauth.NewClientResolver(http.DefaultClient, store)

	// The OAuth server agrees with the (wrong) audience, so the only thing
	// that can catch this is the canonical-URL check itself.
	oauthServerConfig := oauth.ServerConfig{
		Issuer:               testIssuer,
		Resource:             cfg.MCPAudience,
		FirebaseAPIKey:       "test-firebase-api-key",
		FirebaseAuthDomain:   "httpsms-test.firebaseapp.com",
		AuthorizationCodeTTL: cfg.AuthorizationCodeTTL,
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
	}
	oauthServer, err := oauth.NewServer(store, resolver, keys, approvingVerifier{}, oauthServerConfig)
	require.NoError(t, err)

	_, err = server.New(cfg, server.Dependencies{
		Logger:                zerolog.Nop(),
		Keys:                  keys,
		OAuthServer:           oauthServer,
		OAuthServerConfig:     oauthServerConfig,
		OAuthStore:            store,
		APIClient:             stubAPIClient{},
		RedisClient:           redisClient,
		APIDelegationTokenTTL: cfg.APIDelegationTokenTTL,
		ConfirmationTTL:       cfg.ConfirmationTTL,
		RateLimits:            newTestLimits(cfg),
		Version:               "test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "MCPAudience")
	require.Contains(t, err.Error(), testIssuer+"/mcp")
}

// panickingRedisClient is a redis.UniversalClient whose embedded interface
// is nil: every command it does not override panics with a nil-pointer
// dereference. It stands in for a Redis client that panics inside the rate
// limiter, below the MCP dispatcher but above every tool handler.
type panickingRedisClient struct{ redis.UniversalClient }

// TestRateLimiterPanicIsRecoveredAsJSONRPCInternalError asserts a panic
// raised inside the rate limiter -- before any tool handler runs -- is
// recovered by the MCP receiving middleware and returned as a JSON-RPC
// internal error, rather than unwinding through the SDK's dispatch and
// killing the process.
func TestRateLimiterPanicIsRecoveredAsJSONRPCInternalError(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := newTestConfig(t, mr)
	keys := newTestKeys(t, cfg)

	stateRedis := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = stateRedis.Close() })

	store := oauth.NewRedisStore(stateRedis)
	resolver := oauth.NewClientResolver(http.DefaultClient, store)

	oauthServerConfig := oauth.ServerConfig{
		Issuer:               testIssuer,
		Resource:             cfg.MCPAudience,
		FirebaseAPIKey:       "test-firebase-api-key",
		FirebaseAuthDomain:   "httpsms-test.firebaseapp.com",
		AuthorizationCodeTTL: cfg.AuthorizationCodeTTL,
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
	}
	oauthServer, err := oauth.NewServer(store, resolver, keys, approvingVerifier{}, oauthServerConfig)
	require.NoError(t, err)

	handler, err := server.New(cfg, server.Dependencies{
		Logger:                zerolog.Nop(),
		Keys:                  keys,
		OAuthServer:           oauthServer,
		OAuthServerConfig:     oauthServerConfig,
		OAuthStore:            store,
		APIClient:             stubAPIClient{},
		RedisClient:           panickingRedisClient{},
		APIDelegationTokenTTL: cfg.APIDelegationTokenTTL,
		ConfirmationTTL:       cfg.ConfirmationTTL,
		RateLimits:            newTestLimits(cfg),
		Version:               "test",
	})
	require.NoError(t, err)

	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	h := &testHarness{httpServer: httpServer, keys: keys, cfg: cfg}
	token := h.mintToken(t, allScopes...)

	resp := postMCP(t, h, token, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_phones","arguments":{}}}`, nil)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var decoded struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.NotNilf(t, decoded.Error, "expected a JSON-RPC error, body: %s", body)
	require.Equal(t, -32603, decoded.Error.Code)

	// The process survived: a method that never touches the limiter still
	// works.
	resp2 := postMCP(t, h, token, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`, nil)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
}

// TestRotateUserAPIKeyConfirmationPromptFailsClosedOnRedisError asserts a
// confirmation-only rotate_user_api_key call -- which, before this fix,
// bypassed the rate limiter (and therefore Redis) entirely -- fails closed
// when the rate limiter's Redis client errors, exactly like every other
// rate-limited tool call. The call must never be let through just because
// it targets the confirmation-prompt bucket rather than the execution
// bucket.
func TestRotateUserAPIKeyConfirmationPromptFailsClosedOnRedisError(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := newTestConfig(t, mr)
	keys := newTestKeys(t, cfg)

	stateRedis := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = stateRedis.Close() })

	store := oauth.NewRedisStore(stateRedis)
	resolver := oauth.NewClientResolver(http.DefaultClient, store)

	oauthServerConfig := oauth.ServerConfig{
		Issuer:               testIssuer,
		Resource:             cfg.MCPAudience,
		FirebaseAPIKey:       "test-firebase-api-key",
		FirebaseAuthDomain:   "httpsms-test.firebaseapp.com",
		AuthorizationCodeTTL: cfg.AuthorizationCodeTTL,
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
	}
	oauthServer, err := oauth.NewServer(store, resolver, keys, approvingVerifier{}, oauthServerConfig)
	require.NoError(t, err)

	// A separate, standalone miniredis instance backs only the rate
	// limiter's Redis client (Dependencies.RedisClient), so it can be shut
	// down without breaking the OAuth store the bearer-auth path never
	// even needs. Closing it before the request forces every rate-limit
	// command -- including the confirmation-prompt bucket's -- to fail.
	limiterRedisServer := miniredis.RunT(t)
	limiterRedisClient := redis.NewClient(&redis.Options{Addr: limiterRedisServer.Addr()})
	t.Cleanup(func() { _ = limiterRedisClient.Close() })
	limiterRedisServer.Close()

	handler, err := server.New(cfg, server.Dependencies{
		Logger:                zerolog.Nop(),
		Keys:                  keys,
		OAuthServer:           oauthServer,
		OAuthServerConfig:     oauthServerConfig,
		OAuthStore:            store,
		APIClient:             stubAPIClient{},
		RedisClient:           limiterRedisClient,
		APIDelegationTokenTTL: cfg.APIDelegationTokenTTL,
		ConfirmationTTL:       cfg.ConfirmationTTL,
		RateLimits:            newTestLimits(cfg),
		Version:               "test",
	})
	require.NoError(t, err)

	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	h := &testHarness{httpServer: httpServer, keys: keys, cfg: cfg}
	token := h.mintToken(t, allScopes...)

	unconfirmed := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"rotate_user_api_key","arguments":{}}}`
	resp := postMCP(t, h, token, unconfirmed, nil)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var decoded struct {
		Result *struct {
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &decoded))
	// The call must be refused -- as a JSON-RPC error -- rather than
	// silently allowed through to the tool dispatcher just because Redis
	// is unreachable.
	require.Nilf(t, decoded.Result, "expected the call to be refused, not dispatched, body: %s", body)
	require.NotNilf(t, decoded.Error, "expected a JSON-RPC error when Redis is unreachable, body: %s", body)
	require.Contains(t, strings.ToLower(decoded.Error.Message), "rate limit")
}
