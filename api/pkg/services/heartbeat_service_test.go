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
)

type heartbeatServiceRepository struct {
	repositories.HeartbeatRepository
	heartbeat *entities.Heartbeat
	err       error
}

func (repository *heartbeatServiceRepository) Last(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
	return repository.heartbeat, repository.err
}

func (repository *heartbeatServiceRepository) Store(_ context.Context, heartbeat *entities.Heartbeat) error {
	repository.heartbeat = heartbeat
	repository.err = nil
	return nil
}

type heartbeatServiceMonitorRepository struct {
	repositories.HeartbeatMonitorRepository
	monitor *entities.HeartbeatMonitor
	err     error
}

func (repository *heartbeatServiceMonitorRepository) Load(context.Context, entities.UserID, string) (*entities.HeartbeatMonitor, error) {
	return repository.monitor, repository.err
}

func (repository *heartbeatServiceMonitorRepository) UpdateQueueID(_ context.Context, _ uuid.UUID, queueID string) error {
	repository.monitor.QueueID = queueID
	repository.monitor.UpdatedAt = time.Now().UTC()
	return nil
}

func (repository *heartbeatServiceMonitorRepository) UpdatePhoneOnline(_ context.Context, _ entities.UserID, _ uuid.UUID, online bool) error {
	repository.monitor.PhoneOnline = online
	return nil
}

type heartbeatServiceQueue struct {
	phoneNotificationEventQueue
}

func (queue *heartbeatServiceQueue) Enqueue(ctx context.Context, task *PushQueueTask, timeout time.Duration) (string, error) {
	if _, err := queue.phoneNotificationEventQueue.Enqueue(ctx, task, timeout); err != nil {
		return "", err
	}
	return "queue-1", nil
}

func TestHeartbeatService_RestartsChecksAfterHistoryExpires(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	repository := &heartbeatServiceRepository{
		err: stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found"),
	}
	monitor := &entities.HeartbeatMonitor{
		ID:        uuid.New(),
		PhoneID:   uuid.New(),
		UserID:    entities.UserID("user-1"),
		Owner:     "+18005550199",
		QueueID:   "expired-check",
		UpdatedAt: time.Now().UTC(),
	}
	monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
	service := NewHeartbeatService(logger, tracer, repository, monitorRepository,
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com"}

	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Empty(t, queue.events)
	assert.Empty(t, monitor.QueueID)

	heartbeat, err := service.Store(context.Background(), HeartbeatStoreParams{
		UserID: monitor.UserID, Owner: monitor.Owner, Source: params.Source, Timestamp: time.Now().UTC(),
	})
	require.NoError(t, err)
	require.Len(t, queue.events, 2)
	assert.Equal(t, events.EventTypePhoneHeartbeatCheck, queue.events[0].Type())
	assert.Equal(t, events.EventTypePhoneHeartbeatOnline, queue.events[1].Type())
	assert.Equal(t, "queue-1", monitor.QueueID)
	assert.True(t, monitor.PhoneOnline)
	var payload events.PhoneHeartbeatCheckPayload
	require.NoError(t, queue.events[0].DataAs(&payload))
	assert.Equal(t, monitor.ID, payload.MonitorID)
	assert.Equal(t, monitor.PhoneID, payload.PhoneID)
	assert.Equal(t, monitor.UserID, payload.UserID)
	assert.Equal(t, monitor.Owner, payload.Owner)
	assert.WithinDuration(t, heartbeat.Timestamp.Add(heartbeatCheckInterval), payload.ScheduledAt, time.Millisecond)

	repository.heartbeat.Timestamp = time.Now().UTC().Add(-4*heartbeatCheckInterval - time.Minute)
	require.NoError(t, service.Monitor(context.Background(), params))
	require.Len(t, queue.events, 5)
	assert.Equal(t, events.EventTypePhoneHeartbeatOffline, queue.events[4].Type())
}

func TestHeartbeatServiceStore_OnlyRestartsInactiveChecks(t *testing.T) {
	for _, test := range []struct {
		name      string
		updatedAt time.Time
		wantCheck bool
	}{
		{name: "active monitor", updatedAt: time.Now().UTC()},
		{name: "stale monitor", updatedAt: time.Now().UTC().Add(-3 * time.Hour), wantCheck: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: entities.UserID("user-1"),
				Owner: "+18005550199", QueueID: "previous-check", UpdatedAt: test.updatedAt, PhoneOnline: true,
			}
			service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{},
				&heartbeatServiceMonitorRepository{monitor: monitor},
				NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))

			_, err := service.Store(context.Background(), HeartbeatStoreParams{
				UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com", Timestamp: time.Now().UTC(),
			})
			require.NoError(t, err)
			if test.wantCheck {
				require.Len(t, queue.events, 1)
				assert.Equal(t, events.EventTypePhoneHeartbeatCheck, queue.events[0].Type())
			} else {
				assert.Empty(t, queue.events)
			}
		})
	}
}

func TestHeartbeatServiceMonitor_StopsCheckingDeletedMonitor(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitorRepository := &heartbeatServiceMonitorRepository{
		err: stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "monitor not found"),
	}
	service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{}, monitorRepository,
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))

	require.NoError(t, service.Monitor(context.Background(), &HeartbeatMonitorParams{
		UserID: entities.UserID("user-1"), Owner: "+18005550199", Source: "https://example.com",
	}))
	assert.Empty(t, queue.events)
}
