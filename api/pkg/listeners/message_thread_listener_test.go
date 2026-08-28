package listeners

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageThreadListenerMarksInboundMessageUnread(t *testing.T) {
	repository, routes := newMessageThreadListenerForTest()
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource("/v1/messages/phone-received")
	event.SetType(events.EventTypeMessagePhoneReceived)
	event.SetTime(time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC))
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.MessagePhoneReceivedPayload{
		MessageID: uuid.New(),
		UserID:    entities.UserID("user-id"),
		Owner:     "+18005550199",
		Contact:   "+18005550100",
		Content:   "hello",
		Timestamp: time.Date(2026, 7, 18, 6, 59, 0, 0, time.UTC),
	}))

	err := routes[events.EventTypeMessagePhoneReceived](context.Background(), event)

	require.NoError(t, err)
	assert.True(t, repository.activity.CountAsUnread)
}

func TestMessageThreadListenerMarksMissedCallUnread(t *testing.T) {
	repository, routes := newMessageThreadListenerForTest()
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource("/v1/messages/call-missed")
	event.SetType(events.MessageCallMissed)
	event.SetTime(time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC))
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.MessageCallMissedPayload{
		MessageID: uuid.New(),
		UserID:    entities.UserID("user-id"),
		Owner:     "+18005550199",
		Contact:   "+18005550100",
		Timestamp: time.Date(2026, 7, 18, 6, 59, 0, 0, time.UTC),
	}))
	require.Contains(t, routes, events.MessageCallMissed)

	err := routes[events.MessageCallMissed](context.Background(), event)

	require.NoError(t, err)
	assert.True(t, repository.activity.CountAsUnread)
	assert.Equal(t, "Missed phone call", repository.activity.Content)
}

func TestMessageThreadListenerDeletesNonLastUnreadMessage(t *testing.T) {
	repository, routes := newMessageThreadListenerForTest()
	deletedMessageID := uuid.New()
	previousMessageID := uuid.New()
	previousStatus := entities.MessageStatus(entities.MessageStatusDelivered)
	previousContent := "previous"
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource("/v1/messages/deleted")
	event.SetType(events.MessageAPIDeleted)
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.MessageAPIDeletedPayload{
		MessageID:              deletedMessageID,
		UserID:                 entities.UserID("user-id"),
		Owner:                  "+18005550199",
		Contact:                "+18005550100",
		PreviousMessageID:      &previousMessageID,
		PreviousMessageStatus:  &previousStatus,
		PreviousMessageContent: &previousContent,
	}))

	err := routes[events.MessageAPIDeleted](context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, deletedMessageID, repository.deletedUpdate.DeletedMessageID)
	assert.Equal(t, "+18005550199", repository.deletedUpdate.Owner)
	assert.Equal(t, "+18005550100", repository.deletedUpdate.Contact)
	require.NotNil(t, repository.deletedUpdate.LastMessageID)
	assert.Equal(t, previousMessageID, *repository.deletedUpdate.LastMessageID)
}

func newMessageThreadListenerForTest() (*listenerMessageThreadRepository, map[string]events.EventListener) {
	repository := &listenerMessageThreadRepository{}
	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	service := services.NewMessageThreadService(logger, tracer, repository, nil, nil, nil)
	_, routes := NewMessageThreadListener(logger, tracer, service)
	return repository, routes
}
