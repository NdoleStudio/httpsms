// Package config loads and validates the httpSMS MCP service's runtime
// configuration from environment variables. Load returns a single error
// naming every missing or invalid setting so misconfiguration fails fast at
// startup instead of surfacing as a confusing runtime error later.
package config

import (
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment values recognized by Load. Production enforces HTTPS on every
// configured URL; any other value is treated as a local/test environment.
const (
	EnvironmentProduction = "production"
	defaultEnvironment    = "local"
)

// Default values used when their corresponding environment variable is unset.
const (
	defaultPort                  = "8080"
	defaultAccessTokenTTL        = 15 * time.Minute
	defaultAPIDelegationTokenTTL = 2 * time.Minute
	defaultAuthorizationCodeTTL  = 2 * time.Minute
	defaultRefreshTokenTTL       = 30 * 24 * time.Hour
	defaultConfirmationTTL       = 5 * time.Minute
	defaultHTTPTimeout           = 10 * time.Second
	defaultReadToolsPerMinute    = 120
	defaultSendToolsPerMinute    = 30
	defaultKeyCreatesPerHour     = 10
	defaultKeyRotationsPerHour   = 3
	defaultFirebaseCertsURL      = "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com"
)

// Config is the validated runtime configuration for the httpSMS MCP service.
type Config struct {
	// Environment is "production" or a local/development/test value. It
	// controls whether Load enforces HTTPS on every configured URL.
	Environment string

	// Port is the TCP port the HTTP server listens on (Cloud Run supplies
	// this through the PORT environment variable).
	Port string

	// BaseURL is this MCP service's own public base URL, e.g.
	// "https://mcp.httpsms.com". It is used as the issuer of every JWT this
	// service mints.
	BaseURL *url.URL

	// APIURL is the httpSMS API's base URL this service calls on behalf of
	// authenticated users, e.g. "https://api.httpsms.com".
	APIURL *url.URL

	// RedisURL is the connection string for the Redis instance backing
	// OAuth authorization/refresh-token state and confirmation handles.
	RedisURL string

	// FirebaseProjectID is the Firebase project used to verify user ID
	// tokens during the browser login step of the authorization flow.
	FirebaseProjectID string

	// FirebaseAPIKey is the Firebase Web API key used by the hosted login
	// page's client-side Firebase SDK.
	FirebaseAPIKey string

	// FirebaseAuthDomain is the Firebase Auth domain used by the hosted
	// login page's client-side Firebase SDK.
	FirebaseAuthDomain string

	// FirebaseCertsURL is the JWKS endpoint used to verify Firebase ID
	// token signatures.
	FirebaseCertsURL *url.URL

	// SigningPrivateKeyPEM is the PEM-encoded RSA private key this service
	// signs MCP access tokens and API delegation tokens with.
	SigningPrivateKeyPEM []byte

	// SigningKeyID is the `kid` this service signs with and publishes in
	// its JWKS document.
	SigningKeyID string

	// MCPAudience is the audience MCP access tokens are bound to, e.g.
	// "https://mcp.httpsms.com/mcp".
	MCPAudience string

	// APIAudience is the audience API delegation tokens are bound to. It
	// must match the httpSMS API's configured API_AUDIENCE.
	APIAudience string

	// AccessTokenTTL is how long a minted MCP access token is valid.
	AccessTokenTTL time.Duration

	// APIDelegationTokenTTL is how long a minted downstream API delegation
	// token is valid.
	APIDelegationTokenTTL time.Duration

	// AuthorizationCodeTTL is how long an issued OAuth authorization code
	// remains redeemable.
	AuthorizationCodeTTL time.Duration

	// RefreshTokenTTL is how long an issued OAuth refresh token remains
	// valid.
	RefreshTokenTTL time.Duration

	// ConfirmationTTL is how long a primary API-key-rotation confirmation
	// handle remains redeemable.
	ConfirmationTTL time.Duration

	// HTTPTimeout bounds every outbound HTTP call this service makes to the
	// httpSMS API or to OAuth client metadata documents.
	HTTPTimeout time.Duration

	// ReadToolsPerMinute is the per-user rate limit applied to read-only MCP
	// tools (list phones, threads, messages).
	ReadToolsPerMinute int

	// SendToolsPerMinute is the per-user rate limit applied to the send_sms
	// MCP tool.
	SendToolsPerMinute int

	// KeyCreatesPerHour is the per-user rate limit applied to the
	// create_phone_api_key MCP tool.
	KeyCreatesPerHour int

	// KeyRotationsPerHour is the per-user rate limit applied to the
	// rotate_user_api_key MCP tool.
	KeyRotationsPerHour int
}

// Load reads and validates the MCP service configuration from environment
// variables. It returns a single error naming every missing or invalid
// setting.
func Load() (Config, error) {
	var problems []string
	add := func(problem string) { problems = append(problems, problem) }

	environment := stringEnv("ENV", defaultEnvironment)
	production := environment == EnvironmentProduction

	cfg := Config{
		Environment: environment,
		Port:        stringEnv("PORT", defaultPort),
	}

	cfg.BaseURL = requiredURL("MCP_BASE_URL", production, add)
	cfg.APIURL = requiredURL("HTTPSMS_API_URL", production, add)

	cfg.RedisURL = requiredString("REDIS_URL", add)

	cfg.FirebaseProjectID = requiredString("FIREBASE_PROJECT_ID", add)
	cfg.FirebaseAPIKey = requiredString("FIREBASE_API_KEY", add)
	cfg.FirebaseAuthDomain = requiredString("FIREBASE_AUTH_DOMAIN", add)
	cfg.FirebaseCertsURL = optionalURL("FIREBASE_CERTS_URL", defaultFirebaseCertsURL, production, add)

	cfg.SigningPrivateKeyPEM = loadSigningPrivateKeyPEM(add)
	cfg.SigningKeyID = requiredString("MCP_SIGNING_KEY_ID", add)

	if cfg.BaseURL != nil {
		cfg.MCPAudience = stringEnv("MCP_AUDIENCE", strings.TrimRight(cfg.BaseURL.String(), "/")+"/mcp")
	} else {
		cfg.MCPAudience = os.Getenv("MCP_AUDIENCE")
	}
	if cfg.APIURL != nil {
		cfg.APIAudience = stringEnv("API_AUDIENCE", strings.TrimRight(cfg.APIURL.String(), "/"))
	} else {
		cfg.APIAudience = os.Getenv("API_AUDIENCE")
	}

	cfg.AccessTokenTTL = durationEnv("MCP_ACCESS_TOKEN_TTL", defaultAccessTokenTTL, add)
	cfg.APIDelegationTokenTTL = durationEnv("API_DELEGATION_TOKEN_TTL", defaultAPIDelegationTokenTTL, add)
	cfg.AuthorizationCodeTTL = durationEnv("AUTHORIZATION_CODE_TTL", defaultAuthorizationCodeTTL, add)
	cfg.RefreshTokenTTL = durationEnv("REFRESH_TOKEN_TTL", defaultRefreshTokenTTL, add)
	cfg.ConfirmationTTL = durationEnv("CONFIRMATION_TTL", defaultConfirmationTTL, add)
	cfg.HTTPTimeout = durationEnv("HTTP_TIMEOUT", defaultHTTPTimeout, add)

	cfg.ReadToolsPerMinute = intEnv("READ_TOOLS_PER_MINUTE", defaultReadToolsPerMinute, add)
	cfg.SendToolsPerMinute = intEnv("SEND_TOOLS_PER_MINUTE", defaultSendToolsPerMinute, add)
	cfg.KeyCreatesPerHour = intEnv("KEY_CREATES_PER_HOUR", defaultKeyCreatesPerHour, add)
	cfg.KeyRotationsPerHour = intEnv("KEY_ROTATIONS_PER_HOUR", defaultKeyRotationsPerHour, add)

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("config: invalid configuration: %s", strings.Join(problems, "; "))
	}

	return cfg, nil
}

