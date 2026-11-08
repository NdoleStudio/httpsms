package oauth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
)

// refreshGrant builds a RefreshGrant for token in family.
func refreshGrant(token, family string) oauth.RefreshGrant {
	return oauth.RefreshGrant{
		Token:    token,
		UserID:   "firebase-uid",
		Email:    "user@example.com",
		ClientID: "https://client.example/metadata.json",
		Scopes:   []string{"messages:read"},
		Resource: "https://mcp.httpsms.com/mcp",
		FamilyID: family,
	}
}

// TestRedisStorePutRefreshTokenPublishesActiveFamilyMember asserts initial
// issuance atomically stores both the token and the family pointer, so a
// freshly issued token is revocable from the moment it exists.
func TestRedisStorePutRefreshTokenPublishesActiveFamilyMember(t *testing.T) {
	store, server := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("token-1", "family-1"), time.Hour))

	keys := server.Keys()
	require.Len(t, keys, 2)

	var refreshKeys, familyKeys int
	for _, key := range keys {
		switch {
		case strings.HasPrefix(key, "httpsms:mcp:oauth:refresh-family:"):
			familyKeys++
		case strings.HasPrefix(key, "httpsms:mcp:oauth:refresh:"):
			refreshKeys++
		}
		assert.Positive(t, server.TTL(key), "key %q must carry the refresh TTL", key)
	}
	assert.Equal(t, 1, refreshKeys)
	assert.Equal(t, 1, familyKeys)
}

func TestRedisStorePutRefreshTokenRequiresFamilyID(t *testing.T) {
	store, _ := newTestStore(t)

	err := store.PutRefreshToken(context.Background(), oauth.RefreshGrant{Token: "token"}, time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "family ID")
}

func TestRedisStoreRotateRefreshTokenRequiresFamilyID(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("token-1", "family-1"), time.Hour))

	err := store.RotateRefreshToken(ctx, "token-1", oauth.RefreshGrant{Token: "token-2"}, time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "family ID")
}

// TestRedisStoreDetectRefreshTokenReuseRevokesActiveToken asserts the core
// family reuse-detection contract: replaying a consumed token revokes the
// replacement the legitimate client is currently holding.
func TestRedisStoreDetectRefreshTokenReuseRevokesActiveToken(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("token-1", "family-1"), time.Hour))
	require.NoError(t, store.RotateRefreshToken(ctx, "token-1", refreshGrant("token-2", "family-1"), time.Hour))

	// The replacement is live before the replay.
	_, err := store.GetRefreshToken(ctx, "token-2")
	require.NoError(t, err)

	// Replaying the consumed token is reported as reuse, not as a plain
	// "not found".
	require.ErrorIs(t, store.DetectRefreshTokenReuse(ctx, "token-1"), oauth.ErrRefreshTokenReuse)

	// ...and the replacement has been revoked.
	_, err = store.GetRefreshToken(ctx, "token-2")
	require.ErrorIs(t, err, oauth.ErrNotFound)
	require.ErrorIs(t, store.RotateRefreshToken(ctx, "token-2", refreshGrant("token-3", "family-1"), time.Hour), oauth.ErrNotFound)
}

// TestRedisStoreDetectRefreshTokenReuseAfterManyRotationsRevokesNewest
// asserts a token replayed long after the lineage has moved on still
// revokes whatever is active *now*, not the token that directly succeeded
// it.
func TestRedisStoreDetectRefreshTokenReuseAfterManyRotationsRevokesNewest(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("token-1", "family-1"), time.Hour))
	require.NoError(t, store.RotateRefreshToken(ctx, "token-1", refreshGrant("token-2", "family-1"), time.Hour))
	require.NoError(t, store.RotateRefreshToken(ctx, "token-2", refreshGrant("token-3", "family-1"), time.Hour))
	require.NoError(t, store.RotateRefreshToken(ctx, "token-3", refreshGrant("token-4", "family-1"), time.Hour))

	_, err := store.GetRefreshToken(ctx, "token-4")
	require.NoError(t, err)

	// Replay the very first token of the lineage.
	require.ErrorIs(t, store.DetectRefreshTokenReuse(ctx, "token-1"), oauth.ErrRefreshTokenReuse)

	_, err = store.GetRefreshToken(ctx, "token-4")
	require.ErrorIs(t, err, oauth.ErrNotFound, "the newest active token must be revoked")
}

