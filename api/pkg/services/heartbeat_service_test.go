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
	swapErr    error
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
	if repository.swapErr != nil {
		err := repository.swapErr
		repository.swapErr = nil
		return false, err
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
	err          error
	afterEnqueue func()
}

func (queue *heartbeatServiceQueue) Enqueue(ctx context.Context, task *PushQueueTask, timeout time.Duration) (string, error) {
	if queue.err != nil {
		return "", queue.err
	}
	if _, err := queue.phoneNotificationEventQueue.Enqueue(ctx, task, timeout); err != nil {
		return "", err
	}
	if queue.afterEnqueue != nil {
		hook := queue.afterEnqueue
		queue.afterEnqueue = nil
		hook()
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
	assert.Equal(t, "check:"+payload.CheckID, monitor.QueueID)
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
		queueID   string
	}{
		{name: "active monitor", updatedAt: time.Now().UTC()},
		{name: "stale monitor", updatedAt: time.Now().UTC().Add(-3 * time.Hour), wantCheck: true},
		{name: "fresh restart claim", updatedAt: time.Now().UTC(), queueID: "pending::new-check"},
		{name: "expired restart claim", updatedAt: time.Now().UTC().Add(-2 * heartbeatClaimLease), queueID: "pending::new-check", wantCheck: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: entities.UserID("user-1"),
				Owner: "+18005550199", QueueID: "previous-check", UpdatedAt: test.updatedAt, PhoneOnline: true,
			}
			if test.queueID != "" {
				monitor.QueueID = test.queueID
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
			err := service.Monitor(context.Background(), &HeartbeatMonitorParams{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
				Source: "https://example.com", CheckID: monitor.QueueID,
			})
			if arrival == "restart wins" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, queue.events, 1)
			var payload events.PhoneHeartbeatCheckPayload
			require.NoError(t, queue.events[0].DataAs(&payload))
			assert.Equal(t, "check:"+payload.CheckID, monitor.QueueID)
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
		name           string
		marker         string
		expired        bool
		wantRetry      bool
		wantCheck      bool
		missingHistory bool
		legacy         bool
	}{
		{name: "running lease active", marker: "running:current-check:claim", wantRetry: true},
		{name: "running lease expired", marker: "running:current-check:claim", expired: true, wantCheck: true},
		{name: "pending lease active", marker: "pending:current-check:next-check", wantRetry: true},
		{name: "pending lease expired", marker: "pending:current-check:next-check", expired: true, wantCheck: true},
		{name: "restart-only lease active", marker: "pending::next-check", wantRetry: true},
		{name: "restart-only lease expired", marker: "pending::next-check", expired: true, wantCheck: true},
		{name: "queued pending check", marker: "pending:previous-check:current-check", wantCheck: true},
		{name: "interrupted stop", wantCheck: true},
		{name: "interrupted stop without history", missingHistory: true},
		{name: "legacy running recovery", marker: "running:legacy:claim", expired: true, wantCheck: true, legacy: true},
		{name: "legacy pending recovery", marker: "pending:legacy:next-check", expired: true, wantCheck: true, legacy: true},
		{name: "legacy waits for restart-only claim", marker: "pending::next-check", wantRetry: true, legacy: true},
		{name: "legacy recovers abandoned restart-only claim", marker: "pending::next-check", expired: true, wantCheck: true, legacy: true},
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
			repository := &heartbeatServiceRepository{heartbeat: &entities.Heartbeat{
				Timestamp: time.Now().UTC(),
			}}
			if test.missingHistory {
				repository.err = stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found")
			}
			service := NewHeartbeatService(logger, tracer, repository, &heartbeatServiceMonitorRepository{monitor: monitor},
				NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
			params := &HeartbeatMonitorParams{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
				Source: "https://example.com", CheckID: "current-check",
			}
			if test.legacy {
				params.CheckID = ""
			}
			err := service.Monitor(context.Background(), params)
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
	assert.Empty(t, monitor.QueueID)
	assert.Empty(t, queue.events)
	repository.heartbeat = &entities.Heartbeat{Timestamp: time.Now().UTC()}
	repository.err = nil
	require.NoError(t, service.Monitor(context.Background(), params))
	require.Len(t, queue.events, 1)
}

func TestHeartbeatServiceMonitor_RetriesPendingRestartAfterStop(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, QueueID: "current-check", UpdatedAt: time.Now().UTC(),
	}
	repository := &heartbeatServiceRepository{
		err: stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found"),
	}
	repository.afterLast = func() {
		repository.heartbeat = &entities.Heartbeat{Timestamp: time.Now().UTC()}
		repository.err = nil
		repository.afterLast = func() {
			monitor.QueueID = "pending::concurrent-restart"
			monitor.UpdatedAt = time.Now().UTC()
		}
	}
	service := NewHeartbeatService(logger, tracer, repository, &heartbeatServiceMonitorRepository{monitor: monitor},
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
		Source: "https://example.com", CheckID: monitor.QueueID,
	}
	require.Error(t, service.Monitor(context.Background(), params))
	assert.Equal(t, "pending::concurrent-restart", monitor.QueueID)
	assert.Empty(t, queue.events)
	monitor.UpdatedAt = time.Now().UTC().Add(-2 * heartbeatClaimLease)
	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Len(t, queue.events, 1)
}

func TestHeartbeatServiceMonitor_RetriesUncommittedSuccessorOnClaimLoss(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, QueueID: "current-check", UpdatedAt: time.Now().UTC(),
	}
	monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
	monitorRepository.beforeSwap = func() {
		monitorRepository.beforeSwap = func() {
			monitor.QueueID = "pending::concurrent-restart"
			monitor.UpdatedAt = time.Now().UTC()
		}
	}
	service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{heartbeat: &entities.Heartbeat{
		Timestamp: time.Now().UTC(),
	}}, monitorRepository, NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
		Source: "https://example.com", CheckID: monitor.QueueID,
	}
	require.Error(t, service.Monitor(context.Background(), params))
	assert.Equal(t, "pending::concurrent-restart", monitor.QueueID)
	assert.Empty(t, queue.events)
	monitor.UpdatedAt = time.Now().UTC().Add(-2 * heartbeatClaimLease)
	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Len(t, queue.events, 1)
}

