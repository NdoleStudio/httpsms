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
	afterLast func()
}

func (repository *heartbeatServiceRepository) Last(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
	heartbeat, err := repository.heartbeat, repository.err
	if repository.afterLast != nil {
		hook := repository.afterLast
		repository.afterLast = nil
		hook()
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
	monitor    *entities.HeartbeatMonitor
	err        error
	afterLoad  func()
	beforeSwap func()
}

func (repository *heartbeatServiceMonitorRepository) Load(context.Context, entities.UserID, string) (*entities.HeartbeatMonitor, error) {
	if repository.monitor == nil {
		return nil, repository.err
	}
	monitor := *repository.monitor
	if repository.afterLoad != nil {
		hook := repository.afterLoad
		repository.afterLoad = nil
		hook()
	}
	return &monitor, repository.err
}

func (repository *heartbeatServiceMonitorRepository) CompareAndSwapQueueID(_ context.Context, monitorID uuid.UUID, expected, queueID string) (bool, error) {
	if repository.beforeSwap != nil {
		hook := repository.beforeSwap
		repository.beforeSwap = nil
		hook()
	}
	if repository.monitor.ID != monitorID || repository.monitor.QueueID != expected {
		return false, nil
	}
	repository.monitor.QueueID = queueID
	repository.monitor.UpdatedAt = time.Now().UTC()
	return true, nil
}

func (repository *heartbeatServiceMonitorRepository) UpdateQueueID(_ context.Context, _ uuid.UUID, queueID string) error {
	repository.monitor.QueueID = queueID
	repository.monitor.UpdatedAt = time.Now().UTC()
	return nil
}

func (repository *heartbeatServiceMonitorRepository) UpdatePhoneOnline(_ context.Context, _ entities.UserID, _ uuid.UUID, online bool) error {
	repository.monitor.PhoneOnline = online
	repository.monitor.UpdatedAt = time.Now().UTC()
	return nil
}

type heartbeatServiceQueue struct {
	phoneNotificationEventQueue
	err error
}

func (queue *heartbeatServiceQueue) Enqueue(ctx context.Context, task *PushQueueTask, timeout time.Duration) (string, error) {
	if queue.err != nil {
		return "", queue.err
	}
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
	assert.True(t, monitor.PhoneOnline)
	var payload events.PhoneHeartbeatCheckPayload
	require.NoError(t, queue.events[0].DataAs(&payload))
	assert.Equal(t, payload.CheckID, monitor.QueueID)
	assert.NotEmpty(t, payload.CheckID)
	assert.Equal(t, monitor.ID, payload.MonitorID)
	assert.Equal(t, monitor.PhoneID, payload.PhoneID)
	assert.Equal(t, monitor.UserID, payload.UserID)
	assert.Equal(t, monitor.Owner, payload.Owner)
	assert.WithinDuration(t, heartbeat.Timestamp.Add(heartbeatCheckInterval), payload.ScheduledAt, time.Millisecond)

	repository.heartbeat.Timestamp = time.Now().UTC().Add(-4*heartbeatCheckInterval - time.Minute)
	params.CheckID = payload.CheckID
	require.NoError(t, service.Monitor(context.Background(), params))
	require.Len(t, queue.events, 5)
	assert.Equal(t, events.EventTypePhoneHeartbeatOffline, queue.events[3].Type())
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

func TestHeartbeatServiceStore_ReturnsHeartbeatWhenRestartFails(t *testing.T) {
	for _, test := range []struct {
		name   string
		online bool
		stale  bool
	}{
		{name: "online phone", online: true},
		{name: "offline phone"},
		{name: "stale offline phone", stale: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{err: stacktrace.NewError("queue unavailable")}
			repository := &heartbeatServiceRepository{}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: entities.UserID("user-1"),
				Owner: "+18005550199", UpdatedAt: time.Now().UTC(), PhoneOnline: test.online,
			}
			if test.stale {
				monitor.QueueID = "stale-check"
				monitor.UpdatedAt = time.Now().UTC().Add(-3 * time.Hour)
			}
			service := NewHeartbeatService(logger, tracer, repository,
				&heartbeatServiceMonitorRepository{monitor: monitor},
				NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))

			heartbeat, err := service.Store(context.Background(), HeartbeatStoreParams{
				UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com", Timestamp: time.Now().UTC(),
			})

			require.NoError(t, err)
			require.NotNil(t, heartbeat)
			assert.Same(t, repository.heartbeat, heartbeat)
			assert.Empty(t, monitor.QueueID)
			assert.True(t, monitor.PhoneOnline)
			queue.err = nil
			_, err = service.Store(context.Background(), HeartbeatStoreParams{
				UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com", Timestamp: time.Now().UTC(),
			})
			require.NoError(t, err)
			require.Len(t, queue.events, 1)
			assert.Equal(t, events.EventTypePhoneHeartbeatCheck, queue.events[0].Type())
		})
	}
}

func TestHeartbeatServiceStore_ClaimsOverlappingRestarts(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, UpdatedAt: time.Now().UTC(),
	}
	monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
	service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{}, monitorRepository,
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := HeartbeatStoreParams{
		UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com", Timestamp: time.Now().UTC(),
	}
	monitorRepository.afterLoad = func() {
		_, err := service.Store(context.Background(), params)
		require.NoError(t, err)
	}
	_, err := service.Store(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, queue.events, 1)
}

