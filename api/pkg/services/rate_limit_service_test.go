package services

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func newTestRateLimitService(t *testing.T, server *miniredis.Miniredis) *RateLimitService {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: server.Addr(), ContextTimeoutEnabled: true, MaxRetries: -1,
		DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second, PoolTimeout: time.Second,
	})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	logger := &testLogger{}
	return NewRateLimitService(telemetry.NewOtelLogger("test", logger), logger, client)
}

func TestRateLimitService_FreeHourlyBudget(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	for range 200 {
		allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
		require.NoError(t, err)
		require.True(t, allowed)
		require.Zero(t, retryAfter)
	}
	for range 2 {
		allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
		require.NoError(t, err)
		assert.False(t, allowed)
		assert.InDelta(t, time.Hour.Milliseconds(), retryAfter.Milliseconds(), 1000)
	}
	count, err := server.Get("user_rate_limit:hourly:user")
	require.NoError(t, err)
	assert.Equal(t, "200", count)
}

func TestRateLimitService_WeightedDenialDoesNotSpendRemainingBudget(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	for _, test := range []struct {
		cost    int
		allowed bool
		total   string
	}{
		{190, true, "190"},
		{11, false, "190"},
		{10, true, "200"},
		{1, false, "200"},
	} {
		allowed, _, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, test.cost)
		require.NoError(t, err)
		assert.Equal(t, test.allowed, allowed)
		count, err := server.Get("user_rate_limit:hourly:user")
		require.NoError(t, err)
		assert.Equal(t, test.total, count)
	}
}

func TestRateLimitService_FirstDeniedRequestDoesNotStartWindow(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 201)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Zero(t, retryAfter)
	assert.False(t, server.Exists("user_rate_limit:hourly:user"))
	allowed, _, err = service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 200)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, time.Hour, server.TTL("user_rate_limit:hourly:user"))
}

func TestRateLimitService_ConcurrentInstancesNeverOversubscribe(t *testing.T) {
	server := miniredis.RunT(t)
	var admitted atomic.Int64
	failures := make(chan error, 400)
	var workers sync.WaitGroup
	for range 4 {
		service := newTestRateLimitService(t, server)
		workers.Go(func() {
			for range 100 {
				allowed, _, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
				if err != nil {
					failures <- err
				} else if allowed {
					admitted.Add(1)
				}
			}
		})
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		assert.NoError(t, err)
	}
	assert.Equal(t, int64(200), admitted.Load())
	count, err := server.Get("user_rate_limit:hourly:user")
	require.NoError(t, err)
	assert.Equal(t, "200", count)
}

func TestRateLimitService_RestartSeesSharedBudget(t *testing.T) {
	server := miniredis.RunT(t)
	first := newTestRateLimitService(t, server)
	allowed, _, err := first.Allow(context.Background(), "user", entities.SubscriptionNameFree, 199)
	require.NoError(t, err)
	require.True(t, allowed)
	second := newTestRateLimitService(t, server)
	allowed, _, err = second.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, _, err = second.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestRateLimitService_AdmitsAndDenialsNeverExtendWindow(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	allowed, _, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 100)
	require.NoError(t, err)
	require.True(t, allowed)
	server.FastForward(30*time.Minute + 500*time.Millisecond)
	allowed, _, err = service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 30*time.Minute-500*time.Millisecond, server.TTL("user_rate_limit:hourly:user"))
	allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 100)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 30*time.Minute-500*time.Millisecond, retryAfter)
	assert.Equal(t, retryAfter, server.TTL("user_rate_limit:hourly:user"))
}

func TestRateLimitService_WindowResetsAfterExactlyOneHour(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	allowed, _, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 200)
	require.NoError(t, err)
	require.True(t, allowed)
	server.FastForward(3599 * time.Second)
	allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, time.Second, retryAfter)
	server.FastForward(time.Second)
	allowed, _, err = service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 200)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, time.Hour, server.TTL("user_rate_limit:hourly:user"))
}

func TestRateLimitService_SubscriptionHourlyBudgets(t *testing.T) {
	for _, test := range []struct {
		plan  entities.SubscriptionName
		limit int
	}{
		{entities.SubscriptionNameFree, 200},
		{entities.SubscriptionNameProMonthly, 5000},
		{entities.SubscriptionNameProYearly, 5000},
		{entities.SubscriptionNameProLifetime, 5000},
		{entities.SubscriptionNameUltraMonthly, 10000},
		{entities.SubscriptionNameUltraYearly, 10000},
		{entities.SubscriptionName20KMonthly, 20000},
		{entities.SubscriptionName20KYearly, 20000},
		{entities.SubscriptionName50KMonthly, 50000},
		{entities.SubscriptionName100KMonthly, 100000},
		{entities.SubscriptionName200KMonthly, 200000},
	} {
		t.Run(string(test.plan), func(t *testing.T) {
			server := miniredis.RunT(t)
			service := newTestRateLimitService(t, server)
			allowed, _, err := service.Allow(context.Background(), "user", test.plan, test.limit)
			require.NoError(t, err)
			assert.True(t, allowed)
			allowed, _, err = service.Allow(context.Background(), "user", test.plan, 1)
			require.NoError(t, err)
			assert.False(t, allowed)
		})
	}
}

