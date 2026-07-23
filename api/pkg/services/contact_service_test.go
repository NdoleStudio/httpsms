package services

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// --- fake cache -------------------------------------------------------------

type fakeCache struct {
	mu     sync.Mutex
	store  map[string]string
	getErr error
	setErr error
	sets   []cacheSetCall
	gets   int
}

type cacheSetCall struct {
	key   string
	value string
	ttl   time.Duration
}

func newFakeCache() *fakeCache { return &fakeCache{store: map[string]string{}} }

func (c *fakeCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.gets++
	if c.getErr != nil {
		return "", c.getErr
	}
	value, ok := c.store[key]
	if !ok {
		return "", stacktrace.NewErrorf("no item found in cache with key [%s]", key)
	}
	return value, nil
}

func (c *fakeCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sets = append(c.sets, cacheSetCall{key: key, value: value, ttl: ttl})
	if c.setErr != nil {
		return c.setErr
	}
	c.store[key] = value
	return nil
}

// setsFor returns the recorded Set calls for a given key.
func (c *fakeCache) setsFor(key string) []cacheSetCall {
	c.mu.Lock()
	defer c.mu.Unlock()

	var out []cacheSetCall
	for _, s := range c.sets {
		if s.key == key {
			out = append(out, s)
		}
	}
	return out
}

// --- fake repository --------------------------------------------------------

type fakeContactRepo struct {
	mu sync.Mutex

	contacts []*entities.Contact

	storeCalls  [][]*entities.Contact
	updateCalls []*entities.Contact
	loadCalls   []loadCall
	indexCalls  []indexCall
	countCalls  []indexCall
	deleteCalls []deleteCall
	fetchAll    int

	storeErr  error
	updateErr error
	loadErr   error
	indexErr  error
	countErr  error
	deleteErr error
	fetchErr  error

	indexResult []entities.Contact
	countResult int64
}

type loadCall struct {
	userID entities.UserID
	id     uuid.UUID
}

type indexCall struct {
	userID entities.UserID
	params repositories.IndexParams
}

type deleteCall struct {
	userID entities.UserID
	id     uuid.UUID
}

func (r *fakeContactRepo) Store(_ context.Context, contacts []*entities.Contact) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.storeCalls = append(r.storeCalls, contacts)
	if r.storeErr != nil {
		return r.storeErr
	}
	r.contacts = append(r.contacts, contacts...)
	return nil
}

func (r *fakeContactRepo) Update(_ context.Context, contact *entities.Contact) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.updateCalls = append(r.updateCalls, contact)
	if r.updateErr != nil {
		return r.updateErr
	}
	return nil
}

func (r *fakeContactRepo) Load(_ context.Context, userID entities.UserID, id uuid.UUID) (*entities.Contact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.loadCalls = append(r.loadCalls, loadCall{userID: userID, id: id})
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	for _, c := range r.contacts {
		if c.ID == id && c.UserID == userID {
			return c, nil
		}
	}
	return nil, stacktrace.NewErrorWithCodef(repositories.ErrCodeNotFound, "contact [%s] not found", id)
}

func (r *fakeContactRepo) Index(_ context.Context, userID entities.UserID, params repositories.IndexParams) (*[]entities.Contact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.indexCalls = append(r.indexCalls, indexCall{userID: userID, params: params})
	if r.indexErr != nil {
		return nil, r.indexErr
	}
	out := append([]entities.Contact{}, r.indexResult...)
	return &out, nil
}

func (r *fakeContactRepo) Count(_ context.Context, userID entities.UserID, params repositories.IndexParams) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.countCalls = append(r.countCalls, indexCall{userID: userID, params: params})
	if r.countErr != nil {
		return 0, r.countErr
	}
	return r.countResult, nil
}

func (r *fakeContactRepo) FetchAll(_ context.Context, _ entities.UserID) (*[]entities.Contact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.fetchAll++
	if r.fetchErr != nil {
		return nil, r.fetchErr
	}
	out := make([]entities.Contact, 0, len(r.contacts))
	for _, c := range r.contacts {
		out = append(out, *c)
	}
	return &out, nil
}