// stringEnv returns the environment variable named key, or fallback when it
// is unset or empty.
func stringEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// requiredString returns the environment variable named key, recording a
// problem through add when it is unset or empty.
func requiredString(key string, add func(string)) string {
	value := os.Getenv(key)
	if value == "" {
		add(fmt.Sprintf("%s is required", key))
	}
	return value
}

// requiredURL parses the environment variable named key as an absolute URL,
// recording a problem through add when it is unset, invalid, or (in
// production) not HTTPS.
func requiredURL(key string, production bool, add func(string)) *url.URL {
	raw := os.Getenv(key)
	if raw == "" {
		add(fmt.Sprintf("%s is required", key))
		return nil
	}

	parsed, err := parseAbsoluteURL(key, raw, production)
	if err != nil {
		add(err.Error())
		return nil
	}
	return parsed
}

// optionalURL parses the environment variable named key as an absolute URL,
// falling back to fallback when it is unset, and recording a problem through
// add when the resulting value is invalid or (in production) not HTTPS.
func optionalURL(key string, fallback string, production bool, add func(string)) *url.URL {
	raw := stringEnv(key, fallback)
	parsed, err := parseAbsoluteURL(key, raw, production)
	if err != nil {
		add(err.Error())
		return nil
	}
	return parsed
}

