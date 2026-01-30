package middlewares

import (
	"fmt"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// PhoneAPIKeyAuth authenticates a user from the X-API-Key header
func PhoneAPIKeyAuth(logger telemetry.Logger, tracer telemetry.Tracer, repository repositories.PhoneAPIKeyRepository) fiber.Handler {
	logger = logger.WithService("middlewares.APIKeyAuth")

	return func(c *fiber.Ctx) error {
		ctx, span, ctxLogger := tracer.StartFromFiberCtxWithLogger(c, logger, "middlewares.APIKeyAuth")
		defer span.End()

		apiKey := c.Get(authHeaderAPIKey)
		if len(apiKey) == 0 || apiKey == "undefined" || !strings.HasPrefix(apiKey, "pk_") {
			span.AddEvent(fmt.Sprintf("the request header has no [%s] header for the phone key", authHeaderAPIKey))
			return c.Next()
		}

		authUser, err := repository.LoadAuthContext(ctx, apiKey)
		if err != nil {
			ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot load user with phone api key [%s]", apiKey)))
			return c.Next()
		}

		c.Locals(ContextKeyAuthUserID, authUser)
		return c.Next()
	}
}