func TestHeartbeatServiceMonitor_RestartsHeartbeatArrivingDuringStop(t *testing.T) {
	for _, arrival := range []string{"before clear", "after clear", "restart wins"} {
		t.Run(arrival, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{}
			repository := &heartbeatServiceRepository{
				err: stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found"),
			}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
				Owner: "+18005550199", PhoneOnline: true, QueueID: "current-check", UpdatedAt: time.Now().UTC(),
			}
			monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
			if arrival == "restart wins" {
				monitor.UpdatedAt = time.Now().UTC().Add(-3 * time.Hour)
			}
			service := NewHeartbeatService(logger, tracer, repository, monitorRepository,
				NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
			store := func() {
				_, err := service.Store(context.Background(), HeartbeatStoreParams{
					UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com", Timestamp: time.Now().UTC(),
				})
				require.NoError(t, err)
			}
			if arrival != "after clear" {
				repository.afterLast = store
			} else {
				repository.afterLast = func() { repository.afterLast = store }
			}
			require.NoError(t, service.Monitor(context.Background(), &HeartbeatMonitorParams{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
				Source: "https://example.com", CheckID: monitor.QueueID,
			}))
			require.Len(t, queue.events, 1)
			var payload events.PhoneHeartbeatCheckPayload
			require.NoError(t, queue.events[0].DataAs(&payload))
			assert.Equal(t, payload.CheckID, monitor.QueueID)
		})
	}
}

func TestHeartbeatServiceMonitor_RejectsSupersededAndOverlappingChecks(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, QueueID: "current-check", UpdatedAt: time.Now().UTC(),
	}
	repository := &heartbeatServiceRepository{heartbeat: &entities.Heartbeat{
		Timestamp: time.Now().UTC().Add(-4*heartbeatCheckInterval - time.Minute),
	}}
	monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
	service := NewHeartbeatService(logger, tracer, repository, monitorRepository,
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
		Source: "https://example.com", CheckID: monitor.QueueID,
	}
	monitorRepository.beforeSwap = func() {
		copy := params
		require.NoError(t, service.Monitor(context.Background(), &copy))
	}
	require.Error(t, service.Monitor(context.Background(), &params))
	require.Len(t, queue.events, 3)
	assert.Equal(t, events.PhoneHeartbeatMissed, queue.events[0].Type())
	assert.Equal(t, events.EventTypePhoneHeartbeatOffline, queue.events[1].Type())
	assert.Equal(t, events.EventTypePhoneHeartbeatCheck, queue.events[2].Type())
	require.NoError(t, service.Monitor(context.Background(), &params))
	assert.Len(t, queue.events, 3)
}

func TestHeartbeatServiceMonitor_RetriesFailedSuccessor(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{err: stacktrace.NewError("queue unavailable")}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, QueueID: "current-check", UpdatedAt: time.Now().UTC(),
	}
	service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{heartbeat: &entities.Heartbeat{
		Timestamp: time.Now().UTC(),
	}}, &heartbeatServiceMonitorRepository{monitor: monitor},
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
		Source: "https://example.com", CheckID: monitor.QueueID,
	}
	require.Error(t, service.Monitor(context.Background(), params))
	assert.Equal(t, params.CheckID, monitor.QueueID)
	queue.err = nil
	require.NoError(t, service.Monitor(context.Background(), params))
	require.Len(t, queue.events, 1)
}

func TestHeartbeatServiceMonitor_RecoversInterruptedClaims(t *testing.T) {
	for _, test := range []struct {
		name      string
		marker    string
		expired   bool
		wantRetry bool
		wantCheck bool
	}{
		{name: "running lease active", marker: "running:current-check:claim", wantRetry: true},
		{name: "running lease expired", marker: "running:current-check:claim", expired: true, wantCheck: true},
		{name: "pending lease active", marker: "pending:current-check:next-check", wantRetry: true},
		{name: "pending lease expired", marker: "pending:current-check:next-check", expired: true, wantCheck: true},
		{name: "queued pending check", marker: "pending:previous-check:current-check", wantCheck: true},
		{name: "unrelated expired running check", marker: "running:other-check:claim", expired: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
				Owner: "+18005550199", PhoneOnline: true, QueueID: test.marker, UpdatedAt: time.Now().UTC(),
			}
			if test.expired {
				monitor.UpdatedAt = time.Now().UTC().Add(-2 * heartbeatClaimLease)
			}
			service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{heartbeat: &entities.Heartbeat{
				Timestamp: time.Now().UTC(),
			}}, &heartbeatServiceMonitorRepository{monitor: monitor},
				NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
			err := service.Monitor(context.Background(), &HeartbeatMonitorParams{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
				Source: "https://example.com", CheckID: "current-check",
			})
			if test.wantRetry {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if test.wantCheck {
				require.Len(t, queue.events, 1)
			} else {
				assert.Empty(t, queue.events)
				assert.Equal(t, test.marker, monitor.QueueID)
			}
		})
	}
}

func TestHeartbeatServiceMonitor_RetriesHistoryRecheckFailure(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	repository := &heartbeatServiceRepository{
		err: stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found"),
	}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, QueueID: "current-check", UpdatedAt: time.Now().UTC(),
	}
	repository.afterLast = func() { repository.err = stacktrace.NewError("database unavailable") }
	service := NewHeartbeatService(logger, tracer, repository, &heartbeatServiceMonitorRepository{monitor: monitor},
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
		Source: "https://example.com", CheckID: monitor.QueueID,
	}
	require.Error(t, service.Monitor(context.Background(), params))
	assert.Equal(t, params.CheckID, monitor.QueueID)
	assert.Empty(t, queue.events)
	repository.heartbeat = &entities.Heartbeat{Timestamp: time.Now().UTC()}
	repository.err = nil
	require.NoError(t, service.Monitor(context.Background(), params))
	require.Len(t, queue.events, 1)
}
