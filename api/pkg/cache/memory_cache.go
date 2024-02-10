package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	ttlCache "github.com/patrickmn/go-cache"
)

// memoryCache is the Cache implementation in memory
type memoryCache struct {
	tracer telemetry.Tracer
	store  *ttlCache.Cache
}

// NewMemoryCache creates a new instance of memoryCache
func NewMemoryCache(tracer telemetry.Tracer, store *ttlCache.Cache) Cache {
	return &memoryCache{
		tracer: tracer,
		store:  store,
	}
}

// Get an item from the redis cache
func (cache *memoryCache) Get(ctx context.Context, key string) (value string, err error) {
	ctx, span := cache.tracer.Start(ctx)
	defer span.End()

	response, ok := cache.store.Get(key)
	if !ok {
		return "", stacktrace.NewError(fmt.Sprintf("no item found in cache with key [%s]", key))
	}

	return response.(string), nil
}

// Set an item in the redis cache
func (cache *memoryCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	ctx, span := cache.tracer.Start(ctx)
	defer span.End()

	cache.store.Set(key, value, ttl)
	return nil
}
