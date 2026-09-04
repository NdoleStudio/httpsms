package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPReadiness asserts the MCP service's liveness/readiness endpoints,
// which are what Docker Compose and Cloud Run both gate traffic on.
func TestMCPReadiness(t *testing.T) {
	for _, path := range []string{"/health", "/healthz"} {
		response, body := doMCPRequest(t, http.MethodGet, mcpBaseURL+path, "", nil)
		require.Equal(t, http.StatusOK, response.StatusCode, "%s: %s", path, body)
		assert.Equal(t, "ok", strings.TrimSpace(body))
		assert.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", response.Header.Get("X-Frame-Options"))
		assert.NotEmpty(t, response.Header.Get("X-Request-Id"))
	}
}

// TestMCPMetadataDiscovery asserts the RFC 9728 protected-resource document
// (on both the root and path-suffixed routes), the RFC 8414 authorization
// server document, and the JWKS document the httpSMS API verifies delegation
// tokens against.
func TestMCPMetadataDiscovery(t *testing.T) {
	t.Run("protected resource metadata", func(t *testing.T) {
		for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
			var document struct {
				Resource             string   `json:"resource"`
				AuthorizationServers []string `json:"authorization_servers"`
				ScopesSupported      []string `json:"scopes_supported"`
			}
			response := getJSON(t, mcpBaseURL+path, &document)

			require.Equal(t, http.StatusOK, response.StatusCode, path)
			assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
			assert.Equal(t, "*", response.Header.Get("Access-Control-Allow-Origin"))
			assert.Equal(t, mcpResource, document.Resource)
			assert.Equal(t, []string{mcpBaseURL}, document.AuthorizationServers)
			assert.Equal(t, mcpAllScopes, document.ScopesSupported)
		}
	})

	t.Run("authorization server metadata", func(t *testing.T) {
		var document struct {
			Issuer                            string   `json:"issuer"`
			AuthorizationEndpoint             string   `json:"authorization_endpoint"`
			TokenEndpoint                     string   `json:"token_endpoint"`
			RegistrationEndpoint              string   `json:"registration_endpoint"`
			JWKSURI                           string   `json:"jwks_uri"`
			ResponseTypesSupported            []string `json:"response_types_supported"`
			GrantTypesSupported               []string `json:"grant_types_supported"`
			CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
			ScopesSupported                   []string `json:"scopes_supported"`
			IssParameterSupported             bool     `json:"authorization_response_iss_parameter_supported"`
			ClientIDMetadataDocumentSupported bool     `json:"client_id_metadata_document_supported"`
		}
		response := getJSON(t, mcpBaseURL+"/.well-known/oauth-authorization-server", &document)

		require.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, mcpBaseURL, document.Issuer)
		assert.Equal(t, mcpBaseURL+"/oauth/authorize", document.AuthorizationEndpoint)
		assert.Equal(t, mcpBaseURL+"/oauth/token", document.TokenEndpoint)
		assert.Equal(t, mcpBaseURL+"/oauth/register", document.RegistrationEndpoint)
		assert.Equal(t, mcpBaseURL+"/.well-known/jwks.json", document.JWKSURI)
		assert.Equal(t, []string{"code"}, document.ResponseTypesSupported)
		assert.Equal(t, []string{"authorization_code", "refresh_token"}, document.GrantTypesSupported)
		assert.Equal(t, []string{"S256"}, document.CodeChallengeMethodsSupported)
		assert.Equal(t, mcpAllScopes, document.ScopesSupported)
		assert.True(t, document.IssParameterSupported, "RFC 9207 iss must be advertised")
		assert.True(t, document.ClientIDMetadataDocumentSupported, "CIMD must be advertised")
	})

	t.Run("jwks", func(t *testing.T) {
		var document struct {
			Keys []struct {
				Kty string `json:"kty"`
				Use string `json:"use"`
				Alg string `json:"alg"`
				Kid string `json:"kid"`
				N   string `json:"n"`
				E   string `json:"e"`
			} `json:"keys"`
		}
		response := getJSON(t, mcpBaseURL+"/.well-known/jwks.json", &document)

		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Len(t, document.Keys, 1)
		assert.Equal(t, "RSA", document.Keys[0].Kty)
		assert.Equal(t, "sig", document.Keys[0].Use)
		assert.Equal(t, "RS256", document.Keys[0].Alg)
		assert.Equal(t, mcpSigningKeyID, document.Keys[0].Kid)
		assert.NotEmpty(t, document.Keys[0].N)
		assert.NotEmpty(t, document.Keys[0].E)

		// The JWKS document must publish the public half only.
		_, rawBody := doMCPRequest(t, http.MethodGet, mcpBaseURL+"/.well-known/jwks.json", "", nil)
		assert.NotContains(t, rawBody, "\"d\"", "the JWKS must never carry a private exponent")
		assert.NotContains(t, rawBody, "PRIVATE KEY")
	})

	t.Run("metadata preflight", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodOptions, mcpBaseURL+"/.well-known/oauth-protected-resource", nil)
		require.NoError(t, err)

		response, err := noRedirectClient().Do(request)
		require.NoError(t, err)
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, http.StatusNoContent, response.StatusCode)
		assert.Equal(t, "*", response.Header.Get("Access-Control-Allow-Origin"))
		assert.Contains(t, response.Header.Get("Access-Control-Allow-Methods"), http.MethodGet)
	})
}

