package middlewares

import (
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// APIKeyAuth authenticates a user from the X-API-Key header
func APIKeyAuth(logger telemetry.Logger, tracer telemetry.Tracer, userRepository repositories.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.StartFromFiberCtx(c, "middlewares.APIKeyAuth")
		defer span.End()

		ctxLogger := tracer.CtxLogger(logger, span)

		apiKey := c.Get(authHeaderAPIKey)
		if len(apiKey) > 0 {
			span.AddEvent(fmt.Sprintf("the request header has no [%s] api key", authHeaderAPIKey))
			return c.Next()
		}

		authUser, err := userRepository.LoadAuthUser(ctx, apiKey)
		if err != nil {
			ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot load user with api key [%s]", apiKey)))
			return c.Next()
		}

		c.Locals(ContextKeyAuthUserID, authUser)

		ctxLogger.Info(fmt.Sprintf("[%s] set successfully for user with ID [%s]", authUser, authUser.ID))

		return c.Next()
	}
}
