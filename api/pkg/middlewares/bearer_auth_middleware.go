package middlewares

import (
	"context"
	"fmt"
	"strings"

	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// BearerAuth authenticates a user based on the bearer token
func BearerAuth(logger telemetry.Logger, tracer telemetry.Tracer, authClient *auth.Client) fiber.Handler {
	logger = logger.WithService("middlewares.BearerAuth")
	return func(c *fiber.Ctx) error {
		_, span := tracer.StartFromFiberCtx(c, "middlewares.BearerAuth")
		defer span.End()

		authToken := c.Get(authHeaderBearer)
		if !strings.HasPrefix(authToken, bearerScheme) {
			span.AddEvent(fmt.Sprintf("The request header has no [%s] token", bearerScheme))
			return c.Next()
		}

		if len(authToken) > len(bearerScheme)+1 {
			authToken = authToken[len(bearerScheme)+1:]
		}

		ctxLogger := tracer.CtxLogger(logger, span)

		token, err := authClient.VerifyIDToken(context.Background(), authToken)
		if err != nil {
			msg := fmt.Sprintf("invalid firebase id token [%s]", authToken)
			ctxLogger.Warn(tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
			return c.Next()
		}

		span.AddEvent(fmt.Sprintf("[%s] token is valid", bearerScheme))

		authUser := entities.AuthUser{
			Email: token.Claims["email"].(string),
			ID:    entities.UserID(token.Claims["user_id"].(string)),
		}

		c.Locals(ContextKeyAuthUserID, authUser)

		ctxLogger.Info(fmt.Sprintf("[%T] set successfully for user with ID [%s]", authUser, authUser.ID))
		return c.Next()
	}
}