// TestMCPUnauthenticatedChallenge asserts that the MCP endpoint refuses every
// unauthenticated or invalidly authenticated request with a 401 carrying the
// RFC 9728 resource metadata pointer a client needs to start authorization.
func TestMCPUnauthenticatedChallenge(t *testing.T) {
	tests := map[string]string{
		"no token":      "",
		"garbage token": "not-a-jwt",
		"foreign token": signAPIDelegationToken(t, mcpTestUserID, []string{"phones:read"}, http.MethodGet, "/v1/phones"),
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			response, _ := callLegacyMCP(t, token, mcpProtocolLatest, "tools/list", map[string]any{})

			require.Equal(t, http.StatusUnauthorized, response.StatusCode)

			challenge := response.Header.Get("WWW-Authenticate")
			require.NotEmpty(t, challenge, "a 401 must carry a WWW-Authenticate challenge")
			assert.Contains(t, challenge, "Bearer")
			assert.Contains(t, challenge, "resource_metadata=")
			assert.Contains(t, challenge, mcpBaseURL+"/.well-known/oauth-protected-resource")
		})
	}
}

// TestMCPOAuthClientRegistration covers RFC 7591 Dynamic Client Registration
// and the CIMD (Client ID Metadata Document) client resolution path the
// authorization server advertises.
func TestMCPOAuthClientRegistration(t *testing.T) {
	t.Run("registers a public client", func(t *testing.T) {
		clientID, redirectURI := registerMCPOAuthClient(t)
		assert.NotEmpty(t, clientID)
		assert.True(t, strings.HasPrefix(redirectURI, "http://localhost:53682/callback/"))
	})

	t.Run("rejects invalid client metadata", func(t *testing.T) {
		cases := map[string]map[string]any{
			"non loopback http redirect": {
				"client_name":                "bad redirect",
				"redirect_uris":              []string{"http://example.com/callback"},
				"grant_types":                []string{"authorization_code"},
				"response_types":             []string{"code"},
				"token_endpoint_auth_method": "none",
			},
			"confidential client": {
				"client_name":                "confidential",
				"redirect_uris":              []string{"https://example.com/callback"},
				"grant_types":                []string{"authorization_code"},
				"response_types":             []string{"code"},
				"token_endpoint_auth_method": "client_secret_basic",
			},
			"missing redirect uris": {
				"client_name":                "no redirects",
				"grant_types":                []string{"authorization_code"},
				"response_types":             []string{"code"},
				"token_endpoint_auth_method": "none",
			},
		}

		for name, metadata := range cases {
			t.Run(name, func(t *testing.T) {
				body, err := json.Marshal(metadata)
				require.NoError(t, err)

				response, responseBody := doMCPRequest(t, http.MethodPost, mcpBaseURL+"/oauth/register", "application/json", strings.NewReader(string(body)))
				require.Equal(t, http.StatusBadRequest, response.StatusCode, responseBody)
				assert.Equal(t, "no-store", response.Header.Get("Cache-Control"))

				var failure oauthErrorResponse
				require.NoError(t, json.Unmarshal([]byte(responseBody), &failure))
				assert.Contains(t, []string{"invalid_client_metadata", "invalid_redirect_uri"}, failure.Error)
			})
		}
	})

	t.Run("refuses a CIMD client id that resolves to a private host", func(t *testing.T) {
		// A URL client_id is resolved as a Client ID Metadata Document. The
		// resolver must refuse to fetch one from a private or loopback
		// address, which is the SSRF guard the hosted service depends on.
		response, body := getAuthorizePage(t, url.Values{
			"client_id":             {"https://localhost/cimd.json"},
			"redirect_uri":          {"http://localhost:53682/callback"},
			"response_type":         {"code"},
			"state":                 {uuid.NewString()},
			"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
			"code_challenge_method": {"S256"},
			"resource":              {mcpResource},
			"scope":                 {"phones:read"},
		})

		require.Equal(t, http.StatusBadRequest, response.StatusCode, body)

		var failure oauthErrorResponse
		require.NoError(t, json.Unmarshal([]byte(body), &failure))
		assert.Equal(t, "invalid_client", failure.Error)
	})
}

// TestMCPOAuthAuthorizationCodeFlow covers the complete PKCE authorization
// code flow, through real Firebase ID token verification against the
// WireMock-served certificate, and the access token it yields.
func TestMCPOAuthAuthorizationCodeFlow(t *testing.T) {
	params := requestAuthorizationCode(t, mcpTestUserID, mcpTestUserEmail, mcpAllScopes)
	assert.Equal(t, mcpBaseURL, params.Issuer, "RFC 9207 iss must be echoed on the authorization response")

	response, body := redeemAuthorizationCode(t, params)
	require.Equal(t, http.StatusOK, response.StatusCode, body)
	assert.Equal(t, "no-store", response.Header.Get("Cache-Control"))

	var tokens tokenResponse
	require.NoError(t, json.Unmarshal([]byte(body), &tokens))
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Positive(t, tokens.ExpiresIn)
	assert.ElementsMatch(t, mcpAllScopes, strings.Fields(tokens.Scope))

	t.Run("the issued access token is accepted by the MCP endpoint", func(t *testing.T) {
		session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest)
		result, err := session.ListTools(context.Background(), nil)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Tools)
	})

	t.Run("an authorization code cannot be replayed", func(t *testing.T) {
		replayResponse, replayBody := redeemAuthorizationCode(t, params)
		require.Equal(t, http.StatusBadRequest, replayResponse.StatusCode, replayBody)

		var failure oauthErrorResponse
		require.NoError(t, json.Unmarshal([]byte(replayBody), &failure))
		assert.Equal(t, "invalid_grant", failure.Error)
	})
}