// TestRedisStoreDetectRefreshTokenReuseLeavesOtherFamiliesAlone asserts
// revocation is scoped to the compromised lineage: a second, unrelated
// grant for the same user keeps working.
func TestRedisStoreDetectRefreshTokenReuseLeavesOtherFamiliesAlone(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("a-1", "family-a"), time.Hour))
	require.NoError(t, store.RotateRefreshToken(ctx, "a-1", refreshGrant("a-2", "family-a"), time.Hour))
	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("b-1", "family-b"), time.Hour))

	require.ErrorIs(t, store.DetectRefreshTokenReuse(ctx, "a-1"), oauth.ErrRefreshTokenReuse)

	_, err := store.GetRefreshToken(ctx, "a-2")
	require.ErrorIs(t, err, oauth.ErrNotFound)

	got, err := store.GetRefreshToken(ctx, "b-1")
	require.NoError(t, err)
	assert.Equal(t, "family-b", got.FamilyID)
}

// TestRedisStoreDetectRefreshTokenReuseIsIdempotent asserts a second
// replay of the same token is still reported as reuse (the tombstone is
// retained), and does not error even though nothing is left to revoke.
func TestRedisStoreDetectRefreshTokenReuseIsIdempotent(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("token-1", "family-1"), time.Hour))
	require.NoError(t, store.RotateRefreshToken(ctx, "token-1", refreshGrant("token-2", "family-1"), time.Hour))

	require.ErrorIs(t, store.DetectRefreshTokenReuse(ctx, "token-1"), oauth.ErrRefreshTokenReuse)
	require.ErrorIs(t, store.DetectRefreshTokenReuse(ctx, "token-1"), oauth.ErrRefreshTokenReuse)
}

// TestRedisStoreDetectRefreshTokenReuseUnknownTokenIsNotReuse asserts a
// never-issued (or long-expired) token is an ordinary invalid grant, not a
// reuse event that would revoke something.
func TestRedisStoreDetectRefreshTokenReuseUnknownTokenIsNotReuse(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("token-1", "family-1"), time.Hour))

	require.ErrorIs(t, store.DetectRefreshTokenReuse(ctx, "never-issued"), oauth.ErrNotFound)

	// The live token is untouched.
	_, err := store.GetRefreshToken(ctx, "token-1")
	require.NoError(t, err)
}

// TestRedisStoreDetectRefreshTokenReuseIgnoresLiveToken asserts a token
// that has never been rotated has no tombstone, so presenting it (for
// example after its own expiry) is not treated as reuse.
func TestRedisStoreDetectRefreshTokenReuseIgnoresLiveToken(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("token-1", "family-1"), time.Hour))

	require.ErrorIs(t, store.DetectRefreshTokenReuse(ctx, "token-1"), oauth.ErrNotFound)

	_, err := store.GetRefreshToken(ctx, "token-1")
	require.NoError(t, err)
}

// TestRedisStoreRefreshTombstoneOutlivesNothingForever asserts the
// tombstone and family pointer are both TTL-bounded by the refresh TTL, so
// reuse detection state never accumulates without limit.
func TestRedisStoreRefreshTombstoneIsTTLBounded(t *testing.T) {
	store, server := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("token-1", "family-1"), time.Hour))
	require.NoError(t, store.RotateRefreshToken(ctx, "token-1", refreshGrant("token-2", "family-1"), time.Hour))

	for _, key := range server.Keys() {
		assert.Positive(t, server.TTL(key), "key %q must carry a TTL", key)
		assert.LessOrEqual(t, server.TTL(key), time.Hour)
	}

	server.FastForward(2 * time.Hour)
	assert.Empty(t, server.Keys(), "every refresh-token record must expire with the refresh TTL")

	require.ErrorIs(t, store.DetectRefreshTokenReuse(ctx, "token-1"), oauth.ErrNotFound)
}

// TestRedisStoreRefreshTombstoneStoresNoPlaintextToken asserts neither the
// tombstone's key nor its value carries a raw refresh token: the tombstone
// stores the family pointer's hash-derived key name.
func TestRedisStoreRefreshTombstoneStoresNoPlaintextToken(t *testing.T) {
	store, server := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.PutRefreshToken(ctx, refreshGrant("plaintext-token-one", "plaintext-family"), time.Hour))
	require.NoError(t, store.RotateRefreshToken(ctx, "plaintext-token-one", refreshGrant("plaintext-token-two", "plaintext-family"), time.Hour))

	dump := server.Dump()
	assert.NotContains(t, dump, "plaintext-token-one")
	assert.NotContains(t, dump, "plaintext-token-two")

	for _, key := range server.Keys() {
		assert.NotContains(t, key, "plaintext-token")
	}
}
