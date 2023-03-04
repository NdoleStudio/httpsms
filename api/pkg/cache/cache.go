package cache

import (
	"context"
	"time"
)

// Cache stores items temporarily
type Cache interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (value string, err error)
}