func (r *fakeContactRepo) Delete(_ context.Context, userID entities.UserID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deleteCalls = append(r.deleteCalls, deleteCall{userID: userID, id: id})
	return r.deleteErr
}

func (r *fakeContactRepo) DeleteAllForUser(_ context.Context, _ entities.UserID) error { return nil }

// --- recording logger -------------------------------------------------------

type recordingLogger struct {
	*noopLogger
	mu     sync.Mutex
	errors []error
	warns  []error
	infos  []string
	debugs []string
}

func newRecordingLogger() *recordingLogger {
	return &recordingLogger{noopLogger: &noopLogger{}}
}

func (l *recordingLogger) Error(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, err)
}

func (l *recordingLogger) Warn(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warns = append(l.warns, err)
}

func (l *recordingLogger) Info(v string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infos = append(l.infos, v)
}

func (l *recordingLogger) Debug(v string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugs = append(l.debugs, v)
}

func (l *recordingLogger) WithService(_ string) telemetry.Logger   { return l }
func (l *recordingLogger) WithString(_, _ string) telemetry.Logger { return l }
func (l *recordingLogger) WithSpan(_ trace.SpanContext) telemetry.Logger {
	return l
}

func (l *recordingLogger) errorCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.errors)
}

// --- helpers ---------------------------------------------------------------

func newContactServiceForTest(t *testing.T, repo repositories.ContactRepository, appCache *fakeCache, logger telemetry.Logger) *ContactService {
	t.Helper()
	if logger == nil {
		logger = &noopLogger{}
	}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewContactService(logger, tracer, repo, appCache)
}

// --- tests -----------------------------------------------------------------

func TestContactService_GetContactMap_TieBreakMostRecentlyUpdatedWins(t *testing.T) {
	older := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "Old", PhoneNumbers: pq.StringArray{"+18005550199"}, UpdatedAt: time.Now().Add(-time.Hour)}
	newer := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "New", PhoneNumbers: pq.StringArray{"+18005550199"}, UpdatedAt: time.Now()}
	repo := &fakeContactRepo{contacts: []*entities.Contact{older, newer}}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	result, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)

	require.NotNil(t, result["+18005550199"])
	assert.Equal(t, "New", result["+18005550199"].Name)
	assert.Equal(t, newer.ID, result["+18005550199"].ID)
}

func TestContactService_GetContactMap_AllPhoneNumbersMapToOneContact(t *testing.T) {
	contact := &entities.Contact{
		ID:           uuid.New(),
		UserID:       "u1",
		Name:         "Alice",
		PhoneNumbers: pq.StringArray{"+18005550199", "+18005550100", "+18005550111"},
	}
	repo := &fakeContactRepo{contacts: []*entities.Contact{contact}}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	result, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)

	for _, num := range contact.PhoneNumbers {
		got, ok := result[num]
		require.True(t, ok, "missing entry for %s", num)
		assert.Equal(t, contact.ID, got.ID)
		assert.Equal(t, "Alice", got.Name)
	}
}

func TestContactService_GetContactMap_NoCollisionDistinctPointersPerContact(t *testing.T) {
	a := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}, UpdatedAt: time.Now().Add(-2 * time.Hour)}
	b := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "Bob", PhoneNumbers: pq.StringArray{"+18005550100"}, UpdatedAt: time.Now().Add(-time.Hour)}
	c := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "Carol", PhoneNumbers: pq.StringArray{"+18005550111"}, UpdatedAt: time.Now()}
	repo := &fakeContactRepo{contacts: []*entities.Contact{a, b, c}}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	result, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)

	require.Len(t, result, 3)
	assert.Equal(t, "Alice", result["+18005550199"].Name)
	assert.Equal(t, "Bob", result["+18005550100"].Name)
	assert.Equal(t, "Carol", result["+18005550111"].Name)

	// Each entry must point to a stable, independent value.
	assert.NotSame(t, result["+18005550199"], result["+18005550100"])
	assert.NotSame(t, result["+18005550100"], result["+18005550111"])

	// Mutating the returned pointer must not corrupt other entries.
	result["+18005550199"].Name = "Mutated"
	assert.Equal(t, "Bob", result["+18005550100"].Name)
	assert.Equal(t, "Carol", result["+18005550111"].Name)
}

