package oauth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
)

// newTestStore starts an in-memory miniredis server and returns a Store
// backed by it along with the miniredis handle for direct key inspection.
func newTestStore(t *testing.T) (*oauth.RedisStore, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return oauth.NewRedisStore(client), server
}

func TestRedisStorePutGetAuthorizationTransaction(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	transaction := oauth.AuthorizationTransaction{
		ID:                  "transaction-id",
		ClientID:            "https://client.example/metadata.json",
		RedirectURI:         "https://client.example/callback",
		Scopes:              []string{"phones:read", "messages:send"},
		State:               "state-value",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		ResponseType:        "code",
		CreatedAt:           time.Now().UTC().Truncate(time.Second),
	}

	require.NoError(t, store.PutAuthorizationTransaction(ctx, transaction, time.Minute))

	got, err := store.GetAuthorizationTransaction(ctx, "transaction-id")
	require.NoError(t, err)
	assert.Equal(t, transaction, got)

	// A transaction is read, not consumed: it must still be readable a
	// second time (needed across the Firebase-login redirect round trip).
	got2, err := store.GetAuthorizationTransaction(ctx, "transaction-id")
	require.NoError(t, err)
	assert.Equal(t, transaction, got2)
}

func TestRedisStoreGetAuthorizationTransactionNotFound(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.GetAuthorizationTransaction(context.Background(), "missing")
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

func TestRedisStoreConsumeAuthorizationCodeIsOneTimeUse(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	code := oauth.AuthorizationCode{
		Code:                "test-authorization-code",
		ClientID:            "https://client.example/metadata.json",
		RedirectURI:         "https://client.example/callback",
		Scopes:              []string{"phones:read"},
		UserID:              "firebase-uid",
		Email:               "user@example.com",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		CreatedAt:           time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, store.PutAuthorizationCode(ctx, code, time.Minute))

	first, err := store.ConsumeAuthorizationCode(ctx, "test-authorization-code")
	require.NoError(t, err)
	assert.Equal(t, code, first)

	_, err = store.ConsumeAuthorizationCode(ctx, "test-authorization-code")
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

func TestRedisStoreConsumeAuthorizationCodeNotFound(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.ConsumeAuthorizationCode(context.Background(), "never-issued")
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

func TestRedisStoreAuthorizationCodeExpires(t *testing.T) {
	store, server := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutAuthorizationCode(ctx, oauth.AuthorizationCode{Code: "expiring-code"}, time.Minute))
	server.FastForward(2 * time.Minute)

	_, err := store.ConsumeAuthorizationCode(ctx, "expiring-code")
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

func TestRedisStoreRotateRefreshTokenReplacesOldWithNew(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, oauth.RefreshGrant{
		Token:    "old-refresh-token",
		UserID:   "firebase-uid",
		ClientID: "client-id",
		Scopes:   []string{"messages:send"},
		FamilyID: "family-1",
	}, time.Hour))

	newGrant := oauth.RefreshGrant{
		Token:    "new-refresh-token",
		UserID:   "firebase-uid",
		ClientID: "client-id",
		Scopes:   []string{"messages:send"},
		FamilyID: "family-1",
	}
	require.NoError(t, store.RotateRefreshToken(ctx, "old-refresh-token", newGrant, time.Hour))

	// The old refresh token must no longer rotate (it has been consumed).
	err := store.RotateRefreshToken(ctx, "old-refresh-token", oauth.RefreshGrant{Token: "another-token"}, time.Hour)
	require.ErrorIs(t, err, oauth.ErrNotFound)

	// The new refresh token must itself now be rotatable, proving it was
	// actually written by the first rotation.
	require.NoError(t, store.RotateRefreshToken(ctx, "new-refresh-token", oauth.RefreshGrant{
		Token:    "newest-refresh-token",
		UserID:   "firebase-uid",
		ClientID: "client-id",
		FamilyID: "family-1",
	}, time.Hour))
}

func TestRedisStoreRotateRefreshTokenRejectsReplay(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, oauth.RefreshGrant{Token: "token-a"}, time.Hour))

	require.NoError(t, store.RotateRefreshToken(ctx, "token-a", oauth.RefreshGrant{Token: "token-b"}, time.Hour))

	// Replaying rotation with the already-consumed old token must fail even
	// though a (different, unrelated) new token value is supplied.
	err := store.RotateRefreshToken(ctx, "token-a", oauth.RefreshGrant{Token: "token-c"}, time.Hour)
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

