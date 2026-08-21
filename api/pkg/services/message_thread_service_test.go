package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type messageThreadRepositoryStub struct {
	loadByOwnerContact func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error)
	load               func(context.Context, entities.UserID, uuid.UUID) (*entities.MessageThread, error)
	store              func(context.Context, *entities.MessageThread, *uuid.UUID) error
	updateActivity     func(context.Context, repositories.MessageThreadActivityUpdate) error
	updateStatus       func(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error)
	updateAfterDelete  func(context.Context, repositories.MessageThreadDeletedUpdate) error
	delete             func(context.Context, entities.UserID, uuid.UUID) error
}

func (stub *messageThreadRepositoryStub) Store(ctx context.Context, thread *entities.MessageThread, unreadMessageID *uuid.UUID) error {
	if stub.store != nil {
		return stub.store(ctx, thread, unreadMessageID)
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

func (stub *messageThreadRepositoryStub) UpdateAfterDeletedMessage(ctx context.Context, params repositories.MessageThreadDeletedUpdate) error {
	if stub.updateAfterDelete != nil {
		return stub.updateAfterDelete(ctx, params)
	}
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

func (stub *messageThreadRepositoryStub) Delete(ctx context.Context, userID entities.UserID, threadID uuid.UUID) error {
	if stub.delete != nil {
		return stub.delete(ctx, userID, threadID)
	}
	return nil
}

func (stub *messageThreadRepositoryStub) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}

type messageThreadPhoneRepositoryStub struct {
	load func(context.Context, entities.UserID, string) (*entities.Phone, error)
}

func (stub *messageThreadPhoneRepositoryStub) Save(context.Context, *entities.Phone) error {
	return nil
}

func (stub *messageThreadPhoneRepositoryStub) Index(context.Context, entities.UserID, repositories.IndexParams) (*[]entities.Phone, error) {
	phones := []entities.Phone{}
	return &phones, nil
}

func (stub *messageThreadPhoneRepositoryStub) Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.Phone, error) {
	if stub.load != nil {
		return stub.load(ctx, userID, phoneNumber)
	}
	return &entities.Phone{}, nil
}

func (stub *messageThreadPhoneRepositoryStub) LoadByID(context.Context, entities.UserID, uuid.UUID) (*entities.Phone, error) {
	return &entities.Phone{}, nil
}

func (stub *messageThreadPhoneRepositoryStub) Delete(context.Context, entities.UserID, uuid.UUID) error {
	return nil
}

func (stub *messageThreadPhoneRepositoryStub) NullifyScheduleID(context.Context, entities.UserID, uuid.UUID) error {
	return nil
}

func (stub *messageThreadPhoneRepositoryStub) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}

func newMessageThreadServiceForTest(repository repositories.MessageThreadRepository) *MessageThreadService {
	logger := &noopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewMessageThreadService(logger, tracer, repository, nil, nil, nil)
}

func newMessageThreadServiceWithPhoneForTest(repository repositories.MessageThreadRepository, phoneRepository repositories.PhoneRepository) *MessageThreadService {
	logger := &noopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewMessageThreadService(logger, tracer, repository, phoneRepository, nil)
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
		CountAsUnread:  true,
		EventTimestamp: eventTimestamp,
	})

	require.NoError(t, err)
	assert.True(t, captured.CountAsUnread)
	assert.Equal(t, eventTimestamp, captured.EventTimestamp)
}

func TestUpdateThreadPreservesReadStateForOutboundActivity(t *testing.T) {
	var captured repositories.MessageThreadActivityUpdate
	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{ID: uuid.New()}, nil
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
	assert.False(t, captured.CountAsUnread)
}

func TestUpdateThreadUnarchivesArchivedInboundMessageWhenPhoneSettingEnabled(t *testing.T) {
	var captured repositories.MessageThreadActivityUpdate
	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{ID: uuid.New(), IsArchived: true}, nil
		},
		updateActivity: func(_ context.Context, params repositories.MessageThreadActivityUpdate) error {
			captured = params
			return nil
		},
	}
	phoneRepository := &messageThreadPhoneRepositoryStub{
		load: func(context.Context, entities.UserID, string) (*entities.Phone, error) {
			return &entities.Phone{UnarchiveThread: true}, nil
		},
	}

	service := newMessageThreadServiceWithPhoneForTest(repository, phoneRepository)
	err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
		UserID:         entities.UserID("user-id"),
		Owner:          "+18005550199",
		Contact:        "+18005550100",
		MessageID:      uuid.New(),
		Content:        "hello",
		Status:         entities.MessageStatusReceived,
		Timestamp:      time.Now().UTC(),
		CountAsUnread:  true,
		EventTimestamp: time.Now().UTC(),
	})

	require.NoError(t, err)
	assert.True(t, captured.Unarchive)
}