func TestContactService_GetContactMap_CacheHitAvoidsFetchAll(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{{
		ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"},
	}}}
	appCache := newFakeCache()
	service := newContactServiceForTest(t, repo, appCache, nil)

	_, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	_, err = service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)

	assert.Equal(t, 1, repo.fetchAll, "second call must hit cache")
}

func TestContactService_GetContactMap_EmptyMarkerTriggersRebuild(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{{
		ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"},
	}}}
	appCache := newFakeCache()
	// Pre-seed the invalidation marker.
	require.NoError(t, appCache.Set(context.Background(), "contacts.map.u1", "", time.Hour))
	logger := newRecordingLogger()
	service := newContactServiceForTest(t, repo, appCache, logger)

	result, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)

	assert.Equal(t, 1, repo.fetchAll, "empty marker must force rebuild")
	assert.Equal(t, "Alice", result["+18005550199"].Name)
	assert.Equal(t, 0, logger.errorCount(), "empty marker rebuild must not log an error")
}

func TestContactService_GetContactMap_CorruptCacheRebuildsAndLogs(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{{
		ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"},
	}}}
	appCache := newFakeCache()
	require.NoError(t, appCache.Set(context.Background(), "contacts.map.u1", "not-json{", time.Hour))
	logger := newRecordingLogger()
	service := newContactServiceForTest(t, repo, appCache, logger)

	result, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)

	assert.Equal(t, 1, repo.fetchAll, "corrupt cache must force rebuild")
	assert.Equal(t, "Alice", result["+18005550199"].Name)
	assert.GreaterOrEqual(t, logger.errorCount(), 1, "corrupt cache must log an error")
}

func TestContactService_GetContactMap_CachePopulationErrorReturnsMapAndLogs(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{{
		ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"},
	}}}
	appCache := newFakeCache()
	appCache.setErr = errors.New("boom")
	logger := newRecordingLogger()
	service := newContactServiceForTest(t, repo, appCache, logger)

	result, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err, "population failure must not fail the caller")
	require.NotNil(t, result["+18005550199"])
	assert.Equal(t, "Alice", result["+18005550199"].Name)
	assert.GreaterOrEqual(t, logger.errorCount(), 1, "population failure must be logged")
}

func TestContactService_GetContactMap_CachedRoundTripPreservesData(t *testing.T) {
	// After the first call populates the cache, verify the cached payload is well-formed JSON
	// mapping phone numbers to contact objects and that a second call returns equivalent data.
	repo := &fakeContactRepo{contacts: []*entities.Contact{{
		ID:           uuid.New(),
		UserID:       "u1",
		Name:         "Alice",
		PhoneNumbers: pq.StringArray{"+18005550199", "+18005550100"},
	}}}
	appCache := newFakeCache()
	service := newContactServiceForTest(t, repo, appCache, nil)

	first, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)

	raw, err := appCache.Get(context.Background(), "contacts.map.u1")
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	decoded := map[string]*entities.Contact{}
	require.NoError(t, json.Unmarshal([]byte(raw), &decoded))
	assert.Equal(t, first["+18005550199"].ID, decoded["+18005550199"].ID)

	second, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	assert.Equal(t, first["+18005550199"].ID, second["+18005550199"].ID)
	assert.Equal(t, first["+18005550100"].ID, second["+18005550100"].ID)
}

// --- mutation invalidation tests ------------------------------------------

func TestContactService_CreateMany_InvalidatesCacheExactlyOnce(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{{
		ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"},
	}}}
	appCache := newFakeCache()
	service := newContactServiceForTest(t, repo, appCache, nil)

	// Warm the cache.
	_, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	require.Equal(t, 1, repo.fetchAll)

	setsBefore := len(appCache.setsFor("contacts.map.u1"))

	added := []*entities.Contact{
		{ID: uuid.New(), UserID: "u1", Name: "Bob", PhoneNumbers: pq.StringArray{"+18005550100"}},
		{ID: uuid.New(), UserID: "u1", Name: "Carol", PhoneNumbers: pq.StringArray{"+18005550111"}},
	}
	require.NoError(t, service.CreateMany(context.Background(), entities.UserID("u1"), added))

	assert.Len(t, repo.storeCalls, 1, "batch must be persisted in a single Store call")
	assert.Equal(t, added, repo.storeCalls[0])

	afterInvalidation := appCache.setsFor("contacts.map.u1")
	require.Len(t, afterInvalidation, setsBefore+1, "CreateMany must invalidate the cache exactly once")
	assert.Equal(t, "", afterInvalidation[len(afterInvalidation)-1].value, "invalidation marker must be empty")

	// Next GetContactMap rebuilds.
	_, err = service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	assert.Equal(t, 2, repo.fetchAll)
}

