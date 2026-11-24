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
	Allow(context.Context, entities.UserID, entities.SubscriptionName, int) (bool, time.Duration, error)
}

// RateLimit enforces hourly per-user budgets after authentication and before handlers.
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

		authUser, ok := c.Locals(ContextKeyAuthUserID).(entities.AuthContext)
		if !ok || authUser.IsNoop() {
			return c.Next()
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
			return c.Next()
		}

		allowed, retryAfter, err := service.Allow(ctx, userID, plan.(entities.SubscriptionName), cost)
		if err != nil {
			ctxLogger.Error(err)
			return c.Next()
		}
		if !allowed {
			seconds := max(1, int((retryAfter+time.Second-1)/time.Second))
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(seconds))
			ctxLogger.WithString("user_id", string(userID)).WithString("plan", string(plan.(entities.SubscriptionName))).Info("hourly API rate limit exceeded")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  "error",
				"message": "Hourly API rate limit exceeded. Try again after the time indicated by Retry-After.",
			})
		}

		return c.Next()
	}
}