// TestMCPOAuthAuthorizationErrors covers every rejected authorization and
// token request: a mismatched resource, an unregistered redirect URI, an
// unverifiable identity token (wrong issuer, wrong audience, wrong signing
// key), and a token request whose client, redirect URI, or PKCE verifier does
// not match the code.
func TestMCPOAuthAuthorizationErrors(t *testing.T) {
	clientID, redirectURI := registerMCPOAuthClient(t)
	_, challenge := pkcePair(t)

	baseQuery := func() url.Values {
		return url.Values{
			"client_id":             {clientID},
			"redirect_uri":          {redirectURI},
			"response_type":         {"code"},
			"state":                 {uuid.NewString()},
			"code_challenge":        {challenge},
			"code_challenge_method": {"S256"},
			"resource":              {mcpResource},
			"scope":                 {"phones:read"},
		}
	}

	t.Run("unregistered redirect uri is refused directly", func(t *testing.T) {
		query := baseQuery()
		query.Set("redirect_uri", "http://localhost:53682/not-registered")

		response, body := getAuthorizePage(t, query)
		require.Equal(t, http.StatusBadRequest, response.StatusCode, body)

		var failure oauthErrorResponse
		require.NoError(t, json.Unmarshal([]byte(body), &failure))
		assert.Equal(t, "invalid_request", failure.Error)
	})

	t.Run("a wrong resource is refused through the redirect", func(t *testing.T) {
		query := baseQuery()
		query.Set("resource", "https://evil.example.com/mcp")

		response, body := getAuthorizePage(t, query)
		require.Equal(t, http.StatusFound, response.StatusCode, body)

		location, err := url.Parse(response.Header.Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "invalid_target", location.Query().Get("error"))
		assert.Equal(t, mcpBaseURL, location.Query().Get("iss"))
	})

	t.Run("a plain code challenge is refused", func(t *testing.T) {
		query := baseQuery()
		query.Set("code_challenge_method", "plain")

		response, _ := getAuthorizePage(t, query)
		require.Equal(t, http.StatusFound, response.StatusCode)

		location, err := url.Parse(response.Header.Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "invalid_request", location.Query().Get("error"))
	})

	t.Run("an unverifiable identity token is refused", func(t *testing.T) {
		cases := map[string]string{
			"wrong issuer":   signFirebaseTokenWithClaims(t, func(claims jwt.MapClaims) { claims["iss"] = "https://securetoken.google.com/some-other-project" }),
			"wrong audience": signFirebaseTokenWithClaims(t, func(claims jwt.MapClaims) { claims["aud"] = "some-other-project" }),
			"unknown key id": signFirebaseTokenWithKeyID(t, "unknown-key-id"),
			"expired":        signFirebaseTokenWithClaims(t, func(claims jwt.MapClaims) { claims["exp"] = time.Now().Add(-time.Hour).Unix() }),
		}

		for name, idToken := range cases {
			t.Run(name, func(t *testing.T) {
				transactionID := startAuthorization(t, clientID, redirectURI, uuid.NewString(), challenge, []string{"phones:read"})

				form := url.Values{"transaction_id": {transactionID}, "id_token": {idToken}}
				form["approved_scopes"] = []string{"phones:read"}

				response, body := doMCPRequest(
					t,
					http.MethodPost,
					mcpBaseURL+"/oauth/firebase/complete",
					"application/x-www-form-urlencoded",
					strings.NewReader(form.Encode()),
				)
				require.Equal(t, http.StatusUnauthorized, response.StatusCode, body)

				var failure oauthErrorResponse
				require.NoError(t, json.Unmarshal([]byte(body), &failure))
				assert.Equal(t, "access_denied", failure.Error)
			})
		}
	})

	t.Run("a consent transaction cannot be replayed", func(t *testing.T) {
		state := uuid.NewString()
		transactionID := startAuthorization(t, clientID, redirectURI, state, challenge, []string{"phones:read"})
		idToken := signFirebaseTestToken(t, mcpTestUserID, mcpTestUserEmail)

		location := completeFirebaseConsent(t, transactionID, idToken, []string{"phones:read"}, http.StatusFound)
		redirected, err := url.Parse(location)
		require.NoError(t, err)
		require.NotEmpty(t, redirected.Query().Get("code"))

		form := url.Values{"transaction_id": {transactionID}, "id_token": {idToken}}
		form["approved_scopes"] = []string{"phones:read"}

		response, body := doMCPRequest(
			t,
			http.MethodPost,
			mcpBaseURL+"/oauth/firebase/complete",
			"application/x-www-form-urlencoded",
			strings.NewReader(form.Encode()),
		)
		require.Equal(t, http.StatusBadRequest, response.StatusCode, body)
	})

	t.Run("token requests that do not match the code are refused", func(t *testing.T) {
		overrides := map[string]func(url.Values){
			"wrong code verifier": func(form url.Values) { form.Set("code_verifier", strings.Repeat("a", 43)) },
			"wrong client id":     func(form url.Values) { form.Set("client_id", uuid.NewString()) },
			"wrong redirect uri":  func(form url.Values) { form.Set("redirect_uri", "http://localhost:53682/other") },
			"wrong resource":      func(form url.Values) { form.Set("resource", "https://evil.example.com/mcp") },
		}

		for name, override := range overrides {
			t.Run(name, func(t *testing.T) {
				params := requestAuthorizationCode(t, mcpTestUserID, mcpTestUserEmail, []string{"phones:read"})

				response, body := redeemAuthorizationCode(t, params, override)
				require.Equal(t, http.StatusBadRequest, response.StatusCode, body)

				var failure oauthErrorResponse
				require.NoError(t, json.Unmarshal([]byte(body), &failure))
				assert.Equal(t, "invalid_grant", failure.Error)

				// The code was consumed before the mismatch was detected, so
				// a corrected retry must fail too.
				retryResponse, retryBody := redeemAuthorizationCode(t, params)
				assert.Equal(t, http.StatusBadRequest, retryResponse.StatusCode, retryBody)
			})
		}
	})

	t.Run("an approval of no scope is refused", func(t *testing.T) {
		state := uuid.NewString()
		transactionID := startAuthorization(t, clientID, redirectURI, state, challenge, []string{"phones:read"})

		location := completeFirebaseConsent(t, transactionID, signFirebaseTestToken(t, mcpTestUserID, mcpTestUserEmail), nil, http.StatusFound)
		redirected, err := url.Parse(location)
		require.NoError(t, err)
		assert.Equal(t, "access_denied", redirected.Query().Get("error"))
		assert.Empty(t, redirected.Query().Get("code"))
	})
}

