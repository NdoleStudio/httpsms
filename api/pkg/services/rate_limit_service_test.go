package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestRateLimitService_Increment_BasicCount(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Act
	count, exceeded, err := svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.False(t, exceeded)
}

func TestRateLimitService_Increment_WeightedCost(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Act
	count, _, err := svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 10)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)
}

func TestRateLimitService_Increment_ExceedsLimit(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Free plan limit is 400. Exceed it.
	for i := 0; i < 400; i++ {
		_, _, _ = svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 1)
	}

	// Act — this pushes count to 401
	count, exceeded, err := svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(401), count)
	assert.True(t, exceeded)
}

func TestRateLimitService_Increment_MultipleUsers(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Act
	_, _, _ = svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 5)
	count, _, err := svc.Increment(ctx, "user-2", entities.SubscriptionNameProMonthly, 3)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestRateLimitService_Increment_WindowExpiry(t *testing.T) {
	// Arrange
	svc := newTestRateLimitService(t)
	defer svc.Close()

	ctx := context.Background()

	// Simulate an existing counter with an expired window
	svc.mu.Lock()
	svc.counters["user-1"] = &userCounter{
		count:        500,
		windowExpiry: time.Now().Add(-1 * time.Hour), // expired
		dirty:        0,
	}
	svc.mu.Unlock()

	// Act — should reset because the window expired
	count, exceeded, err := svc.Increment(ctx, "user-1", entities.SubscriptionNameFree, 1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.False(t, exceeded)
}

// newTestRateLimitService creates a RateLimitService with nil redis client (no hydration)
// suitable for unit tests that only test in-memory logic.
func newTestRateLimitService(t *testing.T) *RateLimitService {
	t.Helper()
	return NewRateLimitService(nil, &testLogger{}, nil, nil)
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
func (l *testLogger) WithSpan(trace.SpanContext) telemetry.Logger { return l }
func (l *testLogger) Printf(string, ...interface{})              {}
