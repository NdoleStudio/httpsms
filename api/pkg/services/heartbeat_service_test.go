package services

import (
	"context"
	"fmt"
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
	onLast    func()
}

func (repository *heartbeatServiceRepository) Last(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
	heartbeat, err := repository.heartbeat, repository.err
	if repository.onLast != nil {
		repository.onLast()
	}
	return heartbeat, err
}

func (repository *heartbeatServiceRepository) Store(_ context.Context, heartbeat *entities.Heartbeat) error {
	repository.heartbeat = heartbeat
	repository.err = nil
	return nil
}

type heartbeatServiceMonitorRepository struct {
	repositories.HeartbeatMonitorRepository
	monitor        *entities.HeartbeatMonitor
	err            error
	updateQueueErr error
}

func (repository *heartbeatServiceMonitorRepository) Load(context.Context, entities.UserID, string) (*entities.HeartbeatMonitor, error) {
	return repository.monitor, repository.err
}

func (repository *heartbeatServiceMonitorRepository) UpdateQueueID(_ context.Context, _ uuid.UUID, queueID string) error {
	if repository.updateQueueErr != nil {
		return repository.updateQueueErr
	}
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
	return fmt.Sprintf("queue-%d", len(queue.events)), nil
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
		UpdatedAt: time.Now().UTC().Add(-61 * 24 * time.Hour),
	}
	monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
	service := NewHeartbeatService(logger, tracer, repository, monitorRepository,
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com", QueueID: monitor.QueueID}

	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Empty(t, queue.events)
	assert.Equal(t, "expired-check", monitor.QueueID)

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
	params.QueueID = monitor.QueueID
	require.NoError(t, service.Monitor(context.Background(), params))
	require.Len(t, queue.events, 5)
	assert.Equal(t, events.EventTypePhoneHeartbeatOffline, queue.events[4].Type())
}

func TestHeartbeatServiceStore_OnlyRestartsInactiveChecks(t *testing.T) {
	for _, test := range []struct {
		name      string
		updatedAt time.Time
		queueID   string
		wantCheck bool
	}{
		{name: "active monitor", updatedAt: time.Now().UTC(), queueID: "previous-check"},
		{name: "stale monitor", updatedAt: time.Now().UTC().Add(-3 * time.Hour), queueID: "previous-check", wantCheck: true},
		{name: "recent monitor without task ID", updatedAt: time.Now().UTC()},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: entities.UserID("user-1"),
				Owner: "+18005550199", QueueID: test.queueID, UpdatedAt: test.updatedAt, PhoneOnline: true,
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

func TestHeartbeatServiceMonitor_StopsOnHeartbeatReadFailure(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), UserID: "user-1", Owner: "+18005550199", QueueID: "current-check",
	}
	service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{
		err: stacktrace.NewError("heartbeat read failed"),
	}, &heartbeatServiceMonitorRepository{monitor: monitor},
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))

	err := service.Monitor(context.Background(), &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, QueueID: monitor.QueueID,
	})

	require.NoError(t, err)
	assert.Equal(t, "current-check", monitor.QueueID)
	assert.Empty(t, queue.events)
}

func TestHeartbeatServiceMonitor_RecoversFailedQueueIDSavesOnHeartbeat(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1", Owner: "+18005550199",
		QueueID: "current-check", PhoneOnline: true, UpdatedAt: time.Now().UTC().Add(-3 * time.Hour),
	}
	monitorRepository := &heartbeatServiceMonitorRepository{
		monitor: monitor, updateQueueErr: stacktrace.NewError("queue update failed"),
	}
	service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{
		heartbeat: &entities.Heartbeat{Timestamp: time.Now().UTC()},
	}, monitorRepository, NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, QueueID: monitor.QueueID, Source: "https://example.com",
	}

	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Equal(t, "current-check", monitor.QueueID)
	require.Len(t, queue.events, 1)

	monitorRepository.updateQueueErr = nil
	_, err := service.Store(context.Background(), HeartbeatStoreParams{
		UserID: monitor.UserID, Owner: monitor.Owner, Source: params.Source, Timestamp: time.Now().UTC(),
	})
	require.NoError(t, err)
	assert.Equal(t, "queue-2", monitor.QueueID)
	require.Len(t, queue.events, 2)

	params.QueueID = "queue-1"
	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Equal(t, "queue-2", monitor.QueueID)
	assert.Len(t, queue.events, 2)

	params.QueueID = "queue-2"
	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Equal(t, "queue-3", monitor.QueueID)
	assert.Len(t, queue.events, 3)
}

func TestHeartbeatServiceMonitor_PreservesConcurrentReplacement(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1", Owner: "+18005550199",
		QueueID: "old-check", UpdatedAt: time.Now().UTC().Add(-61 * 24 * time.Hour), PhoneOnline: true,
	}
	repository := &heartbeatServiceRepository{
		err: stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found"),
	}
	service := NewHeartbeatService(logger, tracer, repository,
		&heartbeatServiceMonitorRepository{monitor: monitor},
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	repository.onLast = func() {
		repository.onLast = nil
		_, err := service.Store(context.Background(), HeartbeatStoreParams{
			UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com", Timestamp: time.Now().UTC(),
		})
		require.NoError(t, err)
	}
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, QueueID: "old-check", Source: "https://example.com",
	}

	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Equal(t, "queue-1", monitor.QueueID)
	require.Len(t, queue.events, 1)

	params.QueueID = "queue-1"
	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Equal(t, "queue-2", monitor.QueueID)
	assert.Len(t, queue.events, 2)
}
