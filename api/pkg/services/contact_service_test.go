package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// --- fake repository --------------------------------------------------------

type fakeContactRepo struct {
	mu sync.Mutex

	contacts []*entities.Contact

	storeCalls     [][]*entities.Contact
	updateCalls    []*entities.Contact
	loadCalls      []loadCall
	indexCalls     []indexCall
	countCalls     []indexCall
	deleteCalls    []deleteCall
	deleteAllCalls []entities.UserID
	fetchCalls     []fetchCall

	storeErr     error
	updateErr    error
	loadErr      error
	indexErr     error
	countErr     error
	deleteErr    error
	deleteAllErr error
	fetchErr     error

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

type fetchCall struct {
	userID       entities.UserID
	phoneNumbers []string
}

type blockingContactRepo struct {
	*fakeContactRepo
	fetchStarted chan struct{}
	releaseFetch chan struct{}
	fetchOnce    sync.Once
}

func (r *blockingContactRepo) FetchByPhoneNumbers(ctx context.Context, userID entities.UserID, phoneNumbers []string) (*[]entities.Contact, error) {
	var captured []entities.Contact
	r.fetchOnce.Do(func() {
		r.fakeContactRepo.mu.Lock()
		captured = append(captured, *r.fakeContactRepo.contacts[0])
		r.fakeContactRepo.mu.Unlock()
		close(r.fetchStarted)
		<-r.releaseFetch
	})
	if captured != nil {
		return &captured, nil
	}
	return r.fakeContactRepo.FetchByPhoneNumbers(ctx, userID, phoneNumbers)
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
	for index := range r.contacts {
		if r.contacts[index].ID == contact.ID && r.contacts[index].UserID == contact.UserID {
			clone := *contact
			r.contacts[index] = &clone
			break
		}
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

func (r *fakeContactRepo) FetchByPhoneNumbers(_ context.Context, userID entities.UserID, phoneNumbers []string) (*[]entities.Contact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.fetchCalls = append(r.fetchCalls, fetchCall{
		userID:       userID,
		phoneNumbers: append([]string{}, phoneNumbers...),
	})
	if r.fetchErr != nil {
		return nil, r.fetchErr
	}
	requested := make(map[string]struct{}, len(phoneNumbers))
	for _, phoneNumber := range phoneNumbers {
		requested[phoneNumber] = struct{}{}
	}

	out := make([]entities.Contact, 0)
	for _, c := range r.contacts {
		if c.UserID != userID {
			continue
		}
		for _, phoneNumber := range c.PhoneNumbers {
			if _, ok := requested[phoneNumber]; ok {
				out = append(out, *c)
				break
			}
		}
	}
	return &out, nil
}

func (r *fakeContactRepo) Delete(_ context.Context, userID entities.UserID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deleteCalls = append(r.deleteCalls, deleteCall{userID: userID, id: id})
	return r.deleteErr
}

func (r *fakeContactRepo) DeleteAllForUser(_ context.Context, userID entities.UserID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deleteAllCalls = append(r.deleteAllCalls, userID)
	if r.deleteAllErr != nil {
		return r.deleteAllErr
	}
	remaining := r.contacts[:0]
	for _, contact := range r.contacts {
		if contact.UserID != userID {
			remaining = append(remaining, contact)
		}
	}
	r.contacts = remaining
	return nil
}

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

// --- helpers ---------------------------------------------------------------

func newContactCache(t *testing.T) *ristretto.Cache[string, ContactCacheEntry] {
	t.Helper()

	contactCache, err := ristretto.NewCache[string, ContactCacheEntry](&ristretto.Config[string, ContactCacheEntry]{
		MaxCost:     100,
		NumCounters: 1_000,
		BufferItems: 64,
	})
	require.NoError(t, err)
	t.Cleanup(contactCache.Close)
	return contactCache
}

func newContactServiceForTest(t *testing.T, repo repositories.ContactRepository, logger telemetry.Logger) *ContactService {
	t.Helper()
	if logger == nil {
		logger = &noopLogger{}
	}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewContactService(logger, tracer, repo, newContactCache(t))
}

// --- tests -----------------------------------------------------------------

func TestContactService_GetContactMap_FetchesOnlyUncachedPhoneNumbers(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{
		{ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}},
		{ID: uuid.New(), UserID: "u1", Name: "Bob", PhoneNumbers: pq.StringArray{"+18005550100"}},
		{ID: uuid.New(), UserID: "u1", Name: "Carol", PhoneNumbers: pq.StringArray{"+18005550111"}},
	}}
	service := newContactServiceForTest(t, repo, nil)

	first, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{"+18005550199"})
	require.NoError(t, err)
	service.cache.Wait()
	require.Len(t, first, 1)

	second, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{"+18005550199", "+18005550111"})
	require.NoError(t, err)

	require.Len(t, repo.fetchCalls, 2)
	assert.Equal(t, []string{"+18005550199"}, repo.fetchCalls[0].phoneNumbers)
	assert.Equal(t, []string{"+18005550111"}, repo.fetchCalls[1].phoneNumbers)
	assert.Equal(t, "Carol", second["+18005550111"].Name)
}

