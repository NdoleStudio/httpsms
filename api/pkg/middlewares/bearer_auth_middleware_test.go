package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBearerAuth_SkipsFirebaseVerificationWhenAlreadyAuthenticated proves BearerAuth short
// circuits as soon as a prior middleware (MCPDelegationAuth) has already populated
// ContextKeyAuthUserID. authClient is nil: if BearerAuth attempted Firebase verification here it
// would panic on the nil pointer, so reaching the downstream handler proves the short-circuit
// fired instead.
func TestBearerAuth_SkipsFirebaseVerificationWhenAlreadyAuthenticated(t *testing.T) {
	logger := &mcpDelegationAuthTestLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("mcp-user"), Email: "mcp-user@example.com"})
		return c.Next()
	})
	app.Use(BearerAuth(logger, tracer, nil))
	app.Get("/v1/messages", func(c fiber.Ctx) error {
		authUser, _ := c.Locals(ContextKeyAuthUserID).(entities.AuthContext)
		return c.JSON(fiber.Map{"id": authUser.ID})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer some-mcp-delegated-jwt")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestBearerAuth_ContinuesWhenNoBearerTokenIsPresent proves BearerAuth still passes requests
// through to c.Next() unchanged when there is no authentication context yet and no Authorization
// header, preserving existing behavior for normal callers.
func TestBearerAuth_ContinuesWhenNoBearerTokenIsPresent(t *testing.T) {
	logger := &mcpDelegationAuthTestLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)

	app := fiber.New()
	app.Use(BearerAuth(logger, tracer, nil))
	app.Get("/v1/messages", func(c fiber.Ctx) error {
		_, ok := c.Locals(ContextKeyAuthUserID).(entities.AuthContext)
		return c.JSON(fiber.Map{"authenticated": ok})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