// TestMCPOAuthRefreshRotation covers refresh-token rotation, replay of a
// consumed refresh token, scope narrowing, and scope escalation.
func TestMCPOAuthRefreshRotation(t *testing.T) {
	params := requestAuthorizationCode(t, mcpTestUserID, mcpTestUserEmail, []string{"phones:read", "messages:read"})

	response, body := redeemAuthorizationCode(t, params)
	require.Equal(t, http.StatusOK, response.StatusCode, body)

	var issued tokenResponse
	require.NoError(t, json.Unmarshal([]byte(body), &issued))

	t.Run("rotates the refresh token", func(t *testing.T) {
		refreshResponse, refreshBody := refreshTokens(t, issued.RefreshToken, params.ClientID)
		require.Equal(t, http.StatusOK, refreshResponse.StatusCode, refreshBody)

		var rotated tokenResponse
		require.NoError(t, json.Unmarshal([]byte(refreshBody), &rotated))
		assert.NotEqual(t, issued.RefreshToken, rotated.RefreshToken, "the refresh token must rotate")
		assert.NotEmpty(t, rotated.AccessToken)
		assert.ElementsMatch(t, []string{"phones:read", "messages:read"}, strings.Fields(rotated.Scope))

		// The rotated access token must work against the MCP endpoint.
		session := newMCPClient(t, rotated.AccessToken, mcpProtocolLatest)
		_, err := session.ListTools(context.Background(), nil)
		require.NoError(t, err)

		t.Run("a replayed refresh token is refused", func(t *testing.T) {
			replayResponse, replayBody := refreshTokens(t, issued.RefreshToken, params.ClientID)
			require.Equal(t, http.StatusBadRequest, replayResponse.StatusCode, replayBody)

			var failure oauthErrorResponse
			require.NoError(t, json.Unmarshal([]byte(replayBody), &failure))
			assert.Equal(t, "invalid_grant", failure.Error)
		})

		t.Run("scope may be narrowed but never widened", func(t *testing.T) {
			narrowedResponse, narrowedBody := refreshTokens(t, rotated.RefreshToken, params.ClientID, func(form url.Values) {
				form.Set("scope", "phones:read")
			})
			require.Equal(t, http.StatusOK, narrowedResponse.StatusCode, narrowedBody)

			var narrowed tokenResponse
			require.NoError(t, json.Unmarshal([]byte(narrowedBody), &narrowed))
			assert.Equal(t, "phones:read", narrowed.Scope)

			widenedResponse, widenedBody := refreshTokens(t, narrowed.RefreshToken, params.ClientID, func(form url.Values) {
				form.Set("scope", "phones:read messages:send")
			})
			require.Equal(t, http.StatusBadRequest, widenedResponse.StatusCode, widenedBody)

			var failure oauthErrorResponse
			require.NoError(t, json.Unmarshal([]byte(widenedBody), &failure))
			assert.Equal(t, "invalid_scope", failure.Error)
		})
	})

	t.Run("a refresh token is bound to its client", func(t *testing.T) {
		otherParams := requestAuthorizationCode(t, mcpTestUserID, mcpTestUserEmail, []string{"phones:read"})
		otherResponse, otherBody := redeemAuthorizationCode(t, otherParams)
		require.Equal(t, http.StatusOK, otherResponse.StatusCode, otherBody)

		var otherTokens tokenResponse
		require.NoError(t, json.Unmarshal([]byte(otherBody), &otherTokens))

		mismatchResponse, mismatchBody := refreshTokens(t, otherTokens.RefreshToken, params.ClientID)
		require.Equal(t, http.StatusBadRequest, mismatchResponse.StatusCode, mismatchBody)

		var failure oauthErrorResponse
		require.NoError(t, json.Unmarshal([]byte(mismatchBody), &failure))
		assert.Equal(t, "invalid_grant", failure.Error)
	})
}

// TestMCPProtocolNegotiation asserts the served protocol surface for both the
// current (2026-07-28) and previous (2025-11-25) protocol versions, and that
// the tool catalog is exactly the seven approved tools on both.
func TestMCPProtocolNegotiation(t *testing.T) {
	tokens := completeOAuthCodeFlow(t, mcpAllScopes)

	t.Run("2026-07-28 discovery and tool listing", func(t *testing.T) {
		session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest)

		initialized := session.InitializeResult()
		require.NotNil(t, initialized)
		assert.Equal(t, mcpProtocolLatest, initialized.ProtocolVersion)
		require.NotNil(t, initialized.ServerInfo)
		assert.Equal(t, "httpSMS", initialized.ServerInfo.Name)
		assert.NotEmpty(t, initialized.ServerInfo.Version)

		result, err := session.ListTools(context.Background(), nil)
		require.NoError(t, err)

		names := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		assert.ElementsMatch(t, mcpToolNames, names, "the served tool catalog must be exactly the approved seven tools")
	})

	t.Run("2025-11-25 initialize and tool call", func(t *testing.T) {
		response, initialize := callLegacyMCP(t, tokens.AccessToken, mcpProtocolPrevious, "initialize", map[string]any{
			"protocolVersion": mcpProtocolPrevious,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "httpsms-legacy-client", "version": "test"},
		})
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Nil(t, initialize.Error, "legacy initialize failed: %+v", initialize.Error)

		var initialized struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		}
		require.NoError(t, json.Unmarshal(initialize.Result, &initialized))
		assert.Equal(t, mcpProtocolPrevious, initialized.ProtocolVersion)
		assert.Equal(t, "httpSMS", initialized.ServerInfo.Name)

		listResponse, list := callLegacyMCP(t, tokens.AccessToken, mcpProtocolPrevious, "tools/list", map[string]any{})
		require.Equal(t, http.StatusOK, listResponse.StatusCode)
		require.Nil(t, list.Error, "legacy tools/list failed: %+v", list.Error)

		var listed struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		require.NoError(t, json.Unmarshal(list.Result, &listed))

		names := make([]string, 0, len(listed.Tools))
		for _, tool := range listed.Tools {
			names = append(names, tool.Name)
		}
		assert.ElementsMatch(t, mcpToolNames, names)

		callResponse, call := callLegacyMCP(t, tokens.AccessToken, mcpProtocolPrevious, "tools/call", map[string]any{
			"name":      "list_phones",
			"arguments": map[string]any{"limit": 5},
		})
		require.Equal(t, http.StatusOK, callResponse.StatusCode)
		require.Nil(t, call.Error, "legacy tools/call failed: %+v", call.Error)

		var called struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Count int `json:"count"`
			} `json:"structuredContent"`
		}
		require.NoError(t, json.Unmarshal(call.Result, &called))
		assert.False(t, called.IsError, "legacy tool call returned a tool error: %s", string(call.Result))
	})
}