func TestContactService_GetContactMap_CacheKeyIncludesUserAndPhoneNumber(t *testing.T) {
	number := "+18005550199"
	repo := &fakeContactRepo{contacts: []*entities.Contact{
		{ID: uuid.New(), UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{number}},
		{ID: uuid.New(), UserID: "u2", Name: "Bob", PhoneNumbers: pq.StringArray{number}},
	}}
	service := newContactServiceForTest(t, repo, nil)

	first, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{number})
	require.NoError(t, err)
	service.cache.Wait()
	second, err := service.GetContactMap(context.Background(), entities.UserID("u2"), []string{number})
	require.NoError(t, err)

	assert.Equal(t, "Alice", first[number].Name)
	assert.Equal(t, "Bob", second[number].Name)
	require.Len(t, repo.fetchCalls, 2)
}

func TestContactService_GetContactMap_CachesMissingPhoneNumbers(t *testing.T) {
	repo := &fakeContactRepo{}
	service := newContactServiceForTest(t, repo, nil)
	number := "+18005550199"

	first, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{number})
	require.NoError(t, err)
	service.cache.Wait()
	second, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{number})
	require.NoError(t, err)

	assert.Empty(t, first)
	assert.Empty(t, second)
	require.Len(t, repo.fetchCalls, 1)
}

func TestContactService_ExpiresInactiveGenerationStateWithoutReusingEpoch(t *testing.T) {
	service := newContactServiceForTest(t, &fakeContactRepo{}, nil)

	first := service.generation(entities.UserID("u1"))
	service.expireGenerations(time.Now().Add(contactMapCacheTTL + time.Hour))
	second := service.generation(entities.UserID("u1"))

	assert.NotEqual(t, first, second)
}

func TestContactService_GetContactMap_TieBreakMostRecentlyUpdatedWins(t *testing.T) {
	number := "+18005550199"
	older := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "Old", PhoneNumbers: pq.StringArray{number}, UpdatedAt: time.Now().Add(-time.Hour)}
	newer := &entities.Contact{ID: uuid.New(), UserID: "u1", Name: "New", PhoneNumbers: pq.StringArray{number}, UpdatedAt: time.Now()}
	repo := &fakeContactRepo{contacts: []*entities.Contact{older, newer}}
	service := newContactServiceForTest(t, repo, nil)

	result, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{number})
	require.NoError(t, err)

	require.NotNil(t, result[number])
	assert.Equal(t, newer.ID, result[number].ID)
}

func TestContactService_Update_InvalidatesOldAndNewPhoneNumbers(t *testing.T) {
	oldNumber := "+18005550199"
	newNumber := "+18005550100"
	id := uuid.New()
	repo := &fakeContactRepo{contacts: []*entities.Contact{{
		ID: id, UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{oldNumber},
	}}}
	contactCache := newContactCache(t)
	stale := &entities.Contact{ID: id, UserID: "u1", Name: "Stale"}
	require.True(t, contactCache.Set("u1|"+oldNumber, ContactCacheEntry{contact: stale}, 1))
	require.True(t, contactCache.Set("u1|"+newNumber, ContactCacheEntry{contact: stale}, 1))
	contactCache.Wait()
	logger := &noopLogger{}
	service := NewContactService(logger, telemetry.NewOtelLogger("test", logger), repo, contactCache)

	err := service.Update(context.Background(), &entities.Contact{
		ID: id, UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{newNumber},
	}, []string{oldNumber})
	require.NoError(t, err)
	contactCache.Wait()

	_, oldFound := contactCache.Get("u1|" + oldNumber)
	_, newFound := contactCache.Get("u1|" + newNumber)
	assert.False(t, oldFound)
	assert.False(t, newFound)
}

func TestContactService_GetContactMap_DoesNotReuseResultFetchedDuringUpdate(t *testing.T) {
	number := "+18005550199"
	id := uuid.New()
	repo := &blockingContactRepo{
		fakeContactRepo: &fakeContactRepo{contacts: []*entities.Contact{{
			ID: id, UserID: "u1", Name: "Old", PhoneNumbers: pq.StringArray{number},
		}}},
		fetchStarted: make(chan struct{}),
		releaseFetch: make(chan struct{}),
	}
	service := newContactServiceForTest(t, repo, nil)

	firstResult := make(chan map[string]*entities.Contact, 1)
	go func() {
		result, _ := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{number})
		firstResult <- result
	}()
	<-repo.fetchStarted

	require.NoError(t, service.Update(context.Background(), &entities.Contact{
		ID: id, UserID: "u1", Name: "New", PhoneNumbers: pq.StringArray{number},
	}, []string{number}))
	close(repo.releaseFetch)
	assert.Equal(t, "Old", (<-firstResult)[number].Name)
	service.cache.Wait()

	second, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{number})
	require.NoError(t, err)
	assert.Equal(t, "New", second[number].Name)
}