func TestRedisStoreRotateRefreshTokenUnknownOldTokenFails(t *testing.T) {
	store, _ := newTestStore(t)

	err := store.RotateRefreshToken(context.Background(), "never-issued", oauth.RefreshGrant{Token: "new-token"}, time.Hour)
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

// TestRedisStoreGetRefreshTokenReadsWithoutConsuming asserts GetRefreshToken
// is a plain read: the token endpoint must be able to inspect a refresh
// grant's bound user/client/scopes/resource before rotating it, and the
// grant must still be present (and still rotatable) afterward.
func TestRedisStoreGetRefreshTokenReadsWithoutConsuming(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	grant := oauth.RefreshGrant{
		Token:     "refresh-token",
		UserID:    "firebase-uid",
		Email:     "user@example.com",
		ClientID:  "https://client.example/metadata.json",
		Scopes:    []string{"phones:read", "messages:send"},
		Resource:  "https://mcp.httpsms.com/mcp",
		FamilyID:  "family-1",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, store.PutRefreshToken(ctx, grant, time.Hour))

	got, err := store.GetRefreshToken(ctx, "refresh-token")
	require.NoError(t, err)
	assert.Equal(t, grant, got)

	// Reading again must still succeed (not consumed).
	got2, err := store.GetRefreshToken(ctx, "refresh-token")
	require.NoError(t, err)
	assert.Equal(t, grant, got2)

	// The grant must still be rotatable, proving Get did not delete it.
	require.NoError(t, store.RotateRefreshToken(ctx, "refresh-token", oauth.RefreshGrant{Token: "rotated-token"}, time.Hour))
}

func TestRedisStoreGetRefreshTokenNotFound(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.GetRefreshToken(context.Background(), "never-issued")
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

func TestRedisStorePutGetDynamicClient(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	client := oauth.Client{
		ID:                      "dcr-client-id",
		Name:                    "Test Client",
		RedirectURIs:            []string{"https://client.example/callback"},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	}
	require.NoError(t, store.PutDynamicClient(ctx, client, 24*time.Hour))

	got, err := store.GetDynamicClient(ctx, "dcr-client-id")
	require.NoError(t, err)
	assert.Equal(t, client, got)
}

func TestRedisStoreGetDynamicClientNotFound(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.GetDynamicClient(context.Background(), "missing-client")
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

func TestRedisStoreConsumeConfirmationIsOneTimeUse(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	confirmation := oauth.Confirmation{
		Handle:    "confirmation-handle",
		UserID:    "firebase-uid",
		ClientID:  "client-id",
		Operation: "rotate_user_api_key",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, store.PutConfirmation(ctx, confirmation, 5*time.Minute))

	first, err := store.ConsumeConfirmation(ctx, "confirmation-handle")
	require.NoError(t, err)
	assert.Equal(t, confirmation, first)

	_, err = store.ConsumeConfirmation(ctx, "confirmation-handle")
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

func TestRedisStoreConsumeConfirmationNotFound(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.ConsumeConfirmation(context.Background(), "never-issued")
	require.ErrorIs(t, err, oauth.ErrNotFound)
}

// TestRedisStoreKeysAreNamespacedAndHashed asserts that every stored key
// uses the documented namespace prefix and never contains the raw public
// secret value as a substring -- only its SHA-256 hash may appear.
func TestRedisStoreKeysAreNamespacedAndHashed(t *testing.T) {
	store, server := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutAuthorizationTransaction(ctx, oauth.AuthorizationTransaction{ID: "super-secret-transaction-id"}, time.Minute))
	require.NoError(t, store.PutAuthorizationCode(ctx, oauth.AuthorizationCode{Code: "super-secret-code"}, time.Minute))
	require.NoError(t, store.PutRefreshToken(ctx, oauth.RefreshGrant{Token: "super-secret-refresh-token"}, time.Hour))
	require.NoError(t, store.PutDynamicClient(ctx, oauth.Client{ID: "super-secret-client-id"}, time.Hour))
	require.NoError(t, store.PutConfirmation(ctx, oauth.Confirmation{Handle: "super-secret-handle"}, time.Minute))

	keys := server.Keys()
	require.Len(t, keys, 5)

	expectedPrefixes := []string{
		"httpsms:mcp:oauth:transaction:",
		"httpsms:mcp:oauth:code:",
		"httpsms:mcp:oauth:refresh:",
		"httpsms:mcp:oauth:client:",
		"httpsms:mcp:confirmation:",
	}

	for _, key := range keys {
		hasPrefix := false
		for _, prefix := range expectedPrefixes {
			if strings.HasPrefix(key, prefix) {
				hasPrefix = true
				// The remainder must be a 64-character hex-encoded SHA-256
				// digest, not the raw secret.
				remainder := strings.TrimPrefix(key, prefix)
				assert.Len(t, remainder, 64)
				break
			}
		}
		assert.True(t, hasPrefix, "key %q must use one of the documented namespace prefixes", key)

		assert.NotContains(t, key, "super-secret", "Redis key %q must not contain the raw secret value", key)
	}
}

func TestRedisStorePutRejectsNonPositiveTTL(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	err := store.PutAuthorizationCode(ctx, oauth.AuthorizationCode{Code: "code"}, 0)
	require.Error(t, err)

	err = store.PutRefreshToken(ctx, oauth.RefreshGrant{Token: "token"}, -time.Second)
	require.Error(t, err)
}

func TestRedisStoreRotateRefreshTokenRejectsNonPositiveTTL(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, oauth.RefreshGrant{Token: "token"}, time.Hour))

	err := store.RotateRefreshToken(ctx, "token", oauth.RefreshGrant{Token: "new-token"}, 0)
	require.Error(t, err)
}

// TestNewRedisStorePanicsOnClusterClient asserts NewRedisStore fails fast
// for a *redis.ClusterClient: RotateRefreshToken's Lua script touches two
// keys (the old and new refresh-token hashes) that this service's approved
// key format gives no cross-key hash-slot guarantee for, so the script
// would fail against a real cluster with a CROSSSLOT error. This service
// is intentionally constrained to a standalone Redis deployment
// (redis.NewClient); the key format itself must not change to work around
// this, per the accepted design constraint.
func TestNewRedisStorePanicsOnClusterClient(t *testing.T) {
	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{Addrs: []string{"127.0.0.1:0"}})
	t.Cleanup(func() { _ = clusterClient.Close() })

	assert.Panics(t, func() { oauth.NewRedisStore(clusterClient) })
}

// TestNewRedisStorePanicsOnRingClient is the Ring-client counterpart of
// TestNewRedisStorePanicsOnClusterClient: a Ring client also shards keys
// across independent Redis nodes by hash, so the same cross-slot rotation
// script cannot safely run against it either.
func TestNewRedisStorePanicsOnRingClient(t *testing.T) {
	ringClient := redis.NewRing(&redis.RingOptions{Addrs: map[string]string{"shard0": "127.0.0.1:0"}})
	t.Cleanup(func() { _ = ringClient.Close() })

	assert.Panics(t, func() { oauth.NewRedisStore(ringClient) })
}

// TestNewRedisStoreAcceptsStandaloneClient documents the supported,
// required configuration: a plain redis.NewClient must not panic.
func TestNewRedisStoreAcceptsStandaloneClient(t *testing.T) {
	standaloneClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() { _ = standaloneClient.Close() })

	assert.NotPanics(t, func() { oauth.NewRedisStore(standaloneClient) })
}