// TestMCPToolsThroughRealStack drives every read tool, the send tool, and the
// incoming-message path against the real MCP service, the real httpSMS API,
// and the existing FCM emulator.
func TestMCPToolsThroughRealStack(t *testing.T) {
	ctx := context.Background()
	tokens := completeOAuthCodeFlow(t, mcpAllScopes)
	session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest)

	phone := setupPhoneForUser(ctx, t, mcpUserAPIKey(), 60)
	contact := randomPhoneNumber()

	t.Run("list_phones returns the registered phone", func(t *testing.T) {
		var output struct {
			Phones []struct {
				PhoneNumber string `json:"phone_number"`
				SIM         string `json:"sim"`
			} `json:"phones"`
			Count int `json:"count"`
		}
		decodeToolOutput(t, callMCPTool(t, session, "list_phones", map[string]any{"limit": 20}), &output)

		require.Equal(t, len(output.Phones), output.Count)
		numbers := make([]string, 0, len(output.Phones))
		for _, listed := range output.Phones {
			numbers = append(numbers, listed.PhoneNumber)
		}
		assert.Contains(t, numbers, phone.PhoneNumber)
	})

	t.Run("send_sms delivers through the FCM emulator", func(t *testing.T) {
		var output struct {
			Message struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				Owner  string `json:"owner"`
			} `json:"message"`
		}
		decodeToolOutput(t, callMCPTool(t, session, "send_sms", map[string]any{
			"from":       phone.PhoneNumber,
			"to":         contact,
			"content":    "Hello from the MCP integration suite",
			"request_id": uuid.NewString(),
		}), &output)

		require.NotEmpty(t, output.Message.ID)
		assert.Equal(t, phone.PhoneNumber, output.Message.Owner)

		waitForFCMPush(t, output.Message.ID, 30*time.Second)
		fireEvent(ctx, t, phone.PhoneAPIKey, output.Message.ID, "SENT")
		fireEvent(ctx, t, phone.PhoneAPIKey, output.Message.ID, "DELIVERED")
		pollMessageStatusAs(ctx, t, mcpUserAPIKey(), output.Message.ID, "delivered", 30*time.Second)
	})

	t.Run("list_message_threads returns the conversation", func(t *testing.T) {
		var output struct {
			Threads []struct {
				Owner   string `json:"owner"`
				Contact string `json:"contact"`
			} `json:"threads"`
			Count int `json:"count"`
		}

		found := false
		for attempt := 0; attempt < 6 && !found; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Second)
			}
			decodeToolOutput(t, callMCPTool(t, session, "list_message_threads", map[string]any{
				"owner": phone.PhoneNumber,
				"limit": 20,
			}), &output)

			for _, thread := range output.Threads {
				if thread.Contact == contact {
					assert.Equal(t, phone.PhoneNumber, thread.Owner)
					found = true
				}
			}
		}
		assert.True(t, found, "thread %s -> %s was not returned by list_message_threads", phone.PhoneNumber, contact)
	})

	t.Run("list_thread_messages returns the sent message", func(t *testing.T) {
		var output struct {
			Messages []struct {
				Content string `json:"content"`
				Type    string `json:"type"`
			} `json:"messages"`
			Count int `json:"count"`
		}
		decodeToolOutput(t, callMCPTool(t, session, "list_thread_messages", map[string]any{
			"owner":   phone.PhoneNumber,
			"contact": contact,
			"limit":   20,
		}), &output)

		require.Equal(t, len(output.Messages), output.Count)
		contents := make([]string, 0, len(output.Messages))
		for _, message := range output.Messages {
			contents = append(contents, message.Content)
		}
		assert.Contains(t, contents, "Hello from the MCP integration suite")
	})

	t.Run("list_incoming_messages returns received SMS but never missed calls", func(t *testing.T) {
		incomingContent := "Inbound " + uuid.NewString()
		receiveSMSAs(ctx, t, phone, contact, incomingContent)
		reportMissedCallAs(ctx, t, phone, contact)

		type incomingOutput struct {
			Messages []struct {
				Content string `json:"content"`
				Type    string `json:"type"`
			} `json:"messages"`
			Count int `json:"count"`
		}

		var output incomingOutput
		found := false
		for attempt := 0; attempt < 6 && !found; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Second)
			}
			decodeToolOutput(t, callMCPTool(t, session, "list_incoming_messages", map[string]any{
				"owners": []string{phone.PhoneNumber},
				"limit":  50,
			}), &output)

			for _, message := range output.Messages {
				if message.Content == incomingContent {
					found = true
				}
			}
		}
		require.True(t, found, "the received SMS was not returned by list_incoming_messages")

		for _, message := range output.Messages {
			assert.Equal(t, "mobile-originated", message.Type, "list_incoming_messages must only return mobile-originated messages")
			assert.NotEqual(t, "Missed phone call", message.Content, "a missed call must never appear in incoming messages")
		}
	})
}

