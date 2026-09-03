package oauth

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newClientsTestStore starts an in-memory miniredis-backed Store for this
// file's Dynamic Client Registration tests.
func newClientsTestStore(t *testing.T) Store {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return NewRedisStore(client)
}

// newLocalizedHTTPClient returns an *http.Client whose Transport connects
// to server's real listener address regardless of the requested host, with
// TLS hostname verification disabled. This lets a test use a
// realistic-looking "https://client.example/..." client_id (satisfying the
// resolver's public-address SSRF check via a stubbed lookupIP) while the
// document bytes are actually served by a local httptest.Server.
func newLocalizedHTTPClient(t *testing.T, server *httptest.Server) *http.Client {
	t.Helper()

	addr := server.Listener.Addr().String()
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test-only: bypasses hostname check for a locally redirected dial
	}
	t.Cleanup(transport.CloseIdleConnections)

	return &http.Client{Transport: transport}
}

// publicLookupIP is a stub lookupIP reporting that any host resolves to a
// single public IP address, for tests whose client_id host is a fake
// domain served locally through newLocalizedHTTPClient.
func publicLookupIP(context.Context, string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("93.184.216.34")}, nil
}

func validTestClient(clientID string, redirectURIs []string) Client {
	return Client{
		ID:                      clientID,
		Name:                    "Test Client",
		RedirectURIs:            redirectURIs,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	}
}

func TestClientResolverRejectsPrivateMetadataTarget(t *testing.T) {
	resolver := NewClientResolver(http.DefaultClient, newClientsTestStore(t))

	_, err := resolver.Resolve(context.Background(), "https://127.0.0.1/client.json")
	require.ErrorIs(t, err, ErrUnsafeClientMetadataURL)
}

func TestClientResolverRejectsLoopbackIPv6MetadataTarget(t *testing.T) {
	resolver := NewClientResolver(http.DefaultClient, newClientsTestStore(t))

	_, err := resolver.Resolve(context.Background(), "https://[::1]/client.json")
	require.ErrorIs(t, err, ErrUnsafeClientMetadataURL)
}

func TestClientResolverRejectsNonHTTPSClientID(t *testing.T) {
	resolver := NewClientResolver(http.DefaultClient, newClientsTestStore(t))

	_, err := resolver.Resolve(context.Background(), "http://client.example/client.json")
	require.ErrorIs(t, err, ErrUnsafeClientMetadataURL)
}

func TestClientResolverRejectsPrivateDNSResult(t *testing.T) {
	resolver := NewClientResolver(http.DefaultClient, newClientsTestStore(t))
	resolver.lookupIP = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.0.0.5")}, nil
	}

	_, err := resolver.Resolve(context.Background(), "https://internal.example/client.json")
	require.ErrorIs(t, err, ErrUnsafeClientMetadataURL)
}

func TestClientResolverRejectsLinkLocalDNSResult(t *testing.T) {
	resolver := NewClientResolver(http.DefaultClient, newClientsTestStore(t))
	resolver.lookupIP = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("169.254.1.1")}, nil
	}

	_, err := resolver.Resolve(context.Background(), "https://link-local.example/client.json")
	require.ErrorIs(t, err, ErrUnsafeClientMetadataURL)
}

func TestClientResolverAcceptsValidClientMetadataDocument(t *testing.T) {
	const clientID = "https://client.example/client.json"

	var requests int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(validTestClient(clientID, []string{
			"https://client.example/callback",
			"http://127.0.0.1/callback",
		}))
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	client, err := resolver.Resolve(context.Background(), clientID)
	require.NoError(t, err)
	assert.Equal(t, clientID, client.ID)
	assert.Equal(t, "Test Client", client.Name)
	assert.ElementsMatch(t, []string{"authorization_code", "refresh_token"}, client.GrantTypes)

	// A second Resolve within the 15-minute cache window must not issue a
	// second HTTP request.
	_, err = resolver.Resolve(context.Background(), clientID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, atomic.LoadInt32(&requests))
}