func TestUpdateThreadIgnoresPhoneLookupErrorsWhenCheckingUnarchive(t *testing.T) {
	var captured repositories.MessageThreadActivityUpdate
	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{ID: uuid.New(), IsArchived: true}, nil
		},
		updateActivity: func(_ context.Context, params repositories.MessageThreadActivityUpdate) error {
			captured = params
			return nil
		},
	}
	phoneRepository := &messageThreadPhoneRepositoryStub{
		load: func(context.Context, entities.UserID, string) (*entities.Phone, error) {
			return nil, stacktrace.NewError("load failed")
		},
	}

	service := newMessageThreadServiceWithPhoneForTest(repository, phoneRepository)
	err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
		UserID:         entities.UserID("user-id"),
		Owner:          "+18005550199",
		Contact:        "+18005550100",
		MessageID:      uuid.New(),
		Content:        "hello",
		Status:         entities.MessageStatusReceived,
		Timestamp:      time.Now().UTC(),
		CountAsUnread:  true,
		EventTimestamp: time.Now().UTC(),
	})

	require.NoError(t, err)
	assert.False(t, captured.Unarchive)
}

func TestCreateThreadSetsUnreadCountFromActivityDirection(t *testing.T) {
	tests := []struct {
		name              string
		status            entities.MessageStatus
		countAsUnread     bool
		wantUnreadCount   uint
		wantUnreadMessage bool
	}{
		{name: "inbound", status: entities.MessageStatusReceived, countAsUnread: true, wantUnreadCount: 1, wantUnreadMessage: true},
		{name: "outbound", status: entities.MessageStatusSent, countAsUnread: false, wantUnreadCount: 0, wantUnreadMessage: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stored *entities.MessageThread
			var unreadMessageID *uuid.UUID
			repository := &messageThreadRepositoryStub{
				loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
					return nil, stacktrace.PropagateWithCodef(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
				},
				store: func(_ context.Context, thread *entities.MessageThread, messageID *uuid.UUID) error {
					stored = thread
					unreadMessageID = messageID
					return nil
				},
			}

			messageID := uuid.New()
			service := newMessageThreadServiceForTest(repository)
			err := service.UpdateThread(context.Background(), MessageThreadUpdateParams{
				UserID:        entities.UserID("user-id"),
				Owner:         "+18005550199",
				Contact:       "+18005550100",
				MessageID:     messageID,
				Content:       "hello",
				Status:        test.status,
				Timestamp:     time.Now().UTC(),
				CountAsUnread: test.countAsUnread,
			})

			require.NoError(t, err)
			require.NotNil(t, stored)
			assert.Equal(t, test.wantUnreadCount, stored.UnreadCount)
			assert.False(t, stored.LastReadAt.IsZero())
			if test.wantUnreadMessage {
				require.NotNil(t, unreadMessageID)
				assert.Equal(t, messageID, *unreadMessageID)
			} else {
				assert.Nil(t, unreadMessageID)
			}
		})
	}
}

