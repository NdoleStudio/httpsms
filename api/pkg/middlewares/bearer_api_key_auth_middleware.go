package middlewares

import (
	"fmt"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// BearerAPIKeyAuth authenticates an API key using the Bearer header
func BearerAPIKeyAuth(logger telemetry.Logger, tracer telemetry.Tracer, userRepository repositories.UserRepository) fiber.Handler {
	logger = logger.WithService("middlewares.APIKeyAuth")

	return func(c *fiber.Ctx) error {
		ctx, span := tracer.StartFromFiberCtx(c, "middlewares.APIKeyAuth")
		defer span.End()

		ctxLogger := tracer.CtxLogger(logger, span)

		apiKey := strings.TrimSpace(strings.Replace(c.Get(authHeaderBearer), bearerScheme, "", 1))
		if len(apiKey) == 0 {
			span.AddEvent(fmt.Sprintf("the request header has no [%s] api key", authHeaderAPIKey))
			return c.Next()
		}

		authUser, err := userRepository.LoadAuthContext(ctx, apiKey)
		if err != nil {
			ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot load user with api key [%s] using header [%s]", apiKey, c.Get(authHeaderBearer))))
			return c.Next()
		}

		c.Locals(ContextKeyAuthUserID, authUser)

		ctxLogger.Info(fmt.Sprintf("[%T] set successfully for user with ID [%s]", authUser, authUser.ID))

		return c.Next()
	}
}