func TestHeartbeatServiceMonitor_LegacyMigrationIsOneWay(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, QueueID: "legacy-task", UpdatedAt: time.Now().UTC(),
	}
	service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{heartbeat: &entities.Heartbeat{
		Timestamp: time.Now().UTC(),
	}}, &heartbeatServiceMonitorRepository{monitor: monitor},
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID, Source: "https://example.com",
	}
	require.NoError(t, service.Monitor(context.Background(), params))
	require.Len(t, queue.events, 1)
	var payload events.PhoneHeartbeatCheckPayload
	require.NoError(t, queue.events[0].DataAs(&payload))
	assert.Equal(t, "check:"+payload.CheckID, monitor.QueueID)
	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Len(t, queue.events, 1)
	params.CheckID = payload.CheckID
	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Len(t, queue.events, 2)
}

func TestHeartbeatServiceMonitor_RetriesLostStopOwnership(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, QueueID: "current-check", UpdatedAt: time.Now().UTC(),
	}
	repository := &heartbeatServiceRepository{
		err: stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found"),
	}
	monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
	monitorRepository.beforeSwap = func() { monitor.QueueID = "" }
	service := NewHeartbeatService(logger, tracer, repository, monitorRepository,
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
		Source: "https://example.com", CheckID: monitor.QueueID,
	}
	require.Error(t, service.Monitor(context.Background(), params))
	assert.Empty(t, queue.events)
	monitor.QueueID = params.CheckID
	repository.err = nil
	repository.heartbeat = &entities.Heartbeat{Timestamp: time.Now().UTC()}
	require.NoError(t, service.Monitor(context.Background(), params))
	assert.Len(t, queue.events, 1)
}