func TestContactService_Update_InvalidatesCacheExactlyOnce(t *testing.T) {
	c := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}}
	repo := &fakeContactRepo{contacts: []*entities.Contact{c}}
	appCache := newFakeCache()
	service := newContactServiceForTest(t, repo, appCache, nil)

	_, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	setsBefore := len(appCache.setsFor("contacts.map.u1"))

	updated := *c
	updated.Name = "Alicia"
	require.NoError(t, service.Update(context.Background(), &updated))

	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, "Alicia", repo.updateCalls[0].Name)
	assert.Equal(t, entities.UserID("u1"), repo.updateCalls[0].UserID)

	afterInvalidation := appCache.setsFor("contacts.map.u1")
	require.Len(t, afterInvalidation, setsBefore+1)
	assert.Equal(t, "", afterInvalidation[len(afterInvalidation)-1].value)
}

func TestContactService_Delete_InvalidatesCacheExactlyOnce(t *testing.T) {
	id := uuid.New()
	repo := &fakeContactRepo{contacts: []*entities.Contact{{ID: id, UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}}}}
	appCache := newFakeCache()
	service := newContactServiceForTest(t, repo, appCache, nil)

	_, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.NoError(t, err)
	setsBefore := len(appCache.setsFor("contacts.map.u1"))

	require.NoError(t, service.Delete(context.Background(), entities.UserID("u1"), id))

	require.Len(t, repo.deleteCalls, 1)
	assert.Equal(t, deleteCall{userID: "u1", id: id}, repo.deleteCalls[0])

	afterInvalidation := appCache.setsFor("contacts.map.u1")
	require.Len(t, afterInvalidation, setsBefore+1)
	assert.Equal(t, "", afterInvalidation[len(afterInvalidation)-1].value)
}

func TestContactService_InvalidationFailure_LogsErrorButReturnsNil(t *testing.T) {
	// Persistence succeeded but cache invalidation failed. The mutation itself
	// must not return an error (see report: chosen contract), but MUST log the
	// invalidation failure explicitly at Error level so operators are alerted.
	repo := &fakeContactRepo{}
	appCache := newFakeCache()
	appCache.setErr = errors.New("cache down")
	logger := newRecordingLogger()
	service := newContactServiceForTest(t, repo, appCache, logger)

	err := service.CreateMany(context.Background(), entities.UserID("u1"), []*entities.Contact{{
		ID: uuid.New(), UserID: "u1", Name: "Bob", PhoneNumbers: pq.StringArray{"+18005550100"},
	}})
	require.NoError(t, err, "mutation must not surface invalidation failure to caller")

	// Repository actually got the write.
	require.Len(t, repo.storeCalls, 1)
	// The invalidation attempt was logged as an error.
	assert.GreaterOrEqual(t, logger.errorCount(), 1, "invalidation failure must be logged as error")
}

// --- CRUD delegation and user scope tests ---------------------------------

func TestContactService_CreateMany_PersistsBatchInSingleCall(t *testing.T) {
	repo := &fakeContactRepo{}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	batch := []*entities.Contact{
		{ID: uuid.New(), UserID: "u1", Name: "A", PhoneNumbers: pq.StringArray{"+18005550100"}},
		{ID: uuid.New(), UserID: "u1", Name: "B", PhoneNumbers: pq.StringArray{"+18005550111"}},
	}
	require.NoError(t, service.CreateMany(context.Background(), entities.UserID("u1"), batch))

	require.Len(t, repo.storeCalls, 1)
	assert.Equal(t, batch, repo.storeCalls[0])
}

