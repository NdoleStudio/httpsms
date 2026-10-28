package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validRegistrationBody is a well-formed RFC 7591 registration request.
const validRegistrationBody = `{
	"client_name": "Rate Limit Client",
	"redirect_uris": ["https://client.example/callback"],
	"grant_types": ["authorization_code"],
	"response_types": ["code"],
	"token_endpoint_auth_method": "none"
}`

// newLimiterTestRedis starts an in-process miniredis instance and returns
// a standalone client for it.
func newLimiterTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return server, client
}

// stubRegistrationLimiter is a RegistrationLimiter that returns a fixed
// error (or nil) and counts how often it was consulted.
type stubRegistrationLimiter struct {
	err   error
	calls int
}

func (l *stubRegistrationLimiter) Allow(context.Context) error {
	l.calls++
	return l.err
}

func TestRedisRegistrationLimiterAllowsUpToTheBudget(t *testing.T) {
	_, client := newLimiterTestRedis(t)
	limiter := NewRedisRegistrationLimiter(client, 3)
	ctx := context.Background()

	for i := range 3 {
		require.NoError(t, limiter.Allow(ctx), "registration %d must be allowed", i+1)
	}

	err := limiter.Allow(ctx)
	require.ErrorIs(t, err, ErrRegistrationRateLimited)

	var rateLimited *RegistrationRateLimitError
	require.True(t, errors.As(err, &rateLimited))
	assert.Positive(t, rateLimited.RetryAfter)
	assert.LessOrEqual(t, rateLimited.RetryAfter, time.Minute)
}

// TestRedisRegistrationLimiterIsGlobal asserts the budget is shared by
// every caller: two limiter values over the same Redis spend the same
// counter, which is what makes the limit meaningful for an
// unauthenticated endpoint where no trustworthy caller identity exists.
func TestRedisRegistrationLimiterIsGlobal(t *testing.T) {
	_, client := newLimiterTestRedis(t)
	ctx := context.Background()

	first := NewRedisRegistrationLimiter(client, 2)
	second := NewRedisRegistrationLimiter(client, 2)

	require.NoError(t, first.Allow(ctx))
	require.NoError(t, second.Allow(ctx))
	require.ErrorIs(t, first.Allow(ctx), ErrRegistrationRateLimited)
	require.ErrorIs(t, second.Allow(ctx), ErrRegistrationRateLimited)
}

// TestRedisRegistrationLimiterCounterIsBounded asserts the window counter
// always carries a TTL, so a burst can never leave a permanently
// exhausted budget behind.
func TestRedisRegistrationLimiterCounterIsBounded(t *testing.T) {
	server, client := newLimiterTestRedis(t)
	limiter := NewRedisRegistrationLimiter(client, 1)
	ctx := context.Background()

	require.NoError(t, limiter.Allow(ctx))

	keys := server.Keys()
	require.Len(t, keys, 1)
	assert.True(t, strings.HasPrefix(keys[0], "httpsms:mcp:oauth:register:ratelimit:"), "unexpected key %q", keys[0])
	assert.Positive(t, server.TTL(keys[0]))
	assert.LessOrEqual(t, server.TTL(keys[0]), time.Minute)

	require.ErrorIs(t, limiter.Allow(ctx), ErrRegistrationRateLimited)

	server.FastForward(time.Minute + time.Second)
	require.NoError(t, limiter.Allow(ctx), "a fresh window must restore the budget")
}

// TestRedisRegistrationLimiterFailsClosed asserts an unreachable Redis is
// reported as an error (never as "allowed"), so a registration is refused
// rather than persisted when the budget cannot be verified.
func TestRedisRegistrationLimiterFailsClosed(t *testing.T) {
	server, client := newLimiterTestRedis(t)
	limiter := NewRedisRegistrationLimiter(client, 10)

	server.Close()

	err := limiter.Allow(context.Background())
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrRegistrationRateLimited)
}

// TestNewRedisRegistrationLimiterRejectsNonPositiveBudget asserts a
// mis-wired budget falls back to the production default rather than
// silently disabling the limit.
func TestNewRedisRegistrationLimiterRejectsNonPositiveBudget(t *testing.T) {
	_, client := newLimiterTestRedis(t)

	assert.Equal(t, DefaultRegistrationsPerMinute, NewRedisRegistrationLimiter(client, 0).limit)
	assert.Equal(t, DefaultRegistrationsPerMinute, NewRedisRegistrationLimiter(client, -5).limit)
	assert.Equal(t, 7, NewRedisRegistrationLimiter(client, 7).limit)
}

// TestNewRedisRegistrationLimiterRejectsClusterClient mirrors
// NewRedisStore's standalone-Redis constraint.
func TestNewRedisRegistrationLimiterRejectsClusterClient(t *testing.T) {
	assert.Panics(t, func() {
		NewRedisRegistrationLimiter(redis.NewClusterClient(&redis.ClusterOptions{Addrs: []string{"127.0.0.1:6379"}}), 10)
	})
	assert.Panics(t, func() {
		NewRedisRegistrationLimiter(redis.NewRing(&redis.RingOptions{Addrs: map[string]string{"a": "127.0.0.1:6379"}}), 10)
	})
}

