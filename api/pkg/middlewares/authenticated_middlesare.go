package middlewares

import (
	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
)

const (
	authHeaderBearer = "Authorization"
	authHeaderAPIKey = "X-API-Key"
	bearerScheme     = "Bearer"
)

const (
	// ContextKeyAuthUserID is the context key used to store the ID of an authenticated user
	ContextKeyAuthUserID = "auth.user.id"
)

// Authenticated checks if the request is authenticated
func Authenticated(tracer telemetry.Tracer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, span := tracer.StartFromFiberCtx(c, "middlewares.Authenticated")
		defer span.End()

		if tokenUser, ok := c.Locals(ContextKeyAuthUserID).(entities.AuthUser); !ok || tokenUser.IsNoop() {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "You are not authorized to carry out this request.",
				"data":    "Make sure your API key is set in the [X-API-Key] header in the request",
			})
		}

		return c.Next()
	}
}
