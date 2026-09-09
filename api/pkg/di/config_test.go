package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func envMap(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func TestMCPTokenVerifierConfigFromEnv_DisabledWhenAllEmpty(t *testing.T) {
	_, enabled, err := mcpTokenVerifierConfigFromEnv(envMap(map[string]string{}))

	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestMCPTokenVerifierConfigFromEnv_EnabledWhenAllSet(t *testing.T) {
	config, enabled, err := mcpTokenVerifierConfigFromEnv(envMap(map[string]string{
		"MCP_AUTH_ISSUER":   "https://mcp.httpsms.com",
		"MCP_AUTH_AUDIENCE": "https://api.httpsms.com",
		"MCP_AUTH_JWKS_URL": "https://mcp.httpsms.com/.well-known/jwks.json",
	}))

	require.NoError(t, err)
	require.True(t, enabled)
	assert.Equal(t, "https://mcp.httpsms.com", config.Issuer)
	assert.Equal(t, "https://api.httpsms.com", config.Audience)
	assert.Equal(t, "https://mcp.httpsms.com/.well-known/jwks.json", config.JWKSURL)
}

func TestMCPTokenVerifierConfigFromEnv_RejectsPartialConfiguration(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "missing issuer",
			env: map[string]string{
				"MCP_AUTH_AUDIENCE": "https://api.httpsms.com",
				"MCP_AUTH_JWKS_URL": "https://mcp.httpsms.com/.well-known/jwks.json",
			},
		},
		{
			name: "missing audience",
			env: map[string]string{
				"MCP_AUTH_ISSUER":   "https://mcp.httpsms.com",
				"MCP_AUTH_JWKS_URL": "https://mcp.httpsms.com/.well-known/jwks.json",
			},
		},
		{
			name: "missing jwks url",
			env: map[string]string{
				"MCP_AUTH_ISSUER":   "https://mcp.httpsms.com",
				"MCP_AUTH_AUDIENCE": "https://api.httpsms.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, enabled, err := mcpTokenVerifierConfigFromEnv(envMap(tt.env))

			require.Error(t, err)
			assert.False(t, enabled)
		})
	}
}