// TestMCPToolScopes asserts a token is only good for the tools its granted
// scopes cover.
func TestMCPToolScopes(t *testing.T) {
	tokens := completeOAuthCodeFlow(t, []string{"phones:read"})
	session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest)

	t.Run("a granted scope works", func(t *testing.T) {
		var output struct {
			Count int `json:"count"`
		}
		decodeToolOutput(t, callMCPTool(t, session, "list_phones", map[string]any{"limit": 5}), &output)
		assert.GreaterOrEqual(t, output.Count, 0)
	})

	refusals := map[string]struct {
		tool      string
		arguments map[string]any
		scope     string
	}{
		"messages:read": {tool: "list_message_threads", arguments: map[string]any{"owner": "+18005550199", "limit": 5}, scope: "messages:read"},
		"messages:send": {tool: "send_sms", arguments: map[string]any{"from": "+18005550199", "to": "+18005550100", "content": "nope"}, scope: "messages:send"},
		"phone-api-keys:write": {
			tool:      "create_phone_api_key",
			arguments: map[string]any{"name": "should not be created"},
			scope:     "phone-api-keys:write",
		},
		"user-api-key:rotate": {tool: "rotate_user_api_key", arguments: map[string]any{}, scope: "user-api-key:rotate"},
	}

	for name, refusal := range refusals {
		t.Run("a missing "+name+" scope is refused", func(t *testing.T) {
			result := callMCPTool(t, session, refusal.tool, refusal.arguments)
			require.True(t, result.IsError, "%s must be refused without the %s scope", refusal.tool, refusal.scope)
			assert.Contains(t, toolResultText(result), refusal.scope)
		})
	}
}

// TestMCPDelegationTokenBinding asserts the httpSMS API enforces the exact
// method, path, audience, issuer, and scope every MCP delegation token is
// minted for. It signs delegation tokens with the same key the MCP service
// uses, which is the only way to present the API with a token that is valid
// but wrongly bound.
func TestMCPDelegationTokenBinding(t *testing.T) {
	t.Run("a correctly bound token is accepted", func(t *testing.T) {
		token := signAPIDelegationToken(t, mcpTestUserID, []string{"phones:read"}, http.MethodGet, "/v1/phones")

		status, body := apiRequestWithBearer(t, http.MethodGet, "/v1/phones?skip=0&limit=10", token)
		require.Equal(t, http.StatusOK, status, body)
	})

	t.Run("a token bound to another path is refused", func(t *testing.T) {
		token := signAPIDelegationToken(t, mcpTestUserID, []string{"phones:read", "messages:read"}, http.MethodGet, "/v1/phones")

		status, body := apiRequestWithBearer(t, http.MethodGet, "/v1/messages/incoming?skip=0&limit=10", token)
		assert.Equal(t, http.StatusForbidden, status, body)
		assert.Contains(t, body, "MCP token cannot access this API operation")
	})

	t.Run("a token bound to a route outside the MCP catalog is refused", func(t *testing.T) {
		token := signAPIDelegationToken(t, mcpTestUserID, []string{"messages:read"}, http.MethodGet, "/v1/messages/search")

		status, body := apiRequestWithBearer(t, http.MethodGet, "/v1/messages/search?skip=0&limit=10", token)
		assert.Equal(t, http.StatusForbidden, status, body)
	})

	t.Run("a token missing the operation's scope is refused", func(t *testing.T) {
		token := signAPIDelegationToken(t, mcpTestUserID, []string{"messages:read"}, http.MethodGet, "/v1/phones")

		status, body := apiRequestWithBearer(t, http.MethodGet, "/v1/phones?skip=0&limit=10", token)
		assert.Equal(t, http.StatusForbidden, status, body)
	})

	t.Run("a wrong audience or issuer is not authenticated at all", func(t *testing.T) {
		cases := map[string]func(jwt.MapClaims){
			"wrong audience": func(claims jwt.MapClaims) { claims["aud"] = "https://api.example.com" },
			"wrong issuer":   func(claims jwt.MapClaims) { claims["iss"] = "https://evil.example.com" },
			"expired":        func(claims jwt.MapClaims) { claims["exp"] = time.Now().Add(-time.Hour).Unix() },
		}

		for name, mutate := range cases {
			t.Run(name, func(t *testing.T) {
				token := signAPIDelegationToken(t, mcpTestUserID, []string{"phones:read"}, http.MethodGet, "/v1/phones", mutate)

				status, body := apiRequestWithBearer(t, http.MethodGet, "/v1/phones?skip=0&limit=10", token)
				assert.Equal(t, http.StatusUnauthorized, status, body)
			})
		}
	})
}

// TestMCPIncomingIsNotCaptchaProtected asserts the design decision behind
// list_incoming_messages: the incoming route is reachable with a delegated MCP
// token, while the CAPTCHA-protected search route stays protected.
func TestMCPIncomingIsNotCaptchaProtected(t *testing.T) {
	t.Run("the incoming route serves a delegated MCP token", func(t *testing.T) {
		token := signAPIDelegationToken(t, mcpTestUserID, []string{"messages:read"}, http.MethodGet, "/v1/messages/incoming")

		status, body := apiRequestWithBearer(t, http.MethodGet, "/v1/messages/incoming?skip=0&limit=10", token)
		require.Equal(t, http.StatusOK, status, body)
	})

	t.Run("the search route still requires a CAPTCHA token", func(t *testing.T) {
		status, body := apiRequestWithAPIKey(t, http.MethodGet, "/v1/messages/search?skip=0&limit=10", mcpUserAPIKey())
		require.Equal(t, http.StatusUnprocessableEntity, status, body)
		assert.Contains(t, body, "token")
	})
}

