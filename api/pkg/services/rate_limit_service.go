package services

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/redis/go-redis/v9"
)

const (
	rateLimitKeyPrefix    = "user_rate_limit:hourly:"
	rateLimitWindow       = time.Hour
	rateLimitRedisTimeout = time.Second
)

//go:embed rate_limit_allow.lua
var rateLimitAllowScript string

// RateLimitService admits requests against shared fixed hourly user budgets.
type RateLimitService struct {
	tracer telemetry.Tracer
	logger telemetry.Logger
	client *redis.Client
}

// NewRateLimitService creates a synchronous Redis-backed admission service.
func NewRateLimitService(tracer telemetry.Tracer, logger telemetry.Logger, client *redis.Client) (rateLimiter *RateLimitService) {
	return &RateLimitService{
		tracer: tracer,
		logger: logger.WithService(fmt.Sprintf("%T", rateLimiter)),
		client: client,
	}
}

// Allow atomically admits and charges cost, or returns the remaining window TTL.
// Failures are returned with allowed=true so callers can log and fail open.
// An oversized request without an active window is denied with a zero TTL.
func (service *RateLimitService) Allow(ctx context.Context, userID entities.UserID, plan entities.SubscriptionName, cost int) (allowed bool, retryAfter time.Duration, err error) {
	ctx, span, _ := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()
	if cost <= 0 {
		return true, 0, service.tracer.WrapErrorSpan(span, stacktrace.NewErrorf("invalid rate limit cost [%d] for user [%s]", cost, userID))
	}
	if service.client == nil {
		return true, 0, service.tracer.WrapErrorSpan(span, stacktrace.NewErrorf("rate limit Redis client is not configured for user [%s]", userID))
	}
	ctx, cancel := context.WithTimeout(ctx, rateLimitRedisTimeout)
	defer cancel()
	result, err := service.client.Eval(ctx, rateLimitAllowScript, []string{rateLimitKeyPrefix + string(userID)},
		plan.Limit(), cost, rateLimitWindow.Milliseconds()).Int64Slice()
	if err != nil {
		return true, 0, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot check hourly rate limit for user [%s]", userID))
	}
	if len(result) != 2 || (result[0] != 0 && result[0] != 1) || result[1] < 0 || result[1] > rateLimitWindow.Milliseconds() {
		return true, 0, service.tracer.WrapErrorSpan(span, stacktrace.NewErrorf("invalid hourly rate limit result for user [%s]: %v", userID, result))
	}
	return result[0] == 1, time.Duration(result[1]) * time.Millisecond, nil
}
