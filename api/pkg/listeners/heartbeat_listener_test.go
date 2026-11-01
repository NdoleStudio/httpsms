package listeners

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

type offlineHeartbeatRepository struct {
	repositories.HeartbeatRepository
	heartbeat *entities.Heartbeat
	err       error
}

func (repository *offlineHeartbeatRepository) Last(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
	return repository.heartbeat, repository.err
}

func (repository *offlineHeartbeatRepository) Store(_ context.Context, heartbeat *entities.Heartbeat) error {
	repository.heartbeat = heartbeat
	return nil
}

type offlineMonitorRepository struct {
	repositories.HeartbeatMonitorRepository
	monitor         *entities.HeartbeatMonitor
	lastHeartbeatID uuid.UUID
	beforeOffline   func()
}

func (repository *offlineMonitorRepository) Load(context.Context, entities.UserID, string) (*entities.HeartbeatMonitor, error) {
	monitor := *repository.monitor
	monitor.LastHeartbeatID = repository.lastHeartbeatID
	return &monitor, nil
}

func (repository *offlineMonitorRepository) CompareAndSwapQueueID(_ context.Context, _ uuid.UUID, expected, next string) (bool, error) {
	if repository.monitor.QueueID != expected {
		return false, nil
	}
	repository.monitor.QueueID = next
	repository.monitor.UpdatedAt = time.Now().UTC()
	return true, nil
}

func (repository *offlineMonitorRepository) RecordHeartbeat(_ context.Context, _ entities.UserID, _ uuid.UUID, expected, heartbeatID uuid.UUID) (*entities.HeartbeatMonitor, error) {
	if repository.lastHeartbeatID != expected {
		return nil, nil
	}
	previous := *repository.monitor
	previous.LastHeartbeatID = repository.lastHeartbeatID
	repository.lastHeartbeatID = heartbeatID
	repository.monitor.PhoneOnline = true
	return &previous, nil
}

func (repository *offlineMonitorRepository) UpdatePhoneOfflineForHeartbeat(_ context.Context, _ entities.UserID, _ uuid.UUID, expected, heartbeatID uuid.UUID) (bool, error) {
	repository.runOfflineHook()
	if repository.lastHeartbeatID != expected {
		return false, nil
	}
	repository.monitor.PhoneOnline = false
	if heartbeatID != uuid.Nil {
		repository.lastHeartbeatID = heartbeatID
	}
	return true, nil
}

func (repository *offlineMonitorRepository) UpdatePhoneOnline(_ context.Context, _ entities.UserID, _ uuid.UUID, online bool) error {
	if !online {
		repository.runOfflineHook()
	}
	repository.monitor.PhoneOnline = online
	return nil
}

func (repository *offlineMonitorRepository) runOfflineHook() {
	if repository.beforeOffline != nil {
		hook := repository.beforeOffline
		repository.beforeOffline = nil
		hook()
	}
}

type offlineEventQueue struct{ events []cloudevents.Event }

func (queue *offlineEventQueue) Enqueue(_ context.Context, task *services.PushQueueTask, _ time.Duration) (string, error) {
	var event cloudevents.Event
	if err := json.Unmarshal(task.Body, &event); err != nil {
		return "", err
	}
	queue.events = append(queue.events, event)
	return event.ID(), nil
}

func TestHeartbeatOfflineDelivery_RejectsNewerHeartbeatBeforeSubscribers(t *testing.T) {
	for _, arrival := range []string{"before delivery", "between history and status update", "history read fails", "history expired", "no newer heartbeat", "legacy event"} {
		t.Run(arrival, func(t *testing.T) {
			logger := &noopListenerLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			meter, err := noop.NewMeterProvider().Meter("test").Float64Histogram("dispatch")
			require.NoError(t, err)
			queue := &offlineEventQueue{}
			dispatcher := services.NewEventDispatcher(logger, tracer, meter, queue, services.PushQueueConfig{})
			history := &offlineHeartbeatRepository{heartbeat: &entities.Heartbeat{
				ID: uuid.New(), Timestamp: time.Now().UTC().Add(-65 * time.Minute),
			}}
			monitor := &offlineMonitorRepository{monitor: &entities.HeartbeatMonitor{
				ID: uuid.New(), PhoneID: uuid.New(), UserID: "user-1", Owner: "+18005550199",
				PhoneOnline: true, QueueID: "check:current-check", UpdatedAt: time.Now().UTC(),
			}, lastHeartbeatID: history.heartbeat.ID}
			service := services.NewHeartbeatService(logger, tracer, history, monitor, dispatcher)
			heartbeatListener, routes := NewHeartbeatListener(logger, tracer, service)
			dispatcher.SubscribeFilter(events.EventTypePhoneHeartbeatOffline, heartbeatListener.FilterOfflineEvent)
			for eventType, listener := range routes {
				dispatcher.Subscribe(eventType, listener)
			}
			var notifications atomic.Int32
			dispatcher.Subscribe(events.EventTypePhoneHeartbeatOffline, func(context.Context, cloudevents.Event) error {
				notifications.Add(1)
				return nil
			})
			require.NoError(t, service.Monitor(context.Background(), &services.HeartbeatMonitorParams{
				UserID: monitor.monitor.UserID, Owner: monitor.monitor.Owner, MonitorID: monitor.monitor.ID,
				Source: "https://example.com", CheckID: "current-check",
			}))
			require.Len(t, queue.events, 3)
			event := queue.events[1]
			storeHeartbeat := func() {
				_, err := service.Store(context.Background(), services.HeartbeatStoreParams{
					UserID: monitor.monitor.UserID, Owner: monitor.monitor.Owner,
					Source: "https://example.com", Timestamp: time.Now().UTC(),
				})
				require.NoError(t, err)
			}
			switch arrival {
			case "before delivery":
				storeHeartbeat()
			case "between history and status update":
				monitor.beforeOffline = storeHeartbeat
			case "history read fails":
				history.err = stacktrace.NewError("history unavailable")
			case "history expired":
				history.err = stacktrace.NewErrorWithCode(repositories.ErrCodeNotFound, "history expired")
			case "legacy event":
				var data map[string]any
				require.NoError(t, event.DataAs(&data))
				delete(data, "heartbeat_id")
				require.NoError(t, event.SetData(cloudevents.ApplicationJSON, data))
				monitor.lastHeartbeatID = uuid.Nil
			}
			err = dispatcher.DispatchSync(context.Background(), event)
			if arrival == "history read fails" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if arrival == "no newer heartbeat" || arrival == "legacy event" {
				require.EqualValues(t, 1, notifications.Load())
				require.False(t, monitor.monitor.PhoneOnline)
			} else {
				require.Zero(t, notifications.Load(), "stale or unverifiable events must not reach email/webhook subscribers")
				require.True(t, monitor.monitor.PhoneOnline)
			}
		})
	}
}
