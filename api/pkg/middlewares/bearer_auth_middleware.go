package middlewares

import (
	"context"
	"fmt"
	"strings"

	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/gofiber/fiber/v3"
)

// BearerAuth authenticates a user based on the bearer token
func BearerAuth(logger telemetry.Logger, tracer telemetry.Tracer, authClient *auth.Client) fiber.Handler {
	logger = logger.WithService("middlewares.BearerAuth")
	return func(c fiber.Ctx) error {
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
			ctxLogger.Warn(tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "invalid firebase id token [%s]", authToken)))
			return c.Next()
		}

		span.AddEvent(fmt.Sprintf("[%s] token is valid", bearerScheme))

		authUser := entities.AuthContext{
			Email: token.Claims["email"].(string),
			ID:    entities.UserID(token.Claims["user_id"].(string)),
		}

		c.Locals(ContextKeyAuthUserID, authUser)
		return c.Next()
	}
}
