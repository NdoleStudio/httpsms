// Package server assembles the httpSMS MCP service's HTTP surface: OAuth
// discovery/authorization/token endpoints, the stateless MCP Streamable
// HTTP handler, and the middleware chain (request ID, panic recovery,
// secure headers, tracing, redacted logging, bearer authentication, and
// per-user/per-tool rate limiting) around them.
package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrRateLimited is returned (wrapped) by ToolRateLimiter.Allow when userID
// has already exhausted its budget for tool in the current window. Callers
// should use errors.Is against this sentinel rather than matching error
// strings.
var ErrRateLimited = errors.New("server: rate limit exceeded")

// Limits configures the per-user rate-limit budgets ToolRateLimiter
// enforces for each MCP tool bucket, mirroring config.Config's
// ReadToolsPerMinute, SendToolsPerMinute, KeyCreatesPerHour, and
// KeyRotationsPerHour fields.
type Limits struct {
	// ReadPerMinute bounds list_phones, list_message_threads,
	// list_thread_messages, and list_incoming_messages, per user, per
	// rolling one-minute window.
	ReadPerMinute int

	// SendPerMinute bounds send_sms, per user, per rolling one-minute
	// window.
	SendPerMinute int

	// KeyCreatesPerHour bounds create_phone_api_key, per user, per rolling
	// one-hour window.
	KeyCreatesPerHour int

	// KeyRotationsPerHour bounds rotate_user_api_key, per user, per
	// rolling one-hour window.
	KeyRotationsPerHour int
}

// validate returns an error naming the first non-positive budget in l. A
// zero or negative budget is never a valid configuration: Allow treats it
// as "this tool is not rate limited at all", so a mis-wired or forgotten
// budget would silently remove the limit on exactly the tools (sending
// SMS, minting API keys, rotating the primary key) that most need one.
func (l Limits) validate() error {
	switch {
	case l.ReadPerMinute <= 0:
		return errors.New("server: Limits.ReadPerMinute must be positive")
	case l.SendPerMinute <= 0:
		return errors.New("server: Limits.SendPerMinute must be positive")
	case l.KeyCreatesPerHour <= 0:
		return errors.New("server: Limits.KeyCreatesPerHour must be positive")
	case l.KeyRotationsPerHour <= 0:
		return errors.New("server: Limits.KeyRotationsPerHour must be positive")
	default:
		return nil
	}
}

// bucket is one rate-limit budget: how many calls tool may receive from a
// single user within window.
type bucket struct {
	limit  int
	window time.Duration
}

// buckets maps every rate-limited MCP tool name to the budget that bounds
// it. A tool absent from this map (there are none today) is never rate
// limited.
func (l Limits) buckets() map[string]bucket {
	return map[string]bucket{
		"list_phones":            {limit: l.ReadPerMinute, window: time.Minute},
		"list_message_threads":   {limit: l.ReadPerMinute, window: time.Minute},
		"list_thread_messages":   {limit: l.ReadPerMinute, window: time.Minute},
		"list_incoming_messages": {limit: l.ReadPerMinute, window: time.Minute},
		"send_sms":               {limit: l.SendPerMinute, window: time.Minute},
		"create_phone_api_key":   {limit: l.KeyCreatesPerHour, window: time.Hour},
		"rotate_user_api_key":    {limit: l.KeyRotationsPerHour, window: time.Hour},
	}
}

// keyPrefixRateLimit namespaces every rate-limit counter key.
const keyPrefixRateLimit = "httpsms:mcp:ratelimit:"

// rateLimitScript atomically increments the counter for a rate-limit
// window and, only on the first increment (count == 1), sets its expiry.
// A Lua script run through EVAL is the only way to make "increment" and
// "set the window's expiry" a single indivisible operation: running INCR
// and EXPIRE as two separate commands (even inside a MULTI/EXEC
// transaction, which cannot branch) would leave a window without a TTL if
// the process crashed between them, or would reset another caller's
// window if two requests raced to set it.
var rateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

// RateLimitError reports that a caller has exceeded its rate-limit budget.
// It wraps ErrRateLimited (so errors.Is(err, ErrRateLimited) reports true)
// while also carrying the RetryAfter duration a client should wait before
// trying again.
type RateLimitError struct {
	// Tool is the MCP tool name the caller was rate limited on.
	Tool string
	// RetryAfter is how long the caller should wait before its next
	// attempt to this same tool is likely to succeed.
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("server: rate limit exceeded for tool %q, retry after %s", e.Tool, e.RetryAfter)
}

// Unwrap allows errors.Is(err, ErrRateLimited) to succeed for a
// *RateLimitError.
func (e *RateLimitError) Unwrap() error { return ErrRateLimited }

// ToolRateLimiter enforces the per-user, per-tool Redis-backed rate limits
// configured by Limits before every MCP tool call executes.
//
// It fails closed: a Redis error from Allow is returned as its own error
// (never ErrRateLimited, and never silently treated as "the call is
// allowed"). A caller must treat any non-nil error from Allow as "do not
// execute the tool call".
type ToolRateLimiter struct {
	client redis.UniversalClient
	limits Limits
}

// NewToolRateLimiter returns a ToolRateLimiter enforcing limits, using
// client to store per-user/per-tool counters. client must be a standalone
// Redis client (never a cluster or ring client), matching every other
// Redis-backed component in this service.
func NewToolRateLimiter(client redis.UniversalClient, limits Limits) *ToolRateLimiter {
	return &ToolRateLimiter{client: client, limits: limits}
}

// Allow reports whether userID may call tool right now, atomically
// incrementing its counter for the current window as a side effect. It
// returns a *RateLimitError (unwrapping to ErrRateLimited) once userID has
// already made bucket.limit calls to tool within the current window.
//
// Tools with no configured bucket, or a non-positive limit, are never rate
// limited: Allow returns nil immediately without touching Redis.
//
// A Redis failure is returned as its own error (fmt.Errorf-wrapped, never
// ErrRateLimited): this rate limiter fails closed rather than allowing a
// call through when it cannot verify the caller's budget.
func (l *ToolRateLimiter) Allow(ctx context.Context, userID string, tool string) error {
	b, limited := l.limits.buckets()[tool]
	if !limited || b.limit <= 0 {
		return nil
	}

	windowStart := time.Now().UTC().Truncate(b.window)
	key := rateLimitKey(userID, tool, windowStart)

	count, err := rateLimitScript.Run(ctx, l.client, []string{key}, b.window.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("server: cannot check rate limit for tool %q: %w", tool, err)
	}

	if count > b.limit {
		return &RateLimitError{Tool: tool, RetryAfter: windowStart.Add(b.window).Sub(time.Now().UTC())}
	}

	return nil
}

// rateLimitKey returns the Redis key for userID's counter for tool during
// the window starting at windowStart. The key names userID only as the
// hex-encoded SHA-256 hash of the raw Firebase UID, never the raw value
// itself, matching every other Redis key namespace in this service.
func rateLimitKey(userID string, tool string, windowStart time.Time) string {
	sum := sha256.Sum256([]byte(userID))
	return fmt.Sprintf("%s%s:%s:%d", keyPrefixRateLimit, hex.EncodeToString(sum[:]), tool, windowStart.Unix())
}