func TestContactService_CreateMany_RepositoryErrorIsWrapped(t *testing.T) {
	repo := &fakeContactRepo{storeErr: errors.New("db down")}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	err := service.CreateMany(context.Background(), entities.UserID("u1"), []*entities.Contact{{
		ID: uuid.New(), UserID: "u1", Name: "A", PhoneNumbers: pq.StringArray{"+18005550100"},
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

func TestContactService_Get_DelegatesWithUserScope(t *testing.T) {
	id := uuid.New()
	other := uuid.New()
	repo := &fakeContactRepo{contacts: []*entities.Contact{
		{ID: id, UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}},
		{ID: other, UserID: "u2", Name: "Bob", PhoneNumbers: pq.StringArray{"+18005550100"}},
	}}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	got, err := service.Get(context.Background(), entities.UserID("u1"), id)
	require.NoError(t, err)
	assert.Equal(t, "Alice", got.Name)

	// Wrong user scope must not resolve the contact.
	_, err = service.Get(context.Background(), entities.UserID("u1"), other)
	require.Error(t, err)
	assert.Equal(t, repositories.ErrCodeNotFound, stacktrace.GetCode(err))
}

func TestContactService_Index_DelegatesParams(t *testing.T) {
	want := []entities.Contact{{ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}}}
	repo := &fakeContactRepo{indexResult: want}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	params := repositories.IndexParams{Skip: 5, Limit: 10, SortBy: "name", Query: "Ali"}
	got, err := service.Index(context.Background(), entities.UserID("u1"), params)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, want, *got)

	require.Len(t, repo.indexCalls, 1)
	assert.Equal(t, entities.UserID("u1"), repo.indexCalls[0].userID)
	assert.Equal(t, params, repo.indexCalls[0].params)
}

func TestContactService_Index_RepositoryErrorIsWrapped(t *testing.T) {
	repo := &fakeContactRepo{indexErr: errors.New("index boom")}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	_, err := service.Index(context.Background(), entities.UserID("u1"), repositories.IndexParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index boom")
}

func TestContactService_Count_DelegatesParamsAndReturnsTotal(t *testing.T) {
	repo := &fakeContactRepo{countResult: 57}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	params := repositories.IndexParams{Skip: 5, Limit: 10, Query: "Ali"}
	total, err := service.Count(context.Background(), entities.UserID("u1"), params)
	require.NoError(t, err)
	assert.Equal(t, int64(57), total)

	require.Len(t, repo.countCalls, 1)
	assert.Equal(t, entities.UserID("u1"), repo.countCalls[0].userID)
	assert.Equal(t, params, repo.countCalls[0].params)
}

func TestContactService_Count_RepositoryErrorIsWrapped(t *testing.T) {
	repo := &fakeContactRepo{countErr: errors.New("count boom")}
	service := newContactServiceForTest(t, repo, newFakeCache(), nil)

	_, err := service.Count(context.Background(), entities.UserID("u1"), repositories.IndexParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count boom")
}

func TestContactService_Update_RepositoryErrorIsWrappedAndSkipsInvalidation(t *testing.T) {
	repo := &fakeContactRepo{updateErr: errors.New("update boom")}
	appCache := newFakeCache()
	service := newContactServiceForTest(t, repo, appCache, nil)

	err := service.Update(context.Background(), &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "A", PhoneNumbers: pq.StringArray{"+18005550100"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update boom")
	assert.Len(t, appCache.setsFor("contacts.map.u1"), 0, "invalidation must not run when the write fails")
}

func TestContactService_Delete_RepositoryErrorIsWrappedAndSkipsInvalidation(t *testing.T) {
	repo := &fakeContactRepo{deleteErr: errors.New("delete boom")}
	appCache := newFakeCache()
	service := newContactServiceForTest(t, repo, appCache, nil)

	err := service.Delete(context.Background(), entities.UserID("u1"), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete boom")
	assert.Len(t, appCache.setsFor("contacts.map.u1"), 0)
}

func TestContactService_GetContactMap_FetchAllErrorIsWrapped(t *testing.T) {
	repo := &fakeContactRepo{fetchErr: errors.New("fetch boom")}
	appCache := newFakeCache()
	service := newContactServiceForTest(t, repo, appCache, nil)

	_, err := service.GetContactMap(context.Background(), entities.UserID("u1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch boom")
}
