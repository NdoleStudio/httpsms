package middlewares

import (
	"strconv"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v3"
)

const rateLimitCostCap = 100

// RateLimit tracks per-user API request counts without blocking requests.
func RateLimit(
	tracer telemetry.Tracer,
	logger telemetry.Logger,
	service *services.RateLimitService,
	userRepository repositories.UserRepository,
	excludePaths []string,
) fiber.Handler {
	logger = logger.WithService("middlewares.RateLimit")

	return func(c fiber.Ctx) error {
		path := c.Path()
		for _, excluded := range excludePaths {
			if strings.HasPrefix(path, excluded) {
				return c.Next()
			}
		}

		ctx, span := tracer.StartFromFiberCtx(c, "middlewares.RateLimit")
		defer span.End()

		authUser, ok := c.Locals(ContextKeyAuthUserID).(entities.AuthContext)
		if !ok || authUser.IsNoop() {
			return c.Next()
		}

		cost := 1
		if c.Method() == fiber.MethodGet {
			if limitParam := c.Query("limit"); limitParam != "" {
				if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
					cost = min(parsed, rateLimitCostCap)
				}
			}
		}

		user, err := userRepository.Load(ctx, authUser.ID)
		if err != nil {
			ctxLogger := tracer.CtxLogger(logger, span)
			ctxLogger.Error(err)
			return c.Next()
		}

		_, _, _ = service.Increment(ctx, authUser.ID, user.SubscriptionName, cost)

		return c.Next()
	}
}
