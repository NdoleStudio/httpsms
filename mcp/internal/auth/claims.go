// Package auth loads the RSA signing key material for the hosted MCP
// service and mints/validates the JWTs the service issues: MCP access
// tokens (audience-bound to the MCP endpoint) and downstream API
// delegation tokens (audience-bound to api.httpsms.com, scoped to a
// single HTTP method and path).
package auth

import "github.com/golang-jwt/jwt/v5"

// Principal identifies the authenticated Firebase user a token is being
// minted for. It never carries a raw Firebase ID token, API key, or any
// other secret material.
type Principal struct {
	// UserID is the Firebase UID. It is always used as the JWT subject.
	UserID string

	// Email is the user's Firebase account email. It is included in minted
	// tokens for observability only; authorization decisions never depend
	// on it.
	Email string
}

// AccessClaims are the claims embedded in every JWT minted by this service,
// whether an MCP access token or a downstream API delegation token.
//
// MCP access tokens carry ClientID and Scopes but omit Method/Path (they
// authorize calling the MCP endpoint generally, not a single downstream API
// operation). API delegation tokens carry Method, Path, and Scopes bound to
// exactly one downstream API operation; ClientID is not applicable and is
// left empty.
//
// The JSON field names for Scopes, Method, and Path (`scopes`, `http_method`,
// `http_path`) are a wire contract with the httpSMS API's delegated MCP
// token verifier and must not change independently of it.
type AccessClaims struct {
	// ClientID is the OAuth client this MCP access token was issued to. It
	// is empty for API delegation tokens.
	ClientID string `json:"client_id,omitempty"`

	// Email is the Firebase account email of the token's subject.
	Email string `json:"email,omitempty"`

	// Scopes are the scopes granted to this token.
	Scopes []string `json:"scopes"`

	// Method is the HTTP method an API delegation token is bound to. It is
	// empty for MCP access tokens.
	Method string `json:"http_method,omitempty"`

	// Path is the HTTP request path an API delegation token is bound to. It
	// is empty for MCP access tokens.
	Path string `json:"http_path,omitempty"`

	jwt.RegisteredClaims
}
