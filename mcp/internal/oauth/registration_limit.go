package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultRegistrationsPerMinute is the global budget
// NewRedisRegistrationLimiter applies to POST /oauth/register when the
// caller does not choose its own.
//
// Dynamic Client Registration is the only unauthenticated write this
// service exposes: every accepted request mints a client_id and persists a
// Client record for dynamicClientTTL (24 hours). Without a ceiling a single
// unauthenticated caller can fill Redis with 24-hour records indefinitely.
// The budget is deliberately global rather than per-caller: the only
// caller identifiers available on an unauthenticated request are the
// transport peer address and client-IP headers (X-Forwarded-For,
// X-Real-IP, CF-Connecting-IP), and every one of those is either
// attacker-spoofable or trivially rotated behind a proxy, so keying the
// counter on them would produce a limit that does not actually bound
// anything. 60 registrations per minute is far above what any real client
// population needs (a client registers once and reuses its client_id for
// 24 hours) while capping worst-case growth at a bounded number of records
// per day.
const DefaultRegistrationsPerMinute = 60

// registrationRateLimitWindow is the fixed window
// RedisRegistrationLimiter counts registrations in.
const registrationRateLimitWindow = time.Minute

// keyPrefixRegistrationLimit namespaces the DCR registration counter. Like
// every other key in this package it is a fixed, approved prefix; unlike
// the others it has no per-record suffix beyond the window start, because
// the budget is global.
const keyPrefixRegistrationLimit = "httpsms:mcp:oauth:register:ratelimit:"

// ErrRegistrationRateLimited is returned (wrapped in a
// *RegistrationRateLimitError) by RegistrationLimiter.Allow once the
// global registration budget for the current window is exhausted. Callers
// should use errors.Is against this sentinel rather than matching error
// strings.
var ErrRegistrationRateLimited = errors.New("oauth: dynamic client registration rate limit exceeded")

// RegistrationRateLimitError reports an exhausted registration budget and
// carries how long the caller should wait before retrying. It wraps
// ErrRegistrationRateLimited so errors.Is reports true.
type RegistrationRateLimitError struct {
	// RetryAfter is how long until the current fixed window ends and a
	// fresh budget becomes available.
	RetryAfter time.Duration
}

func (e *RegistrationRateLimitError) Error() string {
	return fmt.Sprintf("oauth: dynamic client registration rate limit exceeded, retry after %s", e.RetryAfter)
}

// Unwrap allows errors.Is(err, ErrRegistrationRateLimited) to succeed.
func (e *RegistrationRateLimitError) Unwrap() error { return ErrRegistrationRateLimited }

// RegistrationLimiter bounds how many Dynamic Client Registration requests
// POST /oauth/register accepts. Allow must be called -- and must return
// nil -- before a registration handler generates a client_id or writes a
// Client record.
//
// Allow returns a *RegistrationRateLimitError (unwrapping to
// ErrRegistrationRateLimited) when the budget is exhausted, and any other
// error when it could not determine whether the budget is exhausted.
// Implementations must fail closed: an unavailable backing store is
// reported as an error, never silently treated as "allowed".
type RegistrationLimiter interface {
	Allow(ctx context.Context) error
}

// registrationRateLimitScript atomically increments the current window's
// registration counter and, only on the first increment, sets the window's
// expiry. A Lua script run through EVAL is the only way to make
// "increment" and "set the window's expiry" a single indivisible
// operation: INCR followed by a separate PEXPIRE would leave a counter
// without a TTL if the process died between the two commands, permanently
// pinning the budget at zero for every later caller.
var registrationRateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

// RedisRegistrationLimiter is the Redis-backed RegistrationLimiter. Like
// RedisStore it requires a standalone Redis client (redis.NewClient),
// never a Redis Cluster or Ring client.
type RedisRegistrationLimiter struct {
	client redis.UniversalClient
	limit  int
	window time.Duration
}

// NewRedisRegistrationLimiter returns a RegistrationLimiter allowing at
// most limit registrations per fixed one-minute window across the whole
// deployment. A non-positive limit falls back to
// DefaultRegistrationsPerMinute rather than disabling the limit, so a
// mis-wired budget can never silently reopen the unbounded-registration
// hole this limiter exists to close.
//
// client must be a standalone Redis client; a *redis.ClusterClient or
// *redis.Ring panics, matching NewRedisStore.
func NewRedisRegistrationLimiter(client redis.UniversalClient, limit int) *RedisRegistrationLimiter {
	switch client.(type) {
	case *redis.ClusterClient, *redis.Ring:
		panic("oauth: NewRedisRegistrationLimiter requires a standalone Redis client (redis.NewClient)")
	}

	if limit <= 0 {
		limit = DefaultRegistrationsPerMinute
	}

	return &RedisRegistrationLimiter{client: client, limit: limit, window: registrationRateLimitWindow}
}

// Allow implements RegistrationLimiter. It atomically charges one
// registration against the current window and returns a
// *RegistrationRateLimitError once the window's budget is exhausted.
//
// A Redis failure is returned as its own error (never
// ErrRegistrationRateLimited): this limiter fails closed, and the caller
// must refuse the registration rather than persist a client it could not
// account for.
func (l *RedisRegistrationLimiter) Allow(ctx context.Context) error {
	windowStart := time.Now().UTC().Truncate(l.window)
	key := fmt.Sprintf("%s%d", keyPrefixRegistrationLimit, windowStart.Unix())

	count, err := registrationRateLimitScript.Run(ctx, l.client, []string{key}, l.window.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("oauth: cannot check the dynamic client registration rate limit: %w", err)
	}

	if count > l.limit {
		return &RegistrationRateLimitError{RetryAfter: windowStart.Add(l.window).Sub(time.Now().UTC())}
	}

	return nil
}
