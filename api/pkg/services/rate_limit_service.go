package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"github.com/redis/go-redis/v9"
)

const (
	rateLimitKeyPrefix      = "user_rate_limit:"
	rateLimitNotifiedPrefix = "user_rate_limit_notified:"
	rateLimitWindow         = 24 * time.Hour
	rateLimitFlushInterval  = 30 * time.Second
	rateLimitRedisTimeout   = 1 * time.Second
)

// RateLimitService tracks per-user API request counts with in-memory counters
// and periodic Redis sync.
type RateLimitService struct {
	service
	tracer     telemetry.Tracer
	logger     telemetry.Logger
	client     *redis.Client
	dispatcher *EventDispatcher

	mu       sync.Mutex
	counters map[string]*userCounter
	notified map[string]bool
	done     chan struct{}
}

type userCounter struct {
	count        int64
	windowExpiry time.Time
	dirty        int64
}

// NewRateLimitService creates a new RateLimitService and starts the background flush goroutine.
func NewRateLimitService(
	tracer telemetry.Tracer,
	logger telemetry.Logger,
	client *redis.Client,
	dispatcher *EventDispatcher,
) *RateLimitService {
	rateLimiter := &RateLimitService{
		tracer:     tracer,
		logger:     logger,
		client:     client,
		dispatcher: dispatcher,
		counters:   make(map[string]*userCounter),
		notified:   make(map[string]bool),
		done:       make(chan struct{}),
	}

	go rateLimiter.flushLoop()
	return rateLimiter
}

// Increment adds cost to the user's counter and returns the current count.
// If the count exceeds the plan's rate limit, exceeded is true.
func (service *RateLimitService) Increment(ctx context.Context, userID entities.UserID, plan entities.SubscriptionName, cost int) (count int64, exceeded bool, err error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	key := string(userID)
	counter, exists := service.counters[key]

	if !exists {
		counter, err = service.hydrate(ctx, key)
		if err != nil {
			counter = &userCounter{
				count:        0,
				windowExpiry: time.Now().Add(rateLimitWindow),
				dirty:        0,
			}
		}
		service.counters[key] = counter
	}

	// Reset if window has expired
	if time.Now().After(counter.windowExpiry) {
		counter.count = 0
		counter.dirty = 0
		counter.windowExpiry = time.Now().Add(rateLimitWindow)
		service.notified[key] = false
	}

	counter.count += int64(cost)
	counter.dirty += int64(cost)

	limit := plan.RateLimit()
	exceeded = counter.count > int64(limit)

	if exceeded && !service.notified[key] {
		service.notified[key] = true
		go service.emitExceededEvent(ctx, userID, counter.count, limit, plan)
	}

	return counter.count, exceeded, nil
}

// Close flushes remaining dirty counters and stops the background goroutine.
func (service *RateLimitService) Close() {
	service.logger.Info("RateLimitService shutting down, flushing counters")
	close(service.done)
	if service.client != nil {
		service.flush(context.Background())
	}
	service.logger.Info("RateLimitService shutdown complete")
}

func (service *RateLimitService) flushLoop() {
	ticker := time.NewTicker(rateLimitFlushInterval)
	defer ticker.Stop()

	service.logger.Info(fmt.Sprintf("RateLimitService flush loop started (interval: %s)", rateLimitFlushInterval))

	for {
		select {
		case <-ticker.C:
			if service.client != nil {
				service.flush(context.Background())
			}
		case <-service.done:
			service.logger.Info("RateLimitService flush loop stopped")
			return
		}
	}
}

func (service *RateLimitService) flush(ctx context.Context) {
	service.mu.Lock()
	// Collect dirty entries
	type flushEntry struct {
		key   string
		delta int64
		ttl   time.Duration
	}
	var entries []flushEntry
	now := time.Now()

	for key, counter := range service.counters {
		// Clean up expired windows
		if now.After(counter.windowExpiry) {
			delete(service.counters, key)
			delete(service.notified, key)
			continue
		}
		if counter.dirty > 0 {
			entries = append(entries, flushEntry{
				key:   rateLimitKeyPrefix + key,
				delta: counter.dirty,
				ttl:   time.Until(counter.windowExpiry),
			})
			counter.dirty = 0
		}
	}
	service.mu.Unlock()

	// Flush to Redis outside the lock with timeout using a single pipeline batch
	redisCtx, cancel := context.WithTimeout(ctx, 10*rateLimitRedisTimeout)
	defer cancel()

	pipe := service.client.Pipeline()
	for _, entry := range entries {
		pipe.IncrBy(redisCtx, entry.key, entry.delta)
		pipe.ExpireNX(redisCtx, entry.key, entry.ttl)
	}
	if len(entries) > 0 {
		if _, err := pipe.Exec(redisCtx); err != nil {
			service.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot flush rate limit batch of [%d] entries", len(entries))))
		}
	}
}

func (service *RateLimitService) hydrate(ctx context.Context, userID string) (*userCounter, error) {
	if service.client == nil {
		return &userCounter{
			count:        0,
			windowExpiry: time.Now().Add(rateLimitWindow),
			dirty:        0,
		}, nil
	}

	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	redisCtx, cancel := context.WithTimeout(ctx, rateLimitRedisTimeout)
	defer cancel()

	key := rateLimitKeyPrefix + userID
	countStr, err := service.client.Get(redisCtx, key).Result()
	if err == redis.Nil {
		return &userCounter{
			count:        0,
			windowExpiry: time.Now().Add(rateLimitWindow),
			dirty:        0,
		}, nil
	}
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot hydrate rate limit for user [%s]", userID)))
	}

	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot parse rate limit count [%s] for user [%s]", countStr, userID)))
	}

	ttl, err := service.client.TTL(redisCtx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = rateLimitWindow
	}

	// Check if already notified (in Redis)
	notifiedKey := rateLimitNotifiedPrefix + userID
	if exists, _ := service.client.Exists(redisCtx, notifiedKey).Result(); exists > 0 {
		service.notified[userID] = true
	}

	ctxLogger.Info(fmt.Sprintf("hydrated rate limit for user [%s] with count [%d]", userID, count))

	return &userCounter{
		count:        count,
		windowExpiry: time.Now().Add(ttl),
		dirty:        0,
	}, nil
}

func (service *RateLimitService) emitExceededEvent(ctx context.Context, userID entities.UserID, count int64, limit uint, plan entities.SubscriptionName) {
	if service.dispatcher == nil {
		return
	}

	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	// Set notified flag in Redis (24h TTL) to survive restarts
	if service.client != nil {
		redisCtx, cancel := context.WithTimeout(ctx, rateLimitRedisTimeout)
		defer cancel()
		notifiedKey := rateLimitNotifiedPrefix + string(userID)
		if err := service.client.Set(redisCtx, notifiedKey, "1", rateLimitWindow).Err(); err != nil {
			ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot persist rate limit notified flag for user [%s]", userID))))
		}
	}

	payload := events.RateLimitExceededPayload{
		UserID:    userID,
		Count:     count,
		Limit:     limit,
		Plan:      string(plan),
		Timestamp: time.Now().UTC(),
	}

	event, err := service.createEvent(events.RateLimitExceeded, string(userID), payload)
	if err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot create rate limit exceeded event for user [%s]", userID))))
		return
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch rate limit exceeded event for user [%s]", userID))))
	}
}
