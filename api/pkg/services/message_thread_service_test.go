package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type messageThreadRepositoryStub struct {
	loadByOwnerContact func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error)
	load               func(context.Context, entities.UserID, uuid.UUID) (*entities.MessageThread, error)
	store              func(context.Context, *entities.MessageThread) error
	updateActivity     func(context.Context, repositories.MessageThreadActivityUpdate) error
	updateStatus       func(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error)
}

func (stub *messageThreadRepositoryStub) Store(ctx context.Context, thread *entities.MessageThread) error {
	if stub.store != nil {
		return stub.store(ctx, thread)
	}
	return nil
}

func (stub *messageThreadRepositoryStub) UpdateActivity(ctx context.Context, params repositories.MessageThreadActivityUpdate) error {
	if stub.updateActivity != nil {
		return stub.updateActivity(ctx, params)
	}
	return nil
}

func (stub *messageThreadRepositoryStub) UpdateStatus(ctx context.Context, userID entities.UserID, threadID uuid.UUID, params repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error) {
	if stub.updateStatus != nil {
		return stub.updateStatus(ctx, userID, threadID, params)
	}
	return &entities.MessageThread{ID: threadID}, nil
}

func (stub *messageThreadRepositoryStub) UpdateAfterDeletedMessage(context.Context, repositories.MessageThreadDeletedUpdate) error {
	return nil
}

func (stub *messageThreadRepositoryStub) LoadByOwnerContact(ctx context.Context, userID entities.UserID, owner string, contact string) (*entities.MessageThread, error) {
	return stub.loadByOwnerContact(ctx, userID, owner, contact)
}

func (stub *messageThreadRepositoryStub) Load(ctx context.Context, userID entities.UserID, id uuid.UUID) (*entities.MessageThread, error) {
	return stub.load(ctx, userID, id)
}

func (stub *messageThreadRepositoryStub) Index(context.Context, entities.UserID, string, bool, repositories.IndexParams) (*[]entities.MessageThread, error) {
	threads := []entities.MessageThread{}
	return &threads, nil
}

func (stub *messageThreadRepositoryStub) Delete(context.Context, entities.UserID, uuid.UUID) error {
	return nil
}

func (stub *messageThreadRepositoryStub) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}

func newMessageThreadServiceForTest(repository repositories.MessageThreadRepository) *MessageThreadService {
	logger := &noopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewMessageThreadService(logger, tracer, repository, nil, nil)
}

func TestUpdateThreadPassesUnreadWatermarkForInboundActivity(t *testing.T) {
	threadID := uuid.New()
	eventTimestamp := time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC)
	var captured repositories.MessageThreadActivityUpdate
	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{ID: threadID}, nil
		},
		updateActivity: func(_ context.Context, params repositories.MessageThreadActivityUpdate) error {
			captured = params
			return nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
		UserID:         entities.UserID("user-id"),
		Owner:          "+18005550199",
		Contact:        "+18005550100",
		MessageID:      uuid.New(),
		Content:        "hello",
		Status:         entities.MessageStatusReceived,
		Timestamp:      eventTimestamp,
		MarkAsUnread:   true,
		EventTimestamp: eventTimestamp,
	})

	require.NoError(t, err)
	assert.True(t, captured.MarkAsUnread)
	assert.Equal(t, eventTimestamp, captured.EventTimestamp)
}

func TestUpdateThreadPreservesReadStateForOutboundActivity(t *testing.T) {
	var captured repositories.MessageThreadActivityUpdate
	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{ID: uuid.New(), IsRead: false}, nil
		},
		updateActivity: func(_ context.Context, params repositories.MessageThreadActivityUpdate) error {
			captured = params
			return nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
		UserID:    entities.UserID("user-id"),
		Owner:     "+18005550199",
		Contact:   "+18005550100",
		MessageID: uuid.New(),
		Content:   "outbound",
		Status:    entities.MessageStatusSent,
		Timestamp: time.Now().UTC(),
	})

	require.NoError(t, err)
	assert.False(t, captured.MarkAsUnread)
}

func TestCreateThreadSetsReadStateFromActivityDirection(t *testing.T) {
	tests := []struct {
		name        string
		marksUnread bool
		wantRead    bool
	}{
		{name: "inbound", marksUnread: true, wantRead: false},
		{name: "outbound", marksUnread: false, wantRead: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stored *entities.MessageThread
			repository := &messageThreadRepositoryStub{
				loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
					return nil, stacktrace.PropagateWithCode(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
				},
				store: func(_ context.Context, thread *entities.MessageThread) error {
					stored = thread
					return nil
				},
			}

			service := newMessageThreadServiceForTest(repository)
			err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
				UserID:       entities.UserID("user-id"),
				Owner:        "+18005550199",
				Contact:      "+18005550100",
				MessageID:    uuid.New(),
				Content:      "hello",
				Status:       entities.MessageStatusReceived,
				Timestamp:    time.Now().UTC(),
				MarkAsUnread: test.marksUnread,
			})

			require.NoError(t, err)
			require.NotNil(t, stored)
			assert.Equal(t, test.wantRead, stored.IsRead)
			assert.False(t, stored.LastReadAt.IsZero())
		})
	}
}

func TestUpdateStatusChangesOnlyRequestedState(t *testing.T) {
	threadID := uuid.New()
	isRead := false
	var captured repositories.MessageThreadStatusUpdate
	repository := &messageThreadRepositoryStub{
		updateStatus: func(_ context.Context, _ entities.UserID, _ uuid.UUID, params repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error) {
			captured = params
			return &entities.MessageThread{ID: threadID, IsArchived: true, IsRead: false}, nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	thread, err := service.UpdateStatus(context.Background(), MessageThreadStatusParams{
		UserID:          entities.UserID("user-id"),
		MessageThreadID: threadID,
		IsRead:          &isRead,
	})

	require.NoError(t, err)
	assert.Nil(t, captured.IsArchived)
	assert.Same(t, &isRead, captured.IsRead)
	assert.False(t, captured.ReadAt.IsZero())
	assert.True(t, thread.IsArchived)
	assert.False(t, thread.IsRead)
}

func TestUpdateStatusPreservesNotFoundCode(t *testing.T) {
	repository := &messageThreadRepositoryStub{
		updateStatus: func(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error) {
			return nil, stacktrace.PropagateWithCode(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
		},
	}

	service := newMessageThreadServiceForTest(repository)
	isRead := true
	_, err := service.UpdateStatus(context.Background(), MessageThreadStatusParams{
		UserID:          entities.UserID("user-id"),
		MessageThreadID: uuid.New(),
		IsRead:          &isRead,
	})

	assert.Equal(t, repositories.ErrCodeNotFound, stacktrace.GetCode(err))
}

func TestShouldCheckUnarchive(t *testing.T) {
	service := &MessageThreadService{}

	archived := &entities.MessageThread{IsArchived: true}
	notArchived := &entities.MessageThread{IsArchived: false}

	received := MessageThreadUpdateParams{Status: entities.MessageStatusReceived}
	sent := MessageThreadUpdateParams{Status: entities.MessageStatusSent}

	assert.True(t, service.shouldCheckUnarchive(archived, received), "archived + inbound -> consult phone setting")
	assert.False(t, service.shouldCheckUnarchive(archived, sent), "outbound status -> no check")
	assert.False(t, service.shouldCheckUnarchive(notArchived, received), "already unarchived -> no check")
	assert.False(t, service.shouldCheckUnarchive(notArchived, sent), "not archived + outbound -> no check")
}
