package oauth

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCacheOnlyResolver builds a ClientResolver with nothing but its client
// metadata cache initialized, which is all the cache-bounding behavior under
// test in this file touches.
func newCacheOnlyResolver() *ClientResolver {
	return &ClientResolver{cache: make(map[string]cachedClient)}
}

func cacheTestClientID(index int) string {
	return fmt.Sprintf("https://client-%04d.example/metadata.json", index)
}

func cacheTestClient(clientID string) Client {
	return validTestClient(clientID, []string{"https://client.example/callback"})
}

// TestClientResolverCachePurgesExpiredEntriesWhenFull proves a full cache
// first reclaims space from entries whose 15-minute TTL already elapsed,
// rather than evicting entries that are still valid.
func TestClientResolverCachePurgesExpiredEntriesWhenFull(t *testing.T) {
	resolver := newCacheOnlyResolver()
	now := time.Now()

	for i := 0; i < cimdCacheMaxEntries; i++ {
		clientID := cacheTestClientID(i)
		expiresAt := now.Add(cimdCacheTTL)
		if i%2 == 0 {
			expiresAt = now.Add(-time.Minute)
		}
		resolver.cache[clientID] = cachedClient{client: cacheTestClient(clientID), expiresAt: expiresAt}
	}
	require.Len(t, resolver.cache, cimdCacheMaxEntries)

	newClientID := "https://new-client.example/metadata.json"
	resolver.cacheClient(newClientID, cacheTestClient(newClientID))

	assert.Len(t, resolver.cache, cimdCacheMaxEntries/2+1)

	_, expiredStillCached := resolver.cache[cacheTestClientID(0)]
	assert.False(t, expiredStillCached, "expired entries must be purged")

	_, unexpiredEvicted := resolver.cache[cacheTestClientID(1)]
	assert.True(t, unexpiredEvicted, "unexpired entries must survive an expired-entry purge")

	client, ok := resolver.cached(newClientID)
	require.True(t, ok)
	assert.Equal(t, newClientID, client.ID)
}

// TestClientResolverCacheEvictsOldestWhenNothingIsExpired proves the ceiling
// still holds when no entry can be purged: the entry closest to expiring
// (the one cached longest ago, since every entry shares one TTL) is evicted.
func TestClientResolverCacheEvictsOldestWhenNothingIsExpired(t *testing.T) {
	resolver := newCacheOnlyResolver()
	base := time.Now().Add(cimdCacheTTL)

	for i := 0; i < cimdCacheMaxEntries; i++ {
		clientID := cacheTestClientID(i)
		resolver.cache[clientID] = cachedClient{
			client:    cacheTestClient(clientID),
			expiresAt: base.Add(time.Duration(i) * time.Second),
		}
	}

	newClientID := "https://new-client.example/metadata.json"
	resolver.cacheClient(newClientID, cacheTestClient(newClientID))

	assert.Len(t, resolver.cache, cimdCacheMaxEntries)

	_, oldestStillCached := resolver.cache[cacheTestClientID(0)]
	assert.False(t, oldestStillCached, "the entry closest to expiring must be evicted")

	_, secondOldestEvicted := resolver.cache[cacheTestClientID(1)]
	assert.True(t, secondOldestEvicted, "only one entry may be evicted per insertion")

	_, ok := resolver.cached(newClientID)
	assert.True(t, ok)
}

// TestClientResolverCacheStaysBoundedAcrossManyDistinctClients proves that
// any HTTPS URL serving a valid document being a cacheable client_id can no
// longer grow the cache without limit.
func TestClientResolverCacheStaysBoundedAcrossManyDistinctClients(t *testing.T) {
	resolver := newCacheOnlyResolver()

	total := cimdCacheMaxEntries * 3
	for i := 0; i < total; i++ {
		clientID := cacheTestClientID(i)
		resolver.cacheClient(clientID, cacheTestClient(clientID))
		require.LessOrEqual(t, len(resolver.cache), cimdCacheMaxEntries)
	}

	assert.Len(t, resolver.cache, cimdCacheMaxEntries)

	_, ok := resolver.cached(cacheTestClientID(total - 1))
	assert.True(t, ok, "the most recently resolved client must stay cached")
}

// TestClientResolverCacheRefreshOfExistingClientEvictsNothing proves that
// re-caching a client_id that is already cached only replaces its entry.
func TestClientResolverCacheRefreshOfExistingClientEvictsNothing(t *testing.T) {
	resolver := newCacheOnlyResolver()

	for i := 0; i < cimdCacheMaxEntries; i++ {
		clientID := cacheTestClientID(i)
		resolver.cacheClient(clientID, cacheTestClient(clientID))
	}
	require.Len(t, resolver.cache, cimdCacheMaxEntries)

	existingID := cacheTestClientID(0)
	resolver.cacheClient(existingID, cacheTestClient(existingID))

	assert.Len(t, resolver.cache, cimdCacheMaxEntries)
	_, ok := resolver.cached(existingID)
	assert.True(t, ok)
}

// TestClientResolverCachedIgnoresExpiredEntry proves a cached document is
// never served past its 15-minute TTL.
func TestClientResolverCachedIgnoresExpiredEntry(t *testing.T) {
	resolver := newCacheOnlyResolver()

	clientID := cacheTestClientID(0)
	resolver.cache[clientID] = cachedClient{
		client:    cacheTestClient(clientID),
		expiresAt: time.Now().Add(-time.Second),
	}

	_, ok := resolver.cached(clientID)
	assert.False(t, ok)
}

// TestClientResolverCacheIsConcurrencySafe exercises the cache from many
// goroutines so the race detector can prove every path stays under cacheMu.
func TestClientResolverCacheIsConcurrencySafe(t *testing.T) {
	resolver := newCacheOnlyResolver()

	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				clientID := cacheTestClientID(worker*1000 + i)
				resolver.cacheClient(clientID, cacheTestClient(clientID))
				resolver.cached(clientID)
			}
		}(worker)
	}
	wg.Wait()

	assert.LessOrEqual(t, len(resolver.cache), cimdCacheMaxEntries)
}