func TestHeartbeatServiceMonitor_DoesNotResurrectAcknowledgedStop(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	notFound := stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found")
	repository := &heartbeatServiceRepository{err: notFound}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
		Owner: "+18005550199", PhoneOnline: true, QueueID: "check:current-check", UpdatedAt: time.Now().UTC(),
	}
	service := NewHeartbeatService(logger, tracer, repository, &heartbeatServiceMonitorRepository{monitor: monitor},
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	params := HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
		Source: "https://example.com", CheckID: "current-check",
	}
	repository.afterLast = func() {
		repository.err = stacktrace.NewError("history recheck unavailable")
		repository.afterLast = func() {
			repository.err = notFound
			duplicate := params
			require.NoError(t, service.Monitor(context.Background(), &duplicate))
		}
	}
	require.Error(t, service.Monitor(context.Background(), &params))
	assert.Empty(t, queue.events)
	assert.True(t, monitor.RequiresCheck(), "an acknowledged stop must remain restartable")
	_, err := service.Store(context.Background(), HeartbeatStoreParams{
		UserID: monitor.UserID, Owner: monitor.Owner, Source: params.Source, Timestamp: time.Now().UTC(),
	})
	require.NoError(t, err)
	require.Len(t, queue.events, 1)
	assert.Equal(t, events.EventTypePhoneHeartbeatCheck, queue.events[0].Type())
}

func TestHeartbeatServiceMonitor_RetriesFailedFinalization(t *testing.T) {
	for _, failure := range []string{"database error", "lease recovered by interrupted worker"} {
		t.Run(failure, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
				Owner: "+18005550199", PhoneOnline: true, QueueID: "check:current-check", UpdatedAt: time.Now().UTC(),
			}
			monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
			service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{
				heartbeat: &entities.Heartbeat{Timestamp: time.Now().UTC()},
			}, monitorRepository, NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
			params := HeartbeatMonitorParams{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
				Source: "https://example.com", CheckID: "current-check",
			}
			queue.afterEnqueue = func() {
				if failure == "database error" {
					monitorRepository.swapErr = stacktrace.NewError("finalization unavailable")
				} else {
					monitor.QueueID = "running:current-check:recovered-worker"
					monitor.UpdatedAt = time.Now().UTC()
				}
			}
			require.Error(t, service.Monitor(context.Background(), &params), "incoming task must remain retryable")
			require.Len(t, queue.events, 1)
			monitor.UpdatedAt = time.Now().UTC().Add(-2 * heartbeatClaimLease)
			require.NoError(t, service.Monitor(context.Background(), &params))
			require.Len(t, queue.events, 2)
			var successor events.PhoneHeartbeatCheckPayload
			require.NoError(t, queue.events[1].DataAs(&successor))
			assert.Equal(t, "check:"+successor.CheckID, monitor.QueueID)
			var abandoned events.PhoneHeartbeatCheckPayload
			require.NoError(t, queue.events[0].DataAs(&abandoned))
			params.CheckID = abandoned.CheckID
			require.NoError(t, service.Monitor(context.Background(), &params))
			assert.Len(t, queue.events, 2, "superseded queued task must not create another chain")
		})
	}
}

