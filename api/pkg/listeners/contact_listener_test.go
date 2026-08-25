package listeners

import (
	"context"
	"errors"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contactDeletionServiceStub struct {
	userIDs []entities.UserID
	err     error
}

func (service *contactDeletionServiceStub) DeleteAllForUser(_ context.Context, userID entities.UserID) error {
	service.userIDs = append(service.userIDs, userID)
	return service.err
}

func TestContactListenerDeletesContactsForDeletedUser(t *testing.T) {
	service := &contactDeletionServiceStub{}
	routes := newContactListenerRoutes(t, service)
	event := deletedUserContactEvent(t, entities.UserID("user-id"))

	err := routes[events.UserAccountDeleted](context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, []entities.UserID{"user-id"}, service.userIDs)
}

func TestContactListenerWrapsDeleteError(t *testing.T) {
	service := &contactDeletionServiceStub{err: errors.New("delete contacts boom")}
	routes := newContactListenerRoutes(t, service)
	event := deletedUserContactEvent(t, entities.UserID("user-id"))

	err := routes[events.UserAccountDeleted](context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete contacts boom")
}

func newContactListenerRoutes(t *testing.T, service *contactDeletionServiceStub) map[string]events.EventListener {
	t.Helper()

	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	_, routes := NewContactListener(logger, tracer, service)
	require.Contains(t, routes, events.UserAccountDeleted)
	return routes
}

func deletedUserContactEvent(t *testing.T, userID entities.UserID) cloudevents.Event {
	t.Helper()

	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource("/v1/users")
	event.SetType(events.UserAccountDeleted)
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.UserAccountDeletedPayload{
		UserID: userID,
	}))
	return event
}