func TestRateLimitService_IgnoresLegacyDailyNamespace(t *testing.T) {
	server := miniredis.RunT(t)
	require.NoError(t, server.Set("user_rate_limit:user", "999999"))
	service := newTestRateLimitService(t, server)
	allowed, _, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 200)
	require.NoError(t, err)
	assert.True(t, allowed)
	legacy, err := server.Get("user_rate_limit:user")
	require.NoError(t, err)
	assert.Equal(t, "999999", legacy)
}

func TestRateLimitService_MalformedStateFailsOpenWithError(t *testing.T) {
	for _, state := range []string{"not-an-integer", "-1", "3.5", "1e2", "0200", "9007199254740992", "no-expiry", "long-expiry", "wrong-type"} {
		t.Run(state, func(t *testing.T) {
			server := miniredis.RunT(t)
			key := "user_rate_limit:hourly:user"
			switch state {
			case "no-expiry":
				require.NoError(t, server.Set(key, "10"))
			case "long-expiry":
				require.NoError(t, server.Set(key, "10"))
				server.SetTTL(key, 2*time.Hour)
			case "wrong-type":
				server.HSet(key, "count", "10")
				server.SetTTL(key, time.Hour)
			default:
				require.NoError(t, server.Set(key, state))
				server.SetTTL(key, time.Hour)
			}
			service := newTestRateLimitService(t, server)
			allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "user")
			assert.True(t, allowed)
			assert.Zero(t, retryAfter)
		})
	}
}

func TestRateLimitService_RedisOutageFailsOpen(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	server.Close()
	start := time.Now()
	allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
	require.Error(t, err)
	assert.True(t, allowed)
	assert.Zero(t, retryAfter)
	assert.Less(t, time.Since(start), 1500*time.Millisecond)
}

func TestRateLimitService_RedisDeadlineIsOneSecond(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	service.client.AddHook(&rateLimitProcessHook{run: func(ctx context.Context, _ redis.Cmder, _ redis.ProcessHook) error {
		<-ctx.Done()
		return ctx.Err()
	}})
	start := time.Now()
	allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.True(t, allowed)
	assert.Zero(t, retryAfter)
	assert.Less(t, time.Since(start), 1500*time.Millisecond)
}

func TestRateLimitService_RequestCancellationFailsOpenWithoutSpending(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	allowed, retryAfter, err := service.Allow(ctx, "user", entities.SubscriptionNameFree, 1)
	require.ErrorIs(t, err, context.Canceled)
	assert.True(t, allowed)
	assert.Zero(t, retryAfter)
	assert.False(t, server.Exists("user_rate_limit:hourly:user"))
}

func TestRateLimitService_OneRedisOperationPreservesTraceCorrelation(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	require.NoError(t, service.client.Ping(context.Background()).Err())
	correlation := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}})
	ctx := trace.ContextWithSpanContext(context.Background(), correlation)
	var commands []string
	service.client.AddHook(&rateLimitProcessHook{run: func(ctx context.Context, cmd redis.Cmder, next redis.ProcessHook) error {
		commands = append(commands, cmd.Name())
		assert.Equal(t, correlation.TraceID(), trace.SpanContextFromContext(ctx).TraceID())
		return next(ctx, cmd)
	}})
	allowed, _, err := service.Allow(ctx, "user", entities.SubscriptionNameFree, 1)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, []string{"eval"}, commands)
}

func TestRateLimitService_InvalidCostCannotReduceUsage(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	allowed, _, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 200)
	require.NoError(t, err)
	require.True(t, allowed)
	for _, cost := range []int{0, -1} {
		allowed, _, err = service.Allow(context.Background(), "user", entities.SubscriptionNameFree, cost)
		require.Error(t, err)
		assert.True(t, allowed)
	}
	allowed, _, err = service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestRateLimitService_UserBudgetsAreIndependent(t *testing.T) {
	server := miniredis.RunT(t)
	service := newTestRateLimitService(t, server)
	for _, user := range []entities.UserID{"first", "second"} {
		allowed, _, err := service.Allow(context.Background(), user, entities.SubscriptionNameFree, 200)
		require.NoError(t, err)
		assert.True(t, allowed)
	}
}

func TestRateLimitService_MissingRedisClientFailsOpenWithError(t *testing.T) {
	logger := &testLogger{}
	service := NewRateLimitService(telemetry.NewOtelLogger("test", logger), logger, nil)
	allowed, retryAfter, err := service.Allow(context.Background(), "user", entities.SubscriptionNameFree, 1)
	require.Error(t, err)
	assert.True(t, allowed)
	assert.Zero(t, retryAfter)
}

type rateLimitProcessHook struct {
	run func(context.Context, redis.Cmder, redis.ProcessHook) error
}

func (h *rateLimitProcessHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *rateLimitProcessHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *rateLimitProcessHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error { return h.run(ctx, cmd, next) }
}

type testLogger struct{}

func (l *testLogger) Info(string)                                {}
func (l *testLogger) Error(error)                                {}
func (l *testLogger) Warn(error)                                 {}
func (l *testLogger) Fatal(error)                                {}
func (l *testLogger) Trace(string)                               {}
func (l *testLogger) Debug(string)                               {}
func (l *testLogger) WithService(string) telemetry.Logger        { return l }
func (l *testLogger) WithString(string, string) telemetry.Logger { return l }
func (l *testLogger) WithSpan(trace.Span) telemetry.Logger       { return l }
func (l *testLogger) Printf(string, ...interface{})              {}