func TestContactService_DeleteAllForUser_DoesNotEvictOtherUsers(t *testing.T) {
	repo := &fakeContactRepo{contacts: []*entities.Contact{
		{UserID: "u1", Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}},
		{UserID: "u2", Name: "Bob", PhoneNumbers: pq.StringArray{"+18005550100"}},
	}}
	contactCache := newContactCache(t)
	logger := &noopLogger{}
	service := NewContactService(logger, telemetry.NewOtelLogger("test", logger), repo, contactCache)
	require.True(t, contactCache.Set("u1|+18005550199", ContactCacheEntry{
		contact: repo.contacts[0], generation: service.generation("u1"),
	}, 1))
	require.True(t, contactCache.Set("u2|+18005550100", ContactCacheEntry{
		contact: repo.contacts[1], generation: service.generation("u2"),
	}, 1))
	contactCache.Wait()

	require.NoError(t, service.DeleteAllForUser(context.Background(), entities.UserID("u1")))

	first, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{"+18005550199"})
	require.NoError(t, err)
	second, err := service.GetContactMap(context.Background(), entities.UserID("u2"), []string{"+18005550100"})
	require.NoError(t, err)
	assert.Empty(t, first)
	assert.Equal(t, "Bob", second["+18005550100"].Name)
	require.Len(t, repo.fetchCalls, 1)
	assert.Equal(t, entities.UserID("u1"), repo.fetchCalls[0].userID)
}

// --- CRUD delegation and user scope tests ---------------------------------

func TestContactService_CreateMany_PersistsBatchInSingleCall(t *testing.T) {
	repo := &fakeContactRepo{}
	service := newContactServiceForTest(t, repo, nil)

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
	service := newContactServiceForTest(t, repo, nil)

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
	service := newContactServiceForTest(t, repo, nil)

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
	service := newContactServiceForTest(t, repo, nil)

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
	service := newContactServiceForTest(t, repo, nil)

	_, err := service.Index(context.Background(), entities.UserID("u1"), repositories.IndexParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index boom")
}

func TestContactService_Count_DelegatesParamsAndReturnsTotal(t *testing.T) {
	repo := &fakeContactRepo{countResult: 57}
	service := newContactServiceForTest(t, repo, nil)

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
	service := newContactServiceForTest(t, repo, nil)

	_, err := service.Count(context.Background(), entities.UserID("u1"), repositories.IndexParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count boom")
}

func TestContactService_Update_RepositoryErrorIsWrappedAndSkipsInvalidation(t *testing.T) {
	id := uuid.New()
	repo := &fakeContactRepo{
		contacts:  []*entities.Contact{{ID: id, UserID: "u1", PhoneNumbers: pq.StringArray{"+18005550100"}}},
		updateErr: errors.New("update boom"),
	}
	contactCache := newContactCache(t)
	require.True(t, contactCache.Set("u1|+18005550100", ContactCacheEntry{contact: &entities.Contact{Name: "Cached"}}, 1))
	contactCache.Wait()
	logger := &noopLogger{}
	service := NewContactService(logger, telemetry.NewOtelLogger("test", logger), repo, contactCache)

	err := service.Update(
		context.Background(),
		&entities.Contact{ID: id, UserID: "u1", Name: "A", PhoneNumbers: pq.StringArray{"+18005550100"}},
		[]string{"+18005550100"},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update boom")
	_, found := contactCache.Get("u1|+18005550100")
	assert.True(t, found, "invalidation must not run when the write fails")
}

func TestContactService_Delete_RepositoryErrorIsWrappedAndSkipsInvalidation(t *testing.T) {
	id := uuid.New()
	repo := &fakeContactRepo{
		contacts:  []*entities.Contact{{ID: id, UserID: "u1", PhoneNumbers: pq.StringArray{"+18005550100"}}},
		deleteErr: errors.New("delete boom"),
	}
	contactCache := newContactCache(t)
	require.True(t, contactCache.Set("u1|+18005550100", ContactCacheEntry{contact: &entities.Contact{Name: "Cached"}}, 1))
	contactCache.Wait()
	logger := &noopLogger{}
	service := NewContactService(logger, telemetry.NewOtelLogger("test", logger), repo, contactCache)

	err := service.Delete(context.Background(), entities.UserID("u1"), id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete boom")
	_, found := contactCache.Get("u1|+18005550100")
	assert.True(t, found, "invalidation must not run when the write fails")
}

func TestContactService_GetContactMap_FetchErrorIsWrapped(t *testing.T) {
	repo := &fakeContactRepo{fetchErr: errors.New("fetch boom")}
	service := newContactServiceForTest(t, repo, nil)

	_, err := service.GetContactMap(context.Background(), entities.UserID("u1"), []string{"+18005550100"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch boom")
}