func TestHeartbeatServiceMonitor_RejectsRestoredIncomingAsSuccessor(t *testing.T) {
	for _, path := range []string{"scheduling claim loss", "post-stop recheck"} {
		t.Run(path, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
				Owner: "+18005550199", PhoneOnline: true, QueueID: "check:current-check", UpdatedAt: time.Now().UTC(),
			}
			repository := &heartbeatServiceRepository{heartbeat: &entities.Heartbeat{Timestamp: time.Now().UTC()}}
			monitorRepository := &heartbeatServiceMonitorRepository{monitor: monitor}
			restoreIncoming := func() {
				monitor.QueueID = "check:current-check"
				monitor.UpdatedAt = time.Now().UTC()
			}
			if path == "scheduling claim loss" {
				monitorRepository.beforeSwap = func() { monitorRepository.beforeSwap = restoreIncoming }
			} else {
				repository.err = stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "heartbeat not found")
				repository.afterLast = func() {
					repository.err = nil
					repository.afterLast = restoreIncoming
				}
			}

			service := NewHeartbeatService(logger, tracer, repository, monitorRepository,
				NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
			params := &HeartbeatMonitorParams{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
				Source: "https://example.com", CheckID: "current-check",
			}
			require.Error(t, service.Monitor(context.Background(), params), "incoming marker is not a queued successor")
			assert.Empty(t, queue.events)
			require.NoError(t, service.Monitor(context.Background(), params))
			require.Len(t, queue.events, 1)
		})
	}
}

func TestHeartbeatServiceMonitor_DeduplicatesOfflineAlertOnSuccessorRetry(t *testing.T) {
	for _, checkID := range []string{"current-check", ""} {
		t.Run("incoming:"+checkID, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue, server := newHeartbeatCloudTaskQueue(t, logger, tracer)
			server.rejectChecks = true
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1",
				Owner: "+18005550199", PhoneOnline: true, QueueID: "current-check", UpdatedAt: time.Now().UTC(),
			}
			service := NewHeartbeatService(logger, tracer, &heartbeatServiceRepository{
				heartbeat: &entities.Heartbeat{Timestamp: time.Now().UTC().Add(-4*heartbeatCheckInterval - time.Minute)},
			}, &heartbeatServiceMonitorRepository{monitor: monitor},
				NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{ConsumerEndpoint: "https://example.com/events"}))
			params := &HeartbeatMonitorParams{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
				Source: "https://example.com", CheckID: checkID,
			}
			require.Error(t, service.Monitor(context.Background(), params))
			require.Len(t, server.queuedEvents(t, events.EventTypePhoneHeartbeatOffline), 1)
			assert.True(t, monitor.PhoneOnline, "offline listener has not run yet")
			server.mu.Lock()
			server.rejectChecks = false
			server.mu.Unlock()
			require.NoError(t, service.Monitor(context.Background(), params))
			require.Len(t, server.queuedEvents(t, events.EventTypePhoneHeartbeatOffline), 1, "retry must not queue another email/webhook alert")
			checks := server.queuedEvents(t, events.EventTypePhoneHeartbeatCheck)
			require.Len(t, checks, 1)
			var payload events.PhoneHeartbeatCheckPayload
			require.NoError(t, checks[0].DataAs(&payload))
			assert.Equal(t, "check:"+payload.CheckID, monitor.QueueID)
		})
	}
}

func TestHeartbeatServiceMonitor_RevalidatesAfterMissedNotification(t *testing.T) {
	for _, takeover := range []bool{false, true} {
		t.Run(map[bool]string{false: "returning heartbeat", true: "lease takeover"}[takeover], func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{}
			monitor := &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1", Owner: "+18005550199",
				PhoneOnline: true, QueueID: "check:current-check", UpdatedAt: time.Now().UTC(),
			}
			repository := &heartbeatServiceRepository{heartbeat: &entities.Heartbeat{
				ID: uuid.New(), Timestamp: time.Now().UTC().Add(-4*heartbeatCheckInterval - time.Minute),
			}}
			service := NewHeartbeatService(logger, tracer, repository, &heartbeatServiceMonitorRepository{monitor: monitor},
				NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
			params := HeartbeatMonitorParams{
				UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID,
				Source: "https://example.com", CheckID: "current-check",
			}
			queue.afterEnqueue = func() {
				monitor.UpdatedAt = time.Now().UTC().Add(-2 * heartbeatClaimLease)
				_, err := service.Store(context.Background(), HeartbeatStoreParams{
					UserID: monitor.UserID, Owner: monitor.Owner, Source: params.Source, Timestamp: time.Now().UTC(),
				})
				require.NoError(t, err)
				if takeover {
					competing := params
					require.NoError(t, service.Monitor(context.Background(), &competing))
				}
			}
			require.NoError(t, service.Monitor(context.Background(), &params))
			require.Len(t, queue.events, 2, "only the missed signal and one owned successor should be queued")
			assert.Equal(t, events.EventTypePhoneHeartbeatCheck, queue.events[1].Type())
			var successor events.PhoneHeartbeatCheckPayload
			require.NoError(t, queue.events[1].DataAs(&successor))
			assert.Equal(t, "check:"+successor.CheckID, monitor.QueueID)
			assert.WithinDuration(t, repository.heartbeat.Timestamp.Add(heartbeatCheckInterval), successor.ScheduledAt, time.Millisecond)
		})
	}
}

