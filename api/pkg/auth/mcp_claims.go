package auth

import "github.com/golang-jwt/jwt/v5"

// MCPClaims are the claims embedded in a delegated MCP API JWT minted by the
// hosted MCP service on behalf of an authenticated user. The token is scoped
// to a single API operation: it is only valid for the exact HTTP method and
// path it was minted for, and only when it carries the scope that operation
// requires.
type MCPClaims struct {
	// Scopes are the downstream API scopes granted to this delegated token.
	Scopes []string `json:"scopes"`

	// Method is the HTTP method this delegated token is bound to.
	Method string `json:"http_method"`

	// Path is the HTTP request path this delegated token is bound to.
	Path string `json:"http_path"`

	jwt.RegisteredClaims
}
