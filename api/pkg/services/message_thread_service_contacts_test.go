package services

import (
	"context"
	"errors"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

type messageThreadContactRepositoryStub struct {
	repositories.MessageThreadRepository
	threads []entities.MessageThread
	err     error
	calls   int
}

func (stub *messageThreadContactRepositoryStub) Index(_ context.Context, _ entities.UserID, _ string, _ bool, _ repositories.IndexParams) (*[]entities.MessageThread, error) {
	stub.calls++
	if stub.err != nil {
		return nil, stub.err
	}
	threads := make([]entities.MessageThread, len(stub.threads))
	copy(threads, stub.threads)
	return &threads, nil
}

type messageThreadContactProviderStub struct {
	contacts map[string]*entities.Contact
	err      error
	calls    int
	userID   entities.UserID
}

func (stub *messageThreadContactProviderStub) GetContactMap(_ context.Context, userID entities.UserID) (map[string]*entities.Contact, error) {
	stub.calls++
	stub.userID = userID
	return stub.contacts, stub.err
}

type messageThreadContactLogger struct {
	errors []error
}

var _ telemetry.Logger = (*messageThreadContactLogger)(nil)

func (logger *messageThreadContactLogger) Error(err error) {
	logger.errors = append(logger.errors, err)
}
func (logger *messageThreadContactLogger) WithService(string) telemetry.Logger { return logger }
func (logger *messageThreadContactLogger) WithString(string, string) telemetry.Logger { return logger }

func (logger *messageThreadContactLogger) WithSpan(trace.SpanContext) telemetry.Logger { return logger }
func (logger *messageThreadContactLogger) Trace(string)                                {}
func (logger *messageThreadContactLogger) Info(string)                                 {}
func (logger *messageThreadContactLogger) Warn(error)                                  {}
func (logger *messageThreadContactLogger) Debug(string)                                {}
func (logger *messageThreadContactLogger) Fatal(error)                                 {}
func (logger *messageThreadContactLogger) Printf(string, ...interface{})               {}

func newMessageThreadContactServiceForTest(repository repositories.MessageThreadRepository, provider contactMapProvider, logger telemetry.Logger) *MessageThreadService {
	if logger == nil {
		logger = &noopLogger{}
	}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewMessageThreadService(logger, tracer, repository, nil, nil, provider)
}

func TestGetThreads_SkipsContactLookupWhenFlagOff(t *testing.T) {
	repository := &messageThreadContactRepositoryStub{threads: []entities.MessageThread{{Contact: "+18005550199"}}}
	provider := &messageThreadContactProviderStub{contacts: map[string]*entities.Contact{
		"+18005550199": {Name: "Alice"},
	}}
	service := newMessageThreadContactServiceForTest(repository, provider, nil)

	threads, err := service.GetThreads(context.Background(), MessageThreadGetParams{UserID: entities.UserID("user-id"), WithContacts: false})

	require.NoError(t, err)
	require.Len(t, *threads, 1)
	assert.Nil(t, (*threads)[0].ContactDetails)
	assert.Equal(t, 0, provider.calls)
}

func TestGetThreads_AttachesContactDetailsWhenFlagOn(t *testing.T) {
	alice := &entities.Contact{ID: uuid.New(), Name: "Alice", PhoneNumbers: []string{"+18005550199"}}
	repository := &messageThreadContactRepositoryStub{threads: []entities.MessageThread{
		{Contact: "+18005550199"},
		{Contact: "+18005550100"},
	}}
	provider := &messageThreadContactProviderStub{contacts: map[string]*entities.Contact{
		"+18005550199": alice,
	}}
	service := newMessageThreadContactServiceForTest(repository, provider, nil)

	threads, err := service.GetThreads(context.Background(), MessageThreadGetParams{UserID: entities.UserID("user-id"), WithContacts: true})

	require.NoError(t, err)
	require.Len(t, *threads, 2)
	assert.Equal(t, 1, provider.calls)
	assert.Equal(t, entities.UserID("user-id"), provider.userID)
	require.NotNil(t, (*threads)[0].ContactDetails)
	assert.Same(t, alice, (*threads)[0].ContactDetails)
	assert.Equal(t, "Alice", (*threads)[0].ContactDetails.Name)
	assert.Nil(t, (*threads)[1].ContactDetails)
}

func TestGetThreads_SkipsContactLookupWhenNoThreads(t *testing.T) {
	repository := &messageThreadContactRepositoryStub{threads: []entities.MessageThread{}}
	provider := &messageThreadContactProviderStub{contacts: map[string]*entities.Contact{}}
	service := newMessageThreadContactServiceForTest(repository, provider, nil)

	threads, err := service.GetThreads(context.Background(), MessageThreadGetParams{UserID: entities.UserID("user-id"), WithContacts: true})

	require.NoError(t, err)
	require.Empty(t, *threads)
	assert.Equal(t, 0, provider.calls)
}

func TestGetThreads_LogsContactMapErrorAndReturnsThreads(t *testing.T) {
	repository := &messageThreadContactRepositoryStub{threads: []entities.MessageThread{{Contact: "+18005550199"}}}
	provider := &messageThreadContactProviderStub{err: errors.New("contacts unavailable")}
	logger := &messageThreadContactLogger{}
	service := newMessageThreadContactServiceForTest(repository, provider, logger)

	threads, err := service.GetThreads(context.Background(), MessageThreadGetParams{UserID: entities.UserID("user-id"), WithContacts: true})

	require.NoError(t, err)
	require.Len(t, *threads, 1)
	assert.Nil(t, (*threads)[0].ContactDetails)
	assert.Equal(t, 1, provider.calls)
	require.Len(t, logger.errors, 1)
	assert.ErrorContains(t, logger.errors[0], "cannot build contact map")
}

func TestGetThreads_DoesNotLookupContactsWhenRepositoryFails(t *testing.T) {
	repository := &messageThreadContactRepositoryStub{err: stacktrace.NewError("repository failed")}
	provider := &messageThreadContactProviderStub{contacts: map[string]*entities.Contact{
		"+18005550199": {Name: "Alice"},
	}}
	service := newMessageThreadContactServiceForTest(repository, provider, nil)

	threads, err := service.GetThreads(context.Background(), MessageThreadGetParams{UserID: entities.UserID("user-id"), WithContacts: true})

	require.Error(t, err)
	assert.Nil(t, threads)
	assert.Equal(t, 0, provider.calls)
}