func TestHeartbeatServiceMonitor_OfflineRetryNamesFollowHeartbeatIdentity(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue, server := newHeartbeatCloudTaskQueue(t, logger, tracer)
	server.rejectChecks = true
	first := &entities.Heartbeat{ID: uuid.New(), Timestamp: time.Now().UTC().Add(-65 * time.Minute)}
	history := &heartbeatServiceRepository{heartbeat: first}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1", Owner: "+18005550199",
		PhoneOnline: true, QueueID: "check:current-check", UpdatedAt: time.Now().UTC(),
	}
	service := NewHeartbeatService(logger, tracer, history, &heartbeatServiceMonitorRepository{monitor: monitor},
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{ConsumerEndpoint: "https://example.com/events"}))
	params := &HeartbeatMonitorParams{
		UserID: monitor.UserID, Owner: monitor.Owner, MonitorID: monitor.ID, Source: "https://example.com", CheckID: "current-check",
	}
	require.Error(t, service.Monitor(context.Background(), params))
	second, err := service.Store(context.Background(), HeartbeatStoreParams{
		UserID: monitor.UserID, Owner: monitor.Owner, Source: params.Source, Timestamp: first.Timestamp,
	})
	require.NoError(t, err)
	server.mu.Lock()
	server.rejectChecks = false
	server.mu.Unlock()
	require.NoError(t, service.Monitor(context.Background(), params))
	alerts := server.queuedEvents(t, events.EventTypePhoneHeartbeatOffline)
	require.Len(t, alerts, 2, "a stale queued alert must not suppress a distinct heartbeat version with the same timestamp")
	var firstPayload, secondPayload events.PhoneHeartbeatOfflinePayload
	require.NoError(t, alerts[0].DataAs(&firstPayload))
	require.NoError(t, alerts[1].DataAs(&secondPayload))
	require.Equal(t, first.ID, firstPayload.HeartbeatID)
	require.Equal(t, second.ID, secondPayload.HeartbeatID)
}

func TestHeartbeatServiceStore_DoesNotRestartHistoryRemovedAfterSaving(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	queue := &heartbeatServiceQueue{}
	history := &heartbeatServiceRepository{}
	monitor := &entities.HeartbeatMonitor{
		ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1", Owner: "+18005550199", PhoneOnline: true,
	}
	repository := &heartbeatServiceMonitorRepository{monitor: monitor}
	repository.afterLoad = func() {
		history.heartbeat = nil
		history.err = stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "history expired after saving")
	}
	service := NewHeartbeatService(logger, tracer, history, repository,
		NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{}))
	heartbeat, err := service.Store(context.Background(), HeartbeatStoreParams{
		UserID: monitor.UserID, Owner: monitor.Owner, Source: "https://example.com", Timestamp: time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NotNil(t, heartbeat)
	assert.Empty(t, queue.events, "known missing history must not schedule checks")
	assert.True(t, monitor.RequiresCheck(), "a later heartbeat must still be able to restart")
}
