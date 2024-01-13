package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"github.com/redis/go-redis/v9"
)

// redisCache is the Cache implementation in redis
type redisCache struct {
	tracer telemetry.Tracer
	client *redis.Client
}

// NewRedisCache creates a new instance of RedisCache
func NewRedisCache(tracer telemetry.Tracer, client *redis.Client) Cache {
	return &redisCache{
		tracer: tracer,
		client: client,
	}
}

// Get an item from the redis cache
func (cache *redisCache) Get(ctx context.Context, key string) (value string, err error) {
	ctx, span := cache.tracer.Start(ctx)
	defer span.End()

	response, err := cache.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", stacktrace.Propagate(err, fmt.Sprintf("no item found in redis with key [%s]", key))
	}
	if err != nil {
		return "", stacktrace.Propagate(err, fmt.Sprintf("cannot get item in redis with key [%s]", key))
	}
	return response, nil
}

// Set an item in the redis cache
func (cache *redisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	ctx, span := cache.tracer.Start(ctx)
	defer span.End()

	err := cache.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return cache.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot set item in redis"))
	}
	return nil
}
