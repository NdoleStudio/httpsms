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
	rateLimitKeyPrefix      = "rate_limit:"
	rateLimitNotifiedPrefix = "rate_limit_notified:"
	rateLimitWindow         = 24 * time.Hour
	rateLimitFlushInterval  = 30 * time.Second
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
	svc := &RateLimitService{
		tracer:     tracer,
		logger:     logger,
		client:     client,
		dispatcher: dispatcher,
		counters:   make(map[string]*userCounter),
		notified:   make(map[string]bool),
		done:       make(chan struct{}),
	}

	go svc.flushLoop()
	return svc
}

// Increment adds cost to the user's counter and returns the current count.
// If the count exceeds the plan's rate limit, exceeded is true.
func (svc *RateLimitService) Increment(ctx context.Context, userID entities.UserID, plan entities.SubscriptionName, cost int) (count int64, exceeded bool, err error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	key := string(userID)
	counter, exists := svc.counters[key]

	if !exists {
		counter, err = svc.hydrate(ctx, key)
		if err != nil {
			counter = &userCounter{
				count:        0,
				windowExpiry: time.Now().Add(rateLimitWindow),
				dirty:        0,
			}
		}
		svc.counters[key] = counter
	}

	// Reset if window has expired
	if time.Now().After(counter.windowExpiry) {
		counter.count = 0
		counter.dirty = 0
		counter.windowExpiry = time.Now().Add(rateLimitWindow)
		svc.notified[key] = false
	}

	counter.count += int64(cost)
	counter.dirty += int64(cost)

	limit := plan.RateLimit()
	exceeded = counter.count > int64(limit)

	if exceeded && !svc.notified[key] {
		svc.notified[key] = true
		go svc.emitExceededEvent(ctx, userID, counter.count, limit, plan)
	}

	return counter.count, exceeded, nil
}

// Close flushes remaining dirty counters and stops the background goroutine.
func (svc *RateLimitService) Close() {
	close(svc.done)
	if svc.client != nil {
		svc.flush(context.Background())
	}
}

func (svc *RateLimitService) flushLoop() {
	ticker := time.NewTicker(rateLimitFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if svc.client != nil {
				svc.flush(context.Background())
			}
		case <-svc.done:
			return
		}
	}
}

func (svc *RateLimitService) flush(ctx context.Context) {
	svc.mu.Lock()
	// Collect dirty entries
	type flushEntry struct {
		key   string
		delta int64
		ttl   time.Duration
	}
	var entries []flushEntry
	now := time.Now()

	for key, counter := range svc.counters {
		// Clean up expired windows
		if now.After(counter.windowExpiry) {
			delete(svc.counters, key)
			delete(svc.notified, key)
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
	svc.mu.Unlock()

	// Flush to Redis outside the lock
	for _, entry := range entries {
		pipe := svc.client.Pipeline()
		pipe.IncrBy(ctx, entry.key, entry.delta)
		pipe.ExpireNX(ctx, entry.key, entry.ttl)
		if _, err := pipe.Exec(ctx); err != nil {
			if svc.logger != nil {
				svc.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot flush rate limit for key [%s]", entry.key)))
			}
		}
	}
}

func (svc *RateLimitService) hydrate(ctx context.Context, userID string) (*userCounter, error) {
	if svc.client == nil {
		return &userCounter{
			count:        0,
			windowExpiry: time.Now().Add(rateLimitWindow),
			dirty:        0,
		}, nil
	}

	key := rateLimitKeyPrefix + userID
	countStr, err := svc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return &userCounter{
			count:        0,
			windowExpiry: time.Now().Add(rateLimitWindow),
			dirty:        0,
		}, nil
	}
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot hydrate rate limit for user [%s]", userID))
	}

	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot parse rate limit count [%s] for user [%s]", countStr, userID))
	}

	ttl, err := svc.client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = rateLimitWindow
	}

	// Check if already notified (in Redis)
	notifiedKey := rateLimitNotifiedPrefix + userID
	if exists, _ := svc.client.Exists(ctx, notifiedKey).Result(); exists > 0 {
		svc.notified[userID] = true
	}

	return &userCounter{
		count:        count,
		windowExpiry: time.Now().Add(ttl),
		dirty:        0,
	}, nil
}

func (svc *RateLimitService) emitExceededEvent(ctx context.Context, userID entities.UserID, count int64, limit uint, plan entities.SubscriptionName) {
	if svc.dispatcher == nil {
		return
	}

	// Set notified flag in Redis (24h TTL) to survive restarts
	if svc.client != nil {
		notifiedKey := rateLimitNotifiedPrefix + string(userID)
		svc.client.Set(ctx, notifiedKey, "1", rateLimitWindow)
	}

	payload := events.RateLimitExceededPayload{
		UserID:    userID,
		Count:     count,
		Limit:     limit,
		Plan:      string(plan),
		Timestamp: time.Now().UTC(),
	}

	event, err := svc.createEvent(events.RateLimitExceeded, string(userID), payload)
	if err != nil {
		if svc.logger != nil {
			svc.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot create rate limit exceeded event for user [%s]", userID)))
		}
		return
	}

	if err = svc.dispatcher.Dispatch(ctx, event); err != nil {
		if svc.logger != nil {
			svc.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch rate limit exceeded event for user [%s]", userID)))
		}
	}
}
