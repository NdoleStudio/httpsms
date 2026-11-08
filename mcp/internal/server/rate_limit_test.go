package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/server"
)

// newRateLimitTestRedis starts an in-process miniredis instance and returns
// a standalone *redis.Client pointed at it, matching the standalone-client
// requirement every Redis-backed component in this service shares.
func newRateLimitTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestToolRateLimiterSeparatesUsersAndTools(t *testing.T) {
	ctx := context.Background()
	redisClient := newRateLimitTestRedis(t)

	limiter := server.NewToolRateLimiter(redisClient, server.Limits{ReadPerMinute: 2})
	require.NoError(t, limiter.Allow(ctx, "user-a", "list_phones"))
	require.NoError(t, limiter.Allow(ctx, "user-a", "list_phones"))
	require.ErrorIs(t, limiter.Allow(ctx, "user-a", "list_phones"), server.ErrRateLimited)
	require.NoError(t, limiter.Allow(ctx, "user-b", "list_phones"))
}

func TestToolRateLimiterSeparatesToolBuckets(t *testing.T) {
	ctx := context.Background()
	redisClient := newRateLimitTestRedis(t)

	limiter := server.NewToolRateLimiter(redisClient, server.Limits{ReadPerMinute: 1, SendPerMinute: 1})
	require.NoError(t, limiter.Allow(ctx, "user-a", "list_phones"))
	require.ErrorIs(t, limiter.Allow(ctx, "user-a", "list_phones"), server.ErrRateLimited)
	// send_sms has its own, independent budget from list_phones.
	require.NoError(t, limiter.Allow(ctx, "user-a", "send_sms"))
	require.ErrorIs(t, limiter.Allow(ctx, "user-a", "send_sms"), server.ErrRateLimited)
}

func TestToolRateLimiterAppliesReadSendKeyCreateAndKeyRotateBudgets(t *testing.T) {
	ctx := context.Background()
	redisClient := newRateLimitTestRedis(t)

	limiter := server.NewToolRateLimiter(redisClient, server.Limits{
		ReadPerMinute:       1,
		SendPerMinute:       1,
		KeyCreatesPerHour:   1,
		KeyRotationsPerHour: 1,
	})

	for _, tool := range []string{
		"list_phones", "list_message_threads", "list_thread_messages", "list_incoming_messages",
		"send_sms", "create_phone_api_key", "rotate_user_api_key",
	} {
		require.NoError(t, limiter.Allow(ctx, "user-a", tool), "first call to %q", tool)
		require.ErrorIs(t, limiter.Allow(ctx, "user-a", tool), server.ErrRateLimited, "second call to %q", tool)
	}
}

func TestToolRateLimiterAllowsUnlimitedTools(t *testing.T) {
	ctx := context.Background()
	redisClient := newRateLimitTestRedis(t)

	// A tool with no configured bucket (or a non-positive limit) is never
	// rate limited, and must never touch Redis.
	limiter := server.NewToolRateLimiter(redisClient, server.Limits{})
	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.Allow(ctx, "user-a", "list_phones"))
	}
}

func TestToolRateLimiterErrorUnwrapsToErrRateLimitedAndCarriesRetryAfter(t *testing.T) {
	ctx := context.Background()
	redisClient := newRateLimitTestRedis(t)

	limiter := server.NewToolRateLimiter(redisClient, server.Limits{ReadPerMinute: 1})
	require.NoError(t, limiter.Allow(ctx, "user-a", "list_phones"))

	err := limiter.Allow(ctx, "user-a", "list_phones")
	require.Error(t, err)
	require.ErrorIs(t, err, server.ErrRateLimited)

	var rateLimitErr *server.RateLimitError
	require.ErrorAs(t, err, &rateLimitErr)
	require.Equal(t, "list_phones", rateLimitErr.Tool)
	require.Greater(t, rateLimitErr.RetryAfter, time.Duration(0))
	require.LessOrEqual(t, rateLimitErr.RetryAfter, time.Minute)
}

func TestToolRateLimiterFailsClosedOnRedisError(t *testing.T) {
	ctx := context.Background()

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	mr.Close() // force every subsequent command to fail

	limiter := server.NewToolRateLimiter(redisClient, server.Limits{ReadPerMinute: 5})
	err := limiter.Allow(ctx, "user-a", "list_phones")
	require.Error(t, err)
	require.NotErrorIs(t, err, server.ErrRateLimited)
}

// TestToolRateLimiterRotationConfirmationBucketIsIndependentAndDerived
// asserts the confirmation-prompt bucket ("rotate_user_api_key:confirm")
// has its own budget -- KeyRotationsPerHour * 5 -- entirely independent of
// the execution bucket ("rotate_user_api_key"): exhausting one never
// affects the other, in either direction.
func TestToolRateLimiterRotationConfirmationBucketIsIndependentAndDerived(t *testing.T) {
	ctx := context.Background()
	redisClient := newRateLimitTestRedis(t)

	limiter := server.NewToolRateLimiter(redisClient, server.Limits{
		ReadPerMinute: 1, SendPerMinute: 1, KeyCreatesPerHour: 1, KeyRotationsPerHour: 1,
	})

	// The confirmation-prompt bucket's budget is derived as
	// KeyRotationsPerHour * 5 = 5, independent of the execution bucket's
	// budget of 1.
	for i := 0; i < 5; i++ {
		require.NoErrorf(t, limiter.Allow(ctx, "user-a", "rotate_user_api_key:confirm"), "confirmation call %d", i+1)
	}
	require.ErrorIs(t, limiter.Allow(ctx, "user-a", "rotate_user_api_key:confirm"), server.ErrRateLimited)

	// Exhausting the confirmation-prompt bucket left the execution bucket
	// untouched: it still has its full budget of 1.
	require.NoError(t, limiter.Allow(ctx, "user-a", "rotate_user_api_key"))
	require.ErrorIs(t, limiter.Allow(ctx, "user-a", "rotate_user_api_key"), server.ErrRateLimited)

	// Conversely, exhausting the execution bucket does not touch the
	// confirmation-prompt bucket for a different user.
	require.NoError(t, limiter.Allow(ctx, "user-b", "rotate_user_api_key"))
	for i := 0; i < 5; i++ {
		require.NoErrorf(t, limiter.Allow(ctx, "user-b", "rotate_user_api_key:confirm"), "confirmation call %d", i+1)
	}
	require.ErrorIs(t, limiter.Allow(ctx, "user-b", "rotate_user_api_key:confirm"), server.ErrRateLimited)
}

// TestToolRateLimiterRotationConfirmationBucketFailsClosedOnRedisError
// asserts a Redis failure on the confirmation-prompt bucket is reported as
// its own error (never ErrRateLimited, and never silently "allowed"), the
// same fail-closed guarantee the execution bucket already has.
func TestToolRateLimiterRotationConfirmationBucketFailsClosedOnRedisError(t *testing.T) {
	ctx := context.Background()

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	mr.Close() // force every subsequent command to fail

	limiter := server.NewToolRateLimiter(redisClient, server.Limits{KeyRotationsPerHour: 5})
	err := limiter.Allow(ctx, "user-a", "rotate_user_api_key:confirm")
	require.Error(t, err)
	require.NotErrorIs(t, err, server.ErrRateLimited)
}