func TestUpdateStatusChangesOnlyRequestedState(t *testing.T) {
	threadID := uuid.New()
	unreadCount := uint(0)
	var captured repositories.MessageThreadStatusUpdate
	repository := &messageThreadRepositoryStub{
		updateStatus: func(_ context.Context, _ entities.UserID, _ uuid.UUID, params repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error) {
			captured = params
			return &entities.MessageThread{ID: threadID, IsArchived: true, UnreadCount: 0}, nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	thread, err := service.UpdateStatus(context.Background(), MessageThreadStatusParams{
		UserID:          entities.UserID("user-id"),
		MessageThreadID: threadID,
		UnreadCount:     &unreadCount,
	})

	require.NoError(t, err)
	assert.Nil(t, captured.IsArchived)
	assert.Same(t, &unreadCount, captured.UnreadCount)
	assert.False(t, captured.ReadAt.IsZero())
	assert.True(t, thread.IsArchived)
	assert.Zero(t, thread.UnreadCount)
}

func TestUpdateStatusPreservesNotFoundCode(t *testing.T) {
	repository := &messageThreadRepositoryStub{
		updateStatus: func(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error) {
			return nil, stacktrace.PropagateWithCodef(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
		},
	}

	service := newMessageThreadServiceForTest(repository)
	unreadCount := uint(0)
	_, err := service.UpdateStatus(context.Background(), MessageThreadStatusParams{
		UserID:          entities.UserID("user-id"),
		MessageThreadID: uuid.New(),
		UnreadCount:     &unreadCount,
	})

	assert.Equal(t, repositories.ErrCodeNotFound, stacktrace.GetCode(err))
}

func TestUpdateAfterDeletedMessageCleansUnreadLedgerForNonLastMessage(t *testing.T) {
	threadID := uuid.New()
	deletedMessageID := uuid.New()
	currentLastMessageID := uuid.New()
	previousMessageID := uuid.New()
	previousStatus := entities.MessageStatus(entities.MessageStatusDelivered)
	previousContent := "previous"
	var captured repositories.MessageThreadDeletedUpdate

	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{
				ID:            threadID,
				UserID:        entities.UserID("user-id"),
				Owner:         "+18005550199",
				Contact:       "+18005550100",
				LastMessageID: &currentLastMessageID,
			}, nil
		},
		updateAfterDelete: func(_ context.Context, params repositories.MessageThreadDeletedUpdate) error {
			captured = params
			return nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	err := service.UpdateAfterDeletedMessage(context.Background(), &events.MessageAPIDeletedPayload{
		MessageID:              deletedMessageID,
		UserID:                 entities.UserID("user-id"),
		Owner:                  "+18005550199",
		Contact:                "+18005550100",
		PreviousMessageID:      &previousMessageID,
		PreviousMessageStatus:  &previousStatus,
		PreviousMessageContent: &previousContent,
	})

	require.NoError(t, err)
	assert.Equal(t, threadID, captured.MessageThreadID)
	assert.Equal(t, entities.UserID("user-id"), captured.UserID)
	assert.Equal(t, deletedMessageID, captured.DeletedMessageID)
	assert.False(t, captured.UpdateLastMessage)
	require.NotNil(t, captured.LastMessageID)
	assert.Equal(t, previousMessageID, *captured.LastMessageID)
	require.NotNil(t, captured.LastMessageContent)
	assert.Equal(t, previousContent, *captured.LastMessageContent)
	assert.Equal(t, previousStatus, captured.LastMessageStatus)
}

func TestUpdateAfterDeletedMessageUpdatesLastMessageWhenDeletedMessageIsCurrent(t *testing.T) {
	threadID := uuid.New()
	deletedMessageID := uuid.New()
	previousMessageID := uuid.New()
	previousStatus := entities.MessageStatus(entities.MessageStatusDelivered)
	previousContent := "previous"
	var captured repositories.MessageThreadDeletedUpdate

	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{
				ID:            threadID,
				UserID:        entities.UserID("user-id"),
				Owner:         "+18005550199",
				Contact:       "+18005550100",
				LastMessageID: &deletedMessageID,
			}, nil
		},
		updateAfterDelete: func(_ context.Context, params repositories.MessageThreadDeletedUpdate) error {
			captured = params
			return nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	err := service.UpdateAfterDeletedMessage(context.Background(), &events.MessageAPIDeletedPayload{
		MessageID:              deletedMessageID,
		UserID:                 entities.UserID("user-id"),
		Owner:                  "+18005550199",
		Contact:                "+18005550100",
		PreviousMessageID:      &previousMessageID,
		PreviousMessageStatus:  &previousStatus,
		PreviousMessageContent: &previousContent,
	})

	require.NoError(t, err)
	assert.Equal(t, deletedMessageID, captured.DeletedMessageID)
	assert.True(t, captured.UpdateLastMessage)
	require.NotNil(t, captured.LastMessageID)
	assert.Equal(t, previousMessageID, *captured.LastMessageID)
	assert.Equal(t, previousStatus, captured.LastMessageStatus)
}

func TestUpdateAfterDeletedMessageDeletesThreadWhenNoPreviousMessageExists(t *testing.T) {
	threadID := uuid.New()
	deletedMessageID := uuid.New()
	deleted := false

	repository := &messageThreadRepositoryStub{
		loadByOwnerContact: func(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
			return &entities.MessageThread{
				ID:      threadID,
				UserID:  entities.UserID("user-id"),
				Owner:   "+18005550199",
				Contact: "+18005550100",
			}, nil
		},
		delete: func(_ context.Context, userID entities.UserID, id uuid.UUID) error {
			deleted = true
			assert.Equal(t, entities.UserID("user-id"), userID)
			assert.Equal(t, threadID, id)
			return nil
		},
		updateAfterDelete: func(context.Context, repositories.MessageThreadDeletedUpdate) error {
			t.Fatal("expected whole-thread delete instead of metadata update")
			return nil
		},
	}

	service := newMessageThreadServiceForTest(repository)
	err := service.UpdateAfterDeletedMessage(context.Background(), &events.MessageAPIDeletedPayload{
		MessageID: deletedMessageID,
		UserID:    entities.UserID("user-id"),
		Owner:     "+18005550199",
		Contact:   "+18005550100",
	})

	require.NoError(t, err)
	assert.True(t, deleted)
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
