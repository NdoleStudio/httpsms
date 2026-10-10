package di

import (
	"log"
	"os"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/auth"
	"github.com/NdoleStudio/stacktrace"
	"github.com/joho/godotenv"
)

// LoadEnv will read your .env file(s) and load them into ENV for this process.
func LoadEnv(filenames ...string) {
	err := godotenv.Load(filenames...)
	if err != nil {
		log.Fatalf("Fatal: cannot load .env file: %v", err)
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func splitCommaEnv(key, defaultValue string) []string {
	value := getEnvWithDefault(key, defaultValue)
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// mcpTokenVerifierConfigFromEnv resolves auth.MCPTokenVerifierConfig from MCP_AUTH_ISSUER,
// MCP_AUTH_AUDIENCE, and MCP_AUTH_JWKS_URL using getenv (os.Getenv in production).
//
// enabled is false, with a nil error, when all three variables are empty: delegated MCP
// authentication is optional and stays disabled until it is fully configured.
//
// An error is returned when only some of the three variables are set, since a partially
// configured delegated MCP issuer must never silently run with a missing issuer, audience, or
// JWKS URL.
func mcpTokenVerifierConfigFromEnv(getenv func(string) string) (config auth.MCPTokenVerifierConfig, enabled bool, err error) {
	issuer := getenv("MCP_AUTH_ISSUER")
	audience := getenv("MCP_AUTH_AUDIENCE")
	jwksURL := getenv("MCP_AUTH_JWKS_URL")

	if issuer == "" && audience == "" && jwksURL == "" {
		return auth.MCPTokenVerifierConfig{}, false, nil
	}

	if issuer == "" || audience == "" || jwksURL == "" {
		return auth.MCPTokenVerifierConfig{}, false, stacktrace.NewError(
			"MCP_AUTH_ISSUER, MCP_AUTH_AUDIENCE, and MCP_AUTH_JWKS_URL must all be set together to enable delegated MCP authentication",
		)
	}

	return auth.MCPTokenVerifierConfig{
		Issuer:   issuer,
		Audience: audience,
		JWKSURL:  jwksURL,
	}, true, nil
}
