package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// mcpDelegationAuthTestLogger is a minimal telemetry.Logger test double that records every
// message passed to Warn/Error so tests can assert no raw bearer token is ever logged.
type mcpDelegationAuthTestLogger struct {
	messages []string
}

func (logger *mcpDelegationAuthTestLogger) Error(err error) {
	logger.messages = append(logger.messages, err.Error())
}
func (logger *mcpDelegationAuthTestLogger) WithService(string) telemetry.Logger { return logger }
func (logger *mcpDelegationAuthTestLogger) WithString(string, string) telemetry.Logger {
	return logger
}

func (logger *mcpDelegationAuthTestLogger) WithSpan(trace.SpanContext) telemetry.Logger {
	return logger
}
func (logger *mcpDelegationAuthTestLogger) Trace(string) {}
func (logger *mcpDelegationAuthTestLogger) Info(string)  {}
func (logger *mcpDelegationAuthTestLogger) Warn(err error) {
	logger.messages = append(logger.messages, err.Error())
}
func (logger *mcpDelegationAuthTestLogger) Debug(string)                  {}
func (logger *mcpDelegationAuthTestLogger) Fatal(error)                   {}
func (logger *mcpDelegationAuthTestLogger) Printf(string, ...interface{}) {}

type mcpDelegationAuthVerifierStub struct {
	claims *auth.MCPClaims
	err    error
}

func (stub *mcpDelegationAuthVerifierStub) VerifyRequest(context.Context, string, string, string) (*auth.MCPClaims, error) {
	return stub.claims, stub.err
}

// mcpDelegationAuthUserRepositoryStub implements repositories.UserRepository with only Load
// wired up, which is all MCPDelegationAuth depends on.
type mcpDelegationAuthUserRepositoryStub struct {
	user *entities.User
	err  error
}

func (stub *mcpDelegationAuthUserRepositoryStub) Store(context.Context, *entities.User) error {
	return nil
}

func (stub *mcpDelegationAuthUserRepositoryStub) Update(context.Context, *entities.User) error {
	return nil
}

func (stub *mcpDelegationAuthUserRepositoryStub) LoadAuthContext(context.Context, string) (entities.AuthContext, error) {
	return entities.AuthContext{}, nil
}

func (stub *mcpDelegationAuthUserRepositoryStub) Load(context.Context, entities.UserID) (*entities.User, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.user, nil
}

func (stub *mcpDelegationAuthUserRepositoryStub) RotateAPIKey(context.Context, entities.UserID) (*entities.User, error) {
	return nil, nil
}

func (stub *mcpDelegationAuthUserRepositoryStub) LoadOrStore(context.Context, entities.AuthContext) (*entities.User, bool, error) {
	return nil, false, nil
}

func (stub *mcpDelegationAuthUserRepositoryStub) LoadBySubscriptionID(context.Context, string) (*entities.User, error) {
	return nil, nil
}

func (stub *mcpDelegationAuthUserRepositoryStub) LoadByEmail(context.Context, string) (*entities.User, error) {
	return nil, nil
}

func (stub *mcpDelegationAuthUserRepositoryStub) Delete(context.Context, *entities.User) error {
	return nil
}

func newMCPDelegationAuthTestApp(t *testing.T, logger *mcpDelegationAuthTestLogger, verifier MCPTokenVerifier, users *mcpDelegationAuthUserRepositoryStub) *fiber.App {
	t.Helper()

	tracer := telemetry.NewOtelLogger("test", logger)

	app := fiber.New()
	app.Use(MCPDelegationAuth(logger, tracer, verifier, users))
	app.Get("/v1/messages", func(c fiber.Ctx) error {
		authUser, ok := c.Locals(ContextKeyAuthUserID).(entities.AuthContext)
		if !ok || authUser.IsNoop() {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error"})
		}
		return c.JSON(fiber.Map{"id": authUser.ID, "email": authUser.Email})
	})

	return app
}

func TestMCPDelegationAuthSetsAuthContext(t *testing.T) {
	logger := &mcpDelegationAuthTestLogger{}
	verifier := &mcpDelegationAuthVerifierStub{
		claims: &auth.MCPClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "user-id"}},
	}
	users := &mcpDelegationAuthUserRepositoryStub{
		user: &entities.User{ID: "user-id", Email: "user@example.com"},
	}
	app := newMCPDelegationAuthTestApp(t, logger, verifier, users)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer super-secret-delegated-token")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMCPDelegationAuth_NonBearerRequestPassesThrough(t *testing.T) {
	logger := &mcpDelegationAuthTestLogger{}
	verifier := &mcpDelegationAuthVerifierStub{err: stacktrace.NewErrorWithCodef(auth.ErrCodeInvalidToken, "should never be called")}
	users := &mcpDelegationAuthUserRepositoryStub{}
	app := newMCPDelegationAuthTestApp(t, logger, verifier, users)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	// No Authorization header at all.

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMCPDelegationAuth_InvalidDelegatedTokenPassesThroughForBearerAuth(t *testing.T) {
	logger := &mcpDelegationAuthTestLogger{}
	verifier := &mcpDelegationAuthVerifierStub{err: stacktrace.NewErrorWithCodef(auth.ErrCodeInvalidToken, "invalid MCP delegated token")}
	users := &mcpDelegationAuthUserRepositoryStub{}
	app := newMCPDelegationAuthTestApp(t, logger, verifier, users)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer not-an-mcp-token-super-secret")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	for _, message := range logger.messages {
		assert.NotContains(t, message, "not-an-mcp-token-super-secret")
	}
}

func TestMCPDelegationAuth_InsufficientScopeReturnsForbidden(t *testing.T) {
	logger := &mcpDelegationAuthTestLogger{}
	verifier := &mcpDelegationAuthVerifierStub{err: stacktrace.NewErrorWithCodef(auth.ErrCodeInsufficientScope, "MCP delegated token has insufficient scope")}
	users := &mcpDelegationAuthUserRepositoryStub{}
	app := newMCPDelegationAuthTestApp(t, logger, verifier, users)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer scoped-secret-token")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	for _, message := range logger.messages {
		assert.NotContains(t, message, "scoped-secret-token")
	}
}

func TestMCPDelegationAuth_OperationDeniedReturnsForbidden(t *testing.T) {
	logger := &mcpDelegationAuthTestLogger{}
	verifier := &mcpDelegationAuthVerifierStub{err: stacktrace.NewErrorWithCodef(auth.ErrCodeOperationDenied, "MCP delegated token is not valid for this API operation")}
	users := &mcpDelegationAuthUserRepositoryStub{}
	app := newMCPDelegationAuthTestApp(t, logger, verifier, users)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer denied-secret-token")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestMCPDelegationAuth_UnknownUserPassesThrough(t *testing.T) {
	logger := &mcpDelegationAuthTestLogger{}
	verifier := &mcpDelegationAuthVerifierStub{
		claims: &auth.MCPClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "missing-user-id"}},
	}
	users := &mcpDelegationAuthUserRepositoryStub{err: stacktrace.NewErrorWithCodef(auth.ErrCodeInvalidToken, "user not found")}
	app := newMCPDelegationAuthTestApp(t, logger, verifier, users)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer valid-but-unknown-user-secret")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	for _, message := range logger.messages {
		assert.NotContains(t, message, "valid-but-unknown-user-secret")
	}
}