func TestClientResolverRefetchesAfterCacheExpiry(t *testing.T) {
	const clientID = "https://client.example/client.json"

	var requests int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(validTestClient(clientID, []string{"https://client.example/callback"}))
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	_, err := resolver.Resolve(context.Background(), clientID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, atomic.LoadInt32(&requests))

	// Force the cached entry to have already expired instead of waiting
	// out the real 15-minute cache window.
	resolver.cacheMu.Lock()
	entry := resolver.cache[clientID]
	entry.expiresAt = time.Now().Add(-time.Second)
	resolver.cache[clientID] = entry
	resolver.cacheMu.Unlock()

	_, err = resolver.Resolve(context.Background(), clientID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, atomic.LoadInt32(&requests))
}

func TestClientResolverRejectsResponseOverSizeLimit(t *testing.T) {
	const clientID = "https://client.example/client.json"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bytes.Repeat([]byte("a"), maxClientMetadataBytes+1))
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	_, err := resolver.Resolve(context.Background(), clientID)
	require.ErrorIs(t, err, ErrClientMetadataTooLarge)
}

func TestClientResolverRejectsRedirect(t *testing.T) {
	const clientID = "https://client.example/client.json"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://attacker.example/client.json", http.StatusFound)
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	_, err := resolver.Resolve(context.Background(), clientID)
	require.ErrorIs(t, err, ErrClientMetadataRedirected)
}

func TestClientResolverRejectsClientIDMismatch(t *testing.T) {
	const clientID = "https://client.example/client.json"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(validTestClient("https://client.example/different.json", []string{"https://client.example/callback"}))
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	_, err := resolver.Resolve(context.Background(), clientID)
	require.ErrorIs(t, err, ErrClientIDMismatch)
}

func TestClientResolverRejectsUnsupportedGrantTypes(t *testing.T) {
	testCases := map[string][]string{
		"empty":                      {},
		"unsupported":                {"client_credentials"},
		"missing_authorization_code": {"refresh_token"},
	}

	for name, grantTypes := range testCases {
		t.Run(name, func(t *testing.T) {
			const clientID = "https://client.example/client.json"

			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				client := validTestClient(clientID, []string{"https://client.example/callback"})
				client.GrantTypes = grantTypes
				_ = json.NewEncoder(w).Encode(client)
			}))
			defer server.Close()

			resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
			resolver.lookupIP = publicLookupIP

			_, err := resolver.Resolve(context.Background(), clientID)
			require.ErrorIs(t, err, ErrUnsupportedGrantType)
		})
	}
}

func TestClientResolverRejectsUnsupportedResponseTypes(t *testing.T) {
	const clientID = "https://client.example/client.json"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		client := validTestClient(clientID, []string{"https://client.example/callback"})
		client.ResponseTypes = []string{"token"}
		_ = json.NewEncoder(w).Encode(client)
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	_, err := resolver.Resolve(context.Background(), clientID)
	require.ErrorIs(t, err, ErrUnsupportedResponseType)
}

func TestClientResolverRejectsUnsupportedAuthMethod(t *testing.T) {
	const clientID = "https://client.example/client.json"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		client := validTestClient(clientID, []string{"https://client.example/callback"})
		client.TokenEndpointAuthMethod = "client_secret_basic"
		_ = json.NewEncoder(w).Encode(client)
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	_, err := resolver.Resolve(context.Background(), clientID)
	require.ErrorIs(t, err, ErrUnsupportedAuthMethod)
}

func TestClientResolverRejectsNonLoopbackHTTPRedirectURI(t *testing.T) {
	const clientID = "https://client.example/client.json"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(validTestClient(clientID, []string{"http://attacker.example/callback"}))
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	_, err := resolver.Resolve(context.Background(), clientID)
	require.ErrorIs(t, err, ErrInvalidRedirectURI)
}

func TestClientResolverAllowsLoopbackHTTPRedirectURI(t *testing.T) {
	const clientID = "https://client.example/client.json"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(validTestClient(clientID, []string{
			"http://127.0.0.1:51820/callback",
			"http://localhost:51820/callback",
		}))
	}))
	defer server.Close()

	resolver := NewClientResolver(newLocalizedHTTPClient(t, server), newClientsTestStore(t))
	resolver.lookupIP = publicLookupIP

	client, err := resolver.Resolve(context.Background(), clientID)
	require.NoError(t, err)
	assert.Contains(t, client.RedirectURIs, "http://127.0.0.1:51820/callback")
	assert.Contains(t, client.RedirectURIs, "http://localhost:51820/callback")
}

