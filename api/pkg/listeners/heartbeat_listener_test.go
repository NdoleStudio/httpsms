package listeners

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type heartbeatListenerRepository struct {
	repositories.HeartbeatRepository
	lastCalls int
}

func (repository *heartbeatListenerRepository) Last(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
	repository.lastCalls++
	return &entities.Heartbeat{Timestamp: time.Now().UTC()}, nil
}

type heartbeatListenerMonitorRepository struct {
	repositories.HeartbeatMonitorRepository
	monitor *entities.HeartbeatMonitor
	err     error
}

func (repository *heartbeatListenerMonitorRepository) Load(context.Context, entities.UserID, string) (*entities.HeartbeatMonitor, error) {
	return repository.monitor, repository.err
}

func TestHeartbeatListener_DoesNotRescheduleWhenMonitorCannotBeLoaded(t *testing.T) {
	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	repository := &heartbeatListenerRepository{}
	queue := &heartbeatListenerQueue{}
	service := services.NewHeartbeatService(logger, tracer, repository,
		&heartbeatListenerMonitorRepository{
			monitor: &entities.HeartbeatMonitor{}, err: errors.New("database unavailable"),
		},
		services.NewEventDispatcher(logger, tracer, nil, queue, services.PushQueueConfig{}))
	_, routes := NewHeartbeatListener(logger, tracer, service)
	event := cloudevents.NewEvent()
	event.SetID("event-123")
	event.SetSource("https://example.com")
	event.SetType(events.EventTypePhoneHeartbeatCheck)
	ctx := context.WithValue(context.Background(), events.ExtensionQueueID, "old-check")
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.PhoneHeartbeatCheckPayload{
		UserID: "user-1", Owner: "+18005550199",
	}))

	err := routes[event.Type()](ctx, event)

	require.ErrorContains(t, err, "database unavailable")
	assert.Zero(t, repository.lastCalls)
	assert.Empty(t, queue.events)
}

func (repository *heartbeatListenerMonitorRepository) UpdateQueueID(_ context.Context, _ uuid.UUID, queueID string) error {
	repository.monitor.QueueID = queueID
	return nil
}

type heartbeatListenerQueue struct {
	events []cloudevents.Event
}

func (queue *heartbeatListenerQueue) Enqueue(_ context.Context, task *services.PushQueueTask, _ time.Duration) (string, error) {
	var event cloudevents.Event
	if err := json.Unmarshal(task.Body, &event); err != nil {
		return "", err
	}
	queue.events = append(queue.events, event)
	return "next-check", nil
}

func TestHeartbeatListener_ChecksOnlyCurrentQueueTask(t *testing.T) {
	for _, test := range []struct {
		name        string
		storedQueue string
		eventQueue  string
		wantChecks  int
	}{
		{name: "current task", storedQueue: "current-check", eventQueue: "current-check", wantChecks: 1},
		{name: "superseded task", storedQueue: "current-check", eventQueue: "old-check"},
		{name: "missing task ID", storedQueue: "current-check"},
		{name: "inactive monitor", eventQueue: "old-check"},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := &noopListenerLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1", Owner: "+18005550199",
				QueueID: test.storedQueue, PhoneOnline: true,
			}
			repository := &heartbeatListenerRepository{}
			queue := &heartbeatListenerQueue{}
			service := services.NewHeartbeatService(logger, tracer, repository,
				&heartbeatListenerMonitorRepository{monitor: monitor},
				services.NewEventDispatcher(logger, tracer, nil, queue, services.PushQueueConfig{}))
			_, routes := NewHeartbeatListener(logger, tracer, service)

			event := cloudevents.NewEvent()
			event.SetID(uuid.NewString())
			event.SetSource("https://example.com")
			event.SetType(events.EventTypePhoneHeartbeatCheck)
			ctx := context.Background()
			if test.eventQueue != "" {
				ctx = context.WithValue(ctx, events.ExtensionQueueID, test.eventQueue)
			}
			require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.PhoneHeartbeatCheckPayload{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: uuid.New(), PhoneID: uuid.New(),
			}))

			require.NoError(t, routes[event.Type()](ctx, event))
			assert.Equal(t, test.wantChecks, repository.lastCalls)
			require.Len(t, queue.events, test.wantChecks)
			if test.wantChecks == 0 {
				assert.Equal(t, test.storedQueue, monitor.QueueID)
				return
			}
			assert.Equal(t, "next-check", monitor.QueueID)
			var payload events.PhoneHeartbeatCheckPayload
			require.NoError(t, queue.events[0].DataAs(&payload))
			assert.Equal(t, monitor.ID, payload.MonitorID)
			assert.Equal(t, monitor.PhoneID, payload.PhoneID)

			// Redelivery of the previous task must not start another stream.
			require.NoError(t, routes[event.Type()](ctx, event))
			assert.Equal(t, 1, repository.lastCalls)
			assert.Len(t, queue.events, 1)
		})
	}
}