// parseAbsoluteURL parses raw as an absolute http(s) URL and, when
// production is true, requires the "https" scheme.
func parseAbsoluteURL(key string, raw string, production bool) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%s must be an absolute URL, got %q", key, raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s must use http or https, got %q", key, raw)
	}
	if production && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s must use https in production, got %q", key, raw)
	}
	return parsed, nil
}

// loadSigningPrivateKeyPEM loads the RSA signing key material from either
// MCP_SIGNING_PRIVATE_KEY or MCP_SIGNING_PRIVATE_KEY_FILE, recording a
// problem through add when neither, both, or an unreadable/malformed value is
// configured.
func loadSigningPrivateKeyPEM(add func(string)) []byte {
	inline := os.Getenv("MCP_SIGNING_PRIVATE_KEY")
	file := os.Getenv("MCP_SIGNING_PRIVATE_KEY_FILE")

	switch {
	case inline != "" && file != "":
		add("only one of MCP_SIGNING_PRIVATE_KEY or MCP_SIGNING_PRIVATE_KEY_FILE may be set, not both")
		return nil
	case inline != "":
		return validatePEM("MCP_SIGNING_PRIVATE_KEY", []byte(inline), add)
	case file != "":
		contents, err := os.ReadFile(file)
		if err != nil {
			add(fmt.Sprintf("cannot read MCP_SIGNING_PRIVATE_KEY_FILE %q: %v", file, err))
			return nil
		}
		return validatePEM("MCP_SIGNING_PRIVATE_KEY_FILE", contents, add)
	default:
		add("one of MCP_SIGNING_PRIVATE_KEY or MCP_SIGNING_PRIVATE_KEY_FILE is required")
		return nil
	}
}

// validatePEM confirms keyPEM decodes as a PEM block. It does not parse the
// key's ASN.1 structure or enforce key type/size; that validation belongs to
// auth.NewKeySet, which is the single source of truth for what key material
// this service accepts.
func validatePEM(key string, keyPEM []byte, add func(string)) []byte {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		add(fmt.Sprintf("%s does not contain a PEM-encoded private key", key))
		return nil
	}
	return keyPEM
}

// durationEnv parses the environment variable named key as a time.Duration,
// falling back to fallback when unset and recording a problem through add
// when set but invalid or not positive.
func durationEnv(key string, fallback time.Duration, add func(string)) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		add(fmt.Sprintf("%s must be a valid duration, got %q", key, raw))
		return fallback
	}
	if value <= 0 {
		add(fmt.Sprintf("%s must be positive, got %q", key, raw))
		return fallback
	}
	return value
}

// intEnv parses the environment variable named key as a positive int,
// falling back to fallback when unset and recording a problem through add
// when set but invalid or not positive.
func intEnv(key string, fallback int, add func(string)) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		add(fmt.Sprintf("%s must be a valid integer, got %q", key, raw))
		return fallback
	}
	if value <= 0 {
		add(fmt.Sprintf("%s must be positive, got %q", key, raw))
		return fallback
	}
	return value
}
