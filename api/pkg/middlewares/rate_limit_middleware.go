package middlewares

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v3"
	ttlCache "github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

const rateLimitCostCap = 200

type rateLimitCounter interface {
	Increment(context.Context, entities.UserID, entities.SubscriptionName, int) (int64, bool, error)
}

// RateLimit tracks per-user API request counts without blocking requests.
func RateLimit(
	tracer telemetry.Tracer,
	logger telemetry.Logger,
	service rateLimitCounter,
	userRepository repositories.UserRepository,
	excludePaths []string,
) fiber.Handler {
	logger = logger.WithService("middlewares.RateLimit")
	plans := ttlCache.New(time.Minute, time.Minute)
	var loads singleflight.Group

	return func(c fiber.Ctx) error {
		path := c.Path()
		for _, excluded := range excludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				return c.Next()
			}
		}

		ctx, span, ctxLogger := tracer.StartFromFiberCtxWithLogger(c, logger, "middlewares.RateLimit")
		defer span.End()

		// Route-specific phone and integration authentication runs inside Next.
		nextErr := c.Next()
		authUser, ok := c.Locals(ContextKeyAuthUserID).(entities.AuthContext)
		if !ok || authUser.IsNoop() {
			return nextErr
		}
		// Cache keys outlive Fiber's reusable request buffers.
		userID := entities.UserID(strings.Clone(string(authUser.ID)))

		cost := 1
		if c.Method() == fiber.MethodGet {
			if limitParam := c.Query("limit"); limitParam != "" {
				if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
					cost = min(parsed, rateLimitCostCap)
				}
			}
		}

		plan, err, _ := loads.Do(string(userID), func() (interface{}, error) {
			if cached, found := plans.Get(string(userID)); found {
				return cached, nil
			}
			loadCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			user, err := userRepository.Load(loadCtx, userID)
			if err != nil {
				return nil, err
			}
			plans.SetDefault(string(userID), user.SubscriptionName)
			return user.SubscriptionName, nil
		})
		if err != nil {
			ctxLogger.Error(err)
			return nextErr
		}

		if _, _, err := service.Increment(ctx, userID, plan.(entities.SubscriptionName), cost); err != nil {
			ctxLogger.Error(err)
		}

		return nextErr
	}
}
