package server_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dcrRegistrationBody is a well-formed RFC 7591 Dynamic Client
// Registration request.
const dcrRegistrationBody = `{
	"client_name": "Assembled Server Client",
	"redirect_uris": ["https://client.example/callback"],
	"grant_types": ["authorization_code", "refresh_token"],
	"response_types": ["code"],
	"token_endpoint_auth_method": "none"
}`

// postRegistration posts body to the assembled server's /oauth/register
// endpoint with the given extra headers.
func postRegistration(t *testing.T, harness *testHarness, body string, headers map[string]string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, harness.httpServer.URL+"/oauth/register", strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		request.Header.Set(name, value)
	}

	response, err := harness.httpServer.Client().Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	return response
}

// TestAssembledServerRegistrationIsRateLimited asserts the assembled
// server actually wires a registration rate limiter in front of
// POST /oauth/register: registration is the only unauthenticated write in
// the service, and an unlimited one lets any caller fill Redis with
// 24-hour client records.
func TestAssembledServerRegistrationIsRateLimited(t *testing.T) {
	harness := newTestHarness(t)

	var lastResponse *http.Response
	accepted := 0
	// The production budget is 60 per fixed one-minute window. The loop
	// runs well past two windows' worth so it still observes a refusal
	// even if it happens to straddle a window boundary.
	for range 130 {
		lastResponse = postRegistration(t, harness, dcrRegistrationBody, nil)
		if lastResponse.StatusCode == http.StatusCreated {
			accepted++
			continue
		}
		break
	}

	require.Equal(t, http.StatusTooManyRequests, lastResponse.StatusCode, "the registration endpoint must be rate limited")
	assert.Positive(t, accepted, "a well-formed registration must still be accepted while budget remains")
	assert.LessOrEqual(t, accepted, 120, "no more than one window's budget (plus at most one rollover) may be accepted")

	retryAfter, err := strconv.Atoi(lastResponse.Header.Get("Retry-After"))
	require.NoError(t, err, "a 429 must carry an integer Retry-After header")
	assert.GreaterOrEqual(t, retryAfter, 1)
	assert.LessOrEqual(t, retryAfter, 60)

	assert.Equal(t, "no-store", lastResponse.Header.Get("Cache-Control"))

	var body struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	require.NoError(t, json.NewDecoder(lastResponse.Body).Decode(&body))
	assert.Equal(t, "temporarily_unavailable", body.Error)
	assert.NotEmpty(t, body.ErrorDescription)
}

// TestAssembledServerRegistrationLimitIgnoresClientIPHeaders asserts the
// budget cannot be reset by rotating a spoofable client-IP header: the
// limit is global precisely because no trustworthy caller identity exists
// on an unauthenticated endpoint.
func TestAssembledServerRegistrationLimitIgnoresClientIPHeaders(t *testing.T) {
	harness := newTestHarness(t)

	spoofHeaders := []map[string]string{
		{"X-Forwarded-For": "203.0.113.1"},
		{"X-Real-IP": "203.0.113.2"},
		{"CF-Connecting-IP": "203.0.113.3"},
		{"Forwarded": "for=203.0.113.4"},
		{"True-Client-IP": "203.0.113.5"},
	}

	var lastResponse *http.Response
	for i := range 130 {
		lastResponse = postRegistration(t, harness, dcrRegistrationBody, spoofHeaders[i%len(spoofHeaders)])
		if lastResponse.StatusCode != http.StatusCreated {
			break
		}
	}

	require.Equal(t, http.StatusTooManyRequests, lastResponse.StatusCode,
		"rotating a spoofable client-IP header must never restore registration budget")
}

// TestAssembledServerRegistrationRejectsMalformedWithoutStoring asserts a
// malformed registration is refused by the assembled server and leaves no
// client record behind.
func TestAssembledServerRegistrationRejectsMalformedWithoutStoring(t *testing.T) {
	harness := newTestHarness(t)

	response := postRegistration(t, harness, `{"client_name":"no redirect uris"}`, nil)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)

	var body struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	assert.Equal(t, "invalid_client_metadata", body.Error)

	for _, key := range harness.redis.Keys() {
		assert.False(t, strings.HasPrefix(key, "httpsms:mcp:oauth:client:"),
			"a rejected registration must never persist a client record (found %q)", key)
	}
}

// TestAssembledServerRegistrationStillWorksForValidClients asserts adding
// the rate limit did not break DCR compatibility: a well-formed
// registration still returns 201 with a usable client_id.
func TestAssembledServerRegistrationStillWorksForValidClients(t *testing.T) {
	harness := newTestHarness(t)

	response := postRegistration(t, harness, dcrRegistrationBody, nil)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	var registered struct {
		ClientID                string   `json:"client_id"`
		ClientName              string   `json:"client_name"`
		RedirectURIs            []string `json:"redirect_uris"`
		GrantTypes              []string `json:"grant_types"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&registered))

	assert.NotEmpty(t, registered.ClientID)
	assert.Equal(t, "Assembled Server Client", registered.ClientName)
	assert.Equal(t, []string{"https://client.example/callback"}, registered.RedirectURIs)
	assert.ElementsMatch(t, []string{"authorization_code", "refresh_token"}, registered.GrantTypes)
	assert.Equal(t, "none", registered.TokenEndpointAuthMethod)
}