// TestRegistrationHandlerEnforcesRateLimit asserts an exhausted budget is
// refused with an OAuth-shaped 429 carrying Retry-After, and -- crucially
// -- that nothing is persisted for the refused request.
func TestRegistrationHandlerEnforcesRateLimit(t *testing.T) {
	store := newClientsTestStore(t)
	_, client := newLimiterTestRedis(t)
	handler := NewRegistrationHandler(store, NewRedisRegistrationLimiter(client, 2))

	var created []string
	for range 2 {
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(validRegistrationBody)))
		require.Equal(t, http.StatusCreated, rec.Code)

		var registered Client
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &registered))
		created = append(created, registered.ID)
	}

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(validRegistrationBody)))

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

	retryAfter, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	require.NoError(t, err, "Retry-After must be an integer number of seconds")
	assert.GreaterOrEqual(t, retryAfter, 1)
	assert.LessOrEqual(t, retryAfter, 60)

	var body registrationError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "temporarily_unavailable", body.Error)
	assert.NotEmpty(t, body.ErrorDescription)

	// Exactly the two accepted registrations exist; the refused one
	// persisted nothing.
	for _, id := range created {
		_, err := store.GetDynamicClient(context.Background(), id)
		require.NoError(t, err)
	}
}

// TestRegistrationHandlerFailsClosedWhenLimiterErrors asserts a limiter
// failure refuses the registration with a 503 instead of storing a client
// it could not charge.
func TestRegistrationHandlerFailsClosedWhenLimiterErrors(t *testing.T) {
	miniredisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniredisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	handler := NewRegistrationHandler(NewRedisStore(redisClient), &stubRegistrationLimiter{err: errors.New("redis is down")})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(validRegistrationBody)))

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Empty(t, rec.Header().Get("Retry-After"))

	var body registrationError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "temporarily_unavailable", body.Error)
	assert.NotContains(t, rec.Body.String(), "redis is down", "the underlying failure must never leak to the client")

	assert.Empty(t, miniredisServer.Keys(), "a refused registration must never persist a client")
}

// TestRegistrationHandlerMalformedRequestsPersistNothing asserts every
// rejected registration shape -- unreadable, oversized, malformed JSON, or
// invalid metadata -- leaves the store empty.
func TestRegistrationHandlerMalformedRequestsPersistNothing(t *testing.T) {
	bodies := map[string]string{
		"not json":              "not json at all",
		"empty object":          `{}`,
		"missing redirect_uris": `{"client_name":"No Redirects","grant_types":["authorization_code"],"response_types":["code"],"token_endpoint_auth_method":"none"}`,
		"http redirect_uri":     `{"client_name":"Insecure","redirect_uris":["http://client.example/callback"],"grant_types":["authorization_code"],"response_types":["code"],"token_endpoint_auth_method":"none"}`,
		"unsupported grant":     `{"client_name":"Implicit","redirect_uris":["https://client.example/callback"],"grant_types":["implicit"],"response_types":["code"],"token_endpoint_auth_method":"none"}`,
		"client secret auth":    `{"client_name":"Confidential","redirect_uris":["https://client.example/callback"],"grant_types":["authorization_code"],"response_types":["code"],"token_endpoint_auth_method":"client_secret_basic"}`,
		"oversized":             `{"client_name":"` + strings.Repeat("a", maxClientMetadataBytes+1) + `"}`,
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			miniredisServer := miniredis.RunT(t)
			redisClient := redis.NewClient(&redis.Options{Addr: miniredisServer.Addr()})
			t.Cleanup(func() { _ = redisClient.Close() })

			handler := NewRegistrationHandler(NewRedisStore(redisClient), newClientsTestLimiter(t))

			rec := httptest.NewRecorder()
			handler(rec, httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body)))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Empty(t, miniredisServer.Keys(), "a rejected registration must never persist a client")
		})
	}
}

// TestRegistrationHandlerChecksLimitBeforeReadingBody asserts the budget
// is charged before the request body is parsed, so a flood of malformed
// registrations cannot bypass the limit.
func TestRegistrationHandlerChecksLimitBeforeReadingBody(t *testing.T) {
	limiter := &stubRegistrationLimiter{}
	handler := NewRegistrationHandler(newClientsTestStore(t), limiter)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader("not json")))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 1, limiter.calls, "a malformed registration must still be charged against the budget")
}

// TestRegistrationHandlerDoesNotChargeNonPOST asserts a method-mismatch
// probe (including a CORS preflight) never spends registration budget.
func TestRegistrationHandlerDoesNotChargeNonPOST(t *testing.T) {
	limiter := &stubRegistrationLimiter{}
	handler := NewRegistrationHandler(newClientsTestStore(t), limiter)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/oauth/register", nil))

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Zero(t, limiter.calls)
}

func TestNewRegistrationHandlerRequiresStoreAndLimiter(t *testing.T) {
	assert.Panics(t, func() { NewRegistrationHandler(nil, &stubRegistrationLimiter{}) })
	assert.Panics(t, func() { NewRegistrationHandler(newClientsTestStore(t), nil) })
}
