package middlewares

import (
	"context"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/gofiber/fiber/v3"
)

// MCPTokenVerifier verifies a delegated MCP API JWT is unexpired, issued by the trusted MCP
// issuer for the expected audience, and bound to the exact method, path, and scope of the
// current request. It is satisfied by *auth.MCPTokenVerifier.
type MCPTokenVerifier interface {
	VerifyRequest(ctx context.Context, raw string, method string, path string) (*auth.MCPClaims, error)
}

// MCPDelegationAuth authenticates a user from a delegated MCP API JWT minted by the hosted MCP
// service. It must be registered before BearerAuth: a cryptographically valid MCP token that is
// not bound to the requested operation is rejected with 403 directly, instead of falling through
// to Firebase ID token verification. Malformed or non-MCP bearer tokens continue to the next
// authentication middleware unchanged.
func MCPDelegationAuth(logger telemetry.Logger, tracer telemetry.Tracer, verifier MCPTokenVerifier, users repositories.UserRepository) fiber.Handler {
	logger = logger.WithService("middlewares.MCPDelegationAuth")

	return func(c fiber.Ctx) error {
		ctx, span, ctxLogger := tracer.StartFromFiberCtxWithLogger(c, logger)
		defer span.End()

		raw := bearerToken(c.Get(authHeaderBearer))
		if raw == "" {
			span.AddEvent("the request header has no MCP delegated bearer token")
			return c.Next()
		}

		claims, err := verifier.VerifyRequest(ctx, raw, c.Method(), c.Path())
		if err != nil {
			code := stacktrace.GetCode(err)
			if code == auth.ErrCodeInsufficientScope || code == auth.ErrCodeOperationDenied {
				ctxLogger.Warn(tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "MCP delegated token cannot access this API operation")))
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"status":  "error",
					"message": "MCP token cannot access this API operation",
				})
			}
			span.AddEvent("MCP delegated token is not valid; continuing to the next authentication middleware")
			return c.Next()
		}

		user, err := users.Load(ctx, entities.UserID(claims.Subject))
		if err != nil {
			ctxLogger.Warn(tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot load user for MCP delegated token subject")))
			return c.Next()
		}

		c.Locals(ContextKeyAuthUserID, entities.AuthContext{ID: user.ID, Email: user.Email})
		return c.Next()
	}
}

// bearerToken extracts the raw token from an Authorization header value using the Bearer scheme.
// It returns an empty string when the header is missing or does not use the Bearer scheme.
func bearerToken(header string) string {
	if !strings.HasPrefix(header, bearerScheme) {
		return ""
	}
	if len(header) <= len(bearerScheme)+1 {
		return ""
	}
	return header[len(bearerScheme)+1:]
}