// TestMCPCreatePhoneAPIKey asserts create_phone_api_key mints a real,
// immediately usable phone API key and returns it exactly once, marked
// sensitive.
func TestMCPCreatePhoneAPIKey(t *testing.T) {
	ctx := context.Background()
	tokens := completeOAuthCodeFlow(t, []string{"phone-api-keys:write", "phones:read"})
	session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest)

	var output struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		APIKey    string `json:"api_key"`
		Sensitive bool   `json:"sensitive"`
	}
	result := callMCPTool(t, session, "create_phone_api_key", map[string]any{"name": "mcp-integration-" + uuid.NewString()})
	decodeToolOutput(t, result, &output)

	require.NotEmpty(t, output.ID)
	assert.True(t, output.Sensitive, "a freshly minted key must be marked sensitive")
	assert.True(t, strings.HasPrefix(output.APIKey, "pk_"), "phone API key %q must carry the pk_ prefix", firstChars(output.APIKey, 3))
	assert.Contains(t, toolResultText(result), "will not be shown again")

	// The minted key must actually authenticate a phone against the API.
	phoneNumber := randomPhoneNumber()
	fcmToken := "fcm-" + uuid.NewString()
	requestJSONAs(ctx, t, http.MethodPut, "/v1/phones", mcpUserAPIKey(), map[string]any{
		"phone_number":               phoneNumber,
		"fcm_token":                  fcmToken,
		"messages_per_minute":        60,
		"max_send_attempts":          2,
		"message_expiration_seconds": 600,
		"sim":                        "SIM1",
	}, http.StatusOK, nil)

	// Binding the FCM token with the freshly minted key is what associates
	// the key with this phone, exactly as the Android app does on setup.
	requestJSONAs(ctx, t, http.MethodPut, "/v1/phones/fcm-token", output.APIKey, map[string]any{
		"phone_number": phoneNumber,
		"fcm_token":    fcmToken,
		"sim":          "SIM1",
	}, http.StatusOK, nil)

	waitForPhoneAuthorization(ctx, t, output.APIKey, phoneNumber, 20*time.Second)

	t.Run("the secret never reaches the service logs", func(t *testing.T) {
		logs, ok := mcpContainerLogs(t)
		if !ok {
			t.Skip("the Docker CLI is unavailable")
		}
		assert.NotContains(t, logs, output.APIKey, "a minted phone API key must never be logged")
		assert.NotContains(t, logs, tokens.AccessToken, "an access token must never be logged")
		assert.NotContains(t, logs, tokens.RefreshToken, "a refresh token must never be logged")
	})
}

// TestMCPRateLimit asserts the per-user, per-tool budget is enforced before a
// tool executes, and that the rejection carries a retry hint. It runs as a
// dedicated user so it can never starve the tools the other tests call.
func TestMCPRateLimit(t *testing.T) {
	tokens := completeOAuthCodeFlowAs(t, mcpRateLimitUserID, mcpRateLimitUserEmail, []string{"phone-api-keys:write"})
	session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest)

	// KEY_CREATES_PER_HOUR is 3 in tests/docker-compose.yml, so the fourth
	// call within the hour must be rejected before it reaches the API.
	var rateLimitErr error
	for attempt := 1; attempt <= 4; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "create_phone_api_key",
			Arguments: map[string]any{"name": fmt.Sprintf("rate-limit-%d-%s", attempt, uuid.NewString())},
		})
		cancel()

		if err != nil {
			rateLimitErr = err
			assert.Equal(t, 4, attempt, "the budget of 3 creates per hour must not be exhausted early")
			break
		}
	}

	require.Error(t, rateLimitErr, "the fourth create_phone_api_key call must be rate limited")
	assert.Contains(t, rateLimitErr.Error(), "rate limit exceeded")

	var wireError *jsonrpc.Error
	require.True(t, errors.As(rateLimitErr, &wireError), "a rate-limit rejection must be a structured JSON-RPC error")
	assert.EqualValues(t, -32029, wireError.Code)

	var data struct {
		Tool              string `json:"tool"`
		RetryAfterSeconds int    `json:"retry_after_seconds"`
	}
	require.NoError(t, json.Unmarshal(wireError.Data, &data))
	assert.Equal(t, "create_phone_api_key", data.Tool)
	assert.Positive(t, data.RetryAfterSeconds)
}