func TestClientResolverResolvesDynamicallyRegisteredClient(t *testing.T) {
	store := newClientsTestStore(t)
	require.NoError(t, store.PutDynamicClient(context.Background(), Client{
		ID:                      "dcr-opaque-client-id",
		Name:                    "DCR Client",
		RedirectURIs:            []string{"https://client.example/callback"},
		GrantTypes:              []string{"authorization_code"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	}, dynamicClientTTL))

	resolver := NewClientResolver(http.DefaultClient, store)

	client, err := resolver.Resolve(context.Background(), "dcr-opaque-client-id")
	require.NoError(t, err)
	assert.Equal(t, "DCR Client", client.Name)
}

func TestClientResolverResolveUnknownDynamicClientNotFound(t *testing.T) {
	resolver := NewClientResolver(http.DefaultClient, newClientsTestStore(t))

	_, err := resolver.Resolve(context.Background(), "never-registered")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestRegistrationHandlerCreatesClientAndReturns201(t *testing.T) {
	store := newClientsTestStore(t)
	handler := NewRegistrationHandler(store)

	requestBody := `{
		"client_name": "Test Client",
		"redirect_uris": ["https://client.example/callback"],
		"grant_types": ["authorization_code", "refresh_token"],
		"response_types": ["code"],
		"token_endpoint_auth_method": "none"
	}`

	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(requestBody))
	rec := httptest.NewRecorder()
	handler(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var created Client
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Test Client", created.Name)
	assert.ElementsMatch(t, []string{"authorization_code", "refresh_token"}, created.GrantTypes)

	stored, err := store.GetDynamicClient(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created, stored)
}

func TestRegistrationHandlerStoresRecordFor24Hours(t *testing.T) {
	miniredisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniredisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	store := NewRedisStore(redisClient)

	handler := NewRegistrationHandler(store)
	requestBody := `{
		"client_name": "Test Client",
		"redirect_uris": ["https://client.example/callback"],
		"grant_types": ["authorization_code"],
		"response_types": ["code"],
		"token_endpoint_auth_method": "none"
	}`

	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(requestBody))
	rec := httptest.NewRecorder()
	handler(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	keys := miniredisServer.Keys()
	require.Len(t, keys, 1)
	assert.True(t, strings.HasPrefix(keys[0], "httpsms:mcp:oauth:client:"))

	ttl := miniredisServer.TTL(keys[0])
	assert.Greater(t, ttl, 23*time.Hour)
	assert.LessOrEqual(t, ttl, 24*time.Hour)
}

func TestRegistrationHandlerRejectsInvalidMetadata(t *testing.T) {
	handler := NewRegistrationHandler(newClientsTestStore(t))

	requestBody := `{"client_name": "Missing Redirect URIs"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(requestBody))
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var body registrationError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "invalid_client_metadata", body.Error)
}

func TestRegistrationHandlerRejectsMalformedJSON(t *testing.T) {
	handler := NewRegistrationHandler(newClientsTestStore(t))

	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegistrationHandlerRejectsNonPOST(t *testing.T) {
	handler := NewRegistrationHandler(newClientsTestStore(t))

	req := httptest.NewRequest(http.MethodGet, "/oauth/register", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestIsPublicIPRejectsNonPublicRanges(t *testing.T) {
	testCases := []string{
		"127.0.0.1",
		"::1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"169.254.1.1",
		"224.0.0.1",
		"0.0.0.0",
		"100.64.0.1",
	}

	for _, raw := range testCases {
		t.Run(raw, func(t *testing.T) {
			assert.False(t, isPublicIP(net.ParseIP(raw)), "%s must not be treated as public", raw)
		})
	}
}

func TestIsPublicIPAcceptsPublicAddress(t *testing.T) {
	assert.True(t, isPublicIP(net.ParseIP("93.184.216.34")))
}
