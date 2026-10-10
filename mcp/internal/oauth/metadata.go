package oauth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Scopes lists every OAuth scope this service issues, in the fixed order
// presented in discovery metadata and the consent screen.
var Scopes = []string{
	"phones:read",
	"messages:read",
	"messages:send",
	"phone-api-keys:write",
	"user-api-key:rotate",
}

// protectedResourceMetadata is the RFC 9728 OAuth 2.0 Protected Resource
// Metadata document served for the MCP endpoint.
type protectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

// NewProtectedResourceMetadataHandler returns an http.HandlerFunc serving
// OAuth 2.0 Protected Resource Metadata (RFC 9728) for the MCP endpoint at
// baseURL+"/mcp". baseURL must not have a trailing slash requirement; any
// trailing slash is trimmed.
func NewProtectedResourceMetadataHandler(baseURL string) http.HandlerFunc {
	root := strings.TrimRight(baseURL, "/")

	return newMetadataHandler(protectedResourceMetadata{
		Resource:             root + "/mcp",
		AuthorizationServers: []string{root},
		ScopesSupported:      Scopes,
	})
}

// authorizationServerMetadata is the RFC 8414 OAuth 2.0 Authorization
// Server Metadata document, extended with the CIMD support flag consumed
// by clients implementing the Client ID Metadata Document mechanism.
type authorizationServerMetadata struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	RegistrationEndpoint                       string   `json:"registration_endpoint"`
	JWKSURI                                    string   `json:"jwks_uri"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	ScopesSupported                            []string `json:"scopes_supported"`
	AuthorizationResponseIssParameterSupported bool     `json:"authorization_response_iss_parameter_supported"`
	ClientIDMetadataDocumentSupported          bool     `json:"client_id_metadata_document_supported"`
}

// NewAuthorizationServerMetadataHandler returns an http.HandlerFunc serving
// OAuth 2.0 Authorization Server Metadata (RFC 8414) rooted at baseURL.
func NewAuthorizationServerMetadataHandler(baseURL string) http.HandlerFunc {
	root := strings.TrimRight(baseURL, "/")

	return newMetadataHandler(authorizationServerMetadata{
		Issuer:                        root,
		AuthorizationEndpoint:         root + "/oauth/authorize",
		TokenEndpoint:                 root + "/oauth/token",
		RegistrationEndpoint:          root + "/oauth/register",
		JWKSURI:                       root + "/.well-known/jwks.json",
		ResponseTypesSupported:        []string{"code"},
		GrantTypesSupported:           []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported: []string{"S256"},
		ScopesSupported:               Scopes,
		// Every authorization response this server issues -- success or
		// error, from the authorization endpoint or the Firebase
		// completion endpoint -- carries the RFC 9207 "iss" parameter, so
		// clients can and should enforce it.
		AuthorizationResponseIssParameterSupported: true,
		ClientIDMetadataDocumentSupported:          true,
	})
}

// newMetadataHandler marshals body once and returns an http.HandlerFunc
// that serves it as "application/json" on every request.
func newMetadataHandler(body any) http.HandlerFunc {
	data, err := json.Marshal(body)

	return func(w http.ResponseWriter, _ *http.Request) {
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}