// TestMCPRotateUserAPIKey asserts the destructive rotation path end to end:
// an unconfirmed call never rotates, a legacy confirmation handle completes
// the rotation exactly once, the previous primary key stops working, the
// replacement key works, and a redeemed handle can never be replayed.
//
// It runs last, and only ever against mcp-test-user-api-key, because it is the
// only test that invalidates a seeded API key.
func TestMCPRotateUserAPIKeyLegacyConfirmation(t *testing.T) {
	tokens := completeOAuthCodeFlow(t, []string{"user-api-key:rotate"})

	// MRTR is disabled so the confirmation prompt is returned to the test
	// verbatim, which is exactly what a legacy client that cannot complete an
	// elicitation sees.
	session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest, func(options *mcp.ClientOptions) {
		options.MultiRoundTrip = &mcp.MultiRoundTripOptions{Disabled: true}
	})

	previousAPIKey := mcpUserAPIKey()

	prompt := callMCPTool(t, session, "rotate_user_api_key", map[string]any{})
	require.False(t, prompt.IsError, "the first call must ask for confirmation, not fail: %s", toolResultText(prompt))
	require.NotEmpty(t, prompt.RequestState, "the first call must return a confirmation handle")
	require.Contains(t, prompt.InputRequests, "confirm_rotation")
	assert.Nil(t, prompt.StructuredContent, "an unconfirmed call must never rotate anything")

	status, body := apiRequestWithAPIKey(t, http.MethodGet, "/v1/users/me", previousAPIKey)
	require.Equal(t, http.StatusOK, status, "the primary API key must survive an unconfirmed call: %s", body)

	handle := prompt.RequestState

	var rotated struct {
		User struct {
			ID     string `json:"id"`
			APIKey string `json:"api_key"`
		} `json:"user"`
		Sensitive bool   `json:"sensitive"`
		Warning   string `json:"warning"`
	}
	confirmed := callMCPTool(t, session, "rotate_user_api_key", map[string]any{"confirmation_handle": handle})
	decodeToolOutput(t, confirmed, &rotated)

	require.Equal(t, mcpTestUserID, rotated.User.ID)
	require.NotEmpty(t, rotated.User.APIKey)
	assert.True(t, strings.HasPrefix(rotated.User.APIKey, "uk_"), "a rotated primary key must carry the uk_ prefix")
	assert.NotEqual(t, previousAPIKey, rotated.User.APIKey)
	assert.True(t, rotated.Sensitive)
	assert.Contains(t, rotated.Warning, "invalidated")

	mcpUserAPIKeyValue = rotated.User.APIKey

	status, body = apiRequestWithAPIKey(t, http.MethodGet, "/v1/users/me", previousAPIKey)
	assert.Equal(t, http.StatusUnauthorized, status, "the previous primary API key must stop working: %s", body)

	status, body = apiRequestWithAPIKey(t, http.MethodGet, "/v1/users/me", rotated.User.APIKey)
	require.Equal(t, http.StatusOK, status, "the replacement primary API key must work: %s", body)

	t.Run("a redeemed confirmation handle cannot be replayed", func(t *testing.T) {
		replay := callMCPTool(t, session, "rotate_user_api_key", map[string]any{"confirmation_handle": handle})
		require.True(t, replay.IsError, "a redeemed confirmation handle must be refused")
		assert.Contains(t, toolResultText(replay), "confirmation")

		status, body := apiRequestWithAPIKey(t, http.MethodGet, "/v1/users/me", mcpUserAPIKey())
		require.Equal(t, http.StatusOK, status, "a refused replay must never rotate anything: %s", body)
	})

	t.Run("an unknown confirmation handle is refused", func(t *testing.T) {
		unknown := callMCPTool(t, session, "rotate_user_api_key", map[string]any{"confirmation_handle": uuid.NewString()})
		require.True(t, unknown.IsError)
		assert.Contains(t, toolResultText(unknown), "confirmation")
	})
}

// TestMCPRotateUserAPIKeyMRTRConfirmation asserts the modern multi-round-trip
// confirmation path: an MRTR-capable client fulfills the elicitation and the
// rotation completes in a single CallTool, while a declined elicitation never
// rotates anything.
func TestMCPRotateUserAPIKeyMRTRConfirmation(t *testing.T) {
	tokens := completeOAuthCodeFlow(t, []string{"user-api-key:rotate"})

	t.Run("a declined elicitation never rotates", func(t *testing.T) {
		before := mcpUserAPIKey()
		session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest, func(options *mcp.ClientOptions) {
			options.ElicitationHandler = func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
				return &mcp.ElicitResult{Action: "decline"}, nil
			}
		})

		result := callMCPTool(t, session, "rotate_user_api_key", map[string]any{})
		require.True(t, result.IsError, "a declined confirmation must refuse the rotation")
		assert.Contains(t, strings.ToLower(toolResultText(result)), "confirm")

		status, body := apiRequestWithAPIKey(t, http.MethodGet, "/v1/users/me", before)
		require.Equal(t, http.StatusOK, status, "a declined confirmation must leave the primary API key intact: %s", body)
	})

	t.Run("an accepted elicitation rotates exactly once", func(t *testing.T) {
		before := mcpUserAPIKey()
		session := newMCPClient(t, tokens.AccessToken, mcpProtocolLatest, func(options *mcp.ClientOptions) {
			options.ElicitationHandler = func(_ context.Context, request *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
				assert.Contains(t, request.Params.Message, "invalidates")
				return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirmed": true}}, nil
			}
		})

		var rotated struct {
			User struct {
				ID     string `json:"id"`
				APIKey string `json:"api_key"`
			} `json:"user"`
			Sensitive bool `json:"sensitive"`
		}
		decodeToolOutput(t, callMCPTool(t, session, "rotate_user_api_key", map[string]any{}), &rotated)

		require.Equal(t, mcpTestUserID, rotated.User.ID)
		require.True(t, strings.HasPrefix(rotated.User.APIKey, "uk_"))
		assert.NotEqual(t, before, rotated.User.APIKey)
		assert.True(t, rotated.Sensitive)

		mcpUserAPIKeyValue = rotated.User.APIKey

		status, _ := apiRequestWithAPIKey(t, http.MethodGet, "/v1/users/me", before)
		assert.Equal(t, http.StatusUnauthorized, status, "the previous primary API key must stop working")

		status, body := apiRequestWithAPIKey(t, http.MethodGet, "/v1/users/me", rotated.User.APIKey)
		require.Equal(t, http.StatusOK, status, "the replacement primary API key must work: %s", body)
	})

	t.Run("the rotated secret never reaches the service logs", func(t *testing.T) {
		logs, ok := mcpContainerLogs(t)
		if !ok {
			t.Skip("the Docker CLI is unavailable")
		}
		assert.NotContains(t, logs, mcpUserAPIKey(), "a rotated primary API key must never be logged")
	})
}

// firstChars returns at most n characters of value, for error messages that
// must never echo a whole secret.
func firstChars(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[:n]
}
