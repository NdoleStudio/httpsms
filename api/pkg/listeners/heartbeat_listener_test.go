package listeners

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

type offlineMonitorRepository struct {
	repositories.HeartbeatMonitorRepository
	monitor *entities.HeartbeatMonitor
}

func (repository *offlineMonitorRepository) UpdatePhoneOnline(_ context.Context, userID entities.UserID, monitorID uuid.UUID, online bool) error {
	if repository.monitor.UserID == userID && repository.monitor.ID == monitorID {
		repository.monitor.PhoneOnline = online
	}
	return nil
}

func TestHeartbeatOfflineDelivery_AllowsRepeatedAndOutOfOrderEvents(t *testing.T) {
	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	meter, err := noop.NewMeterProvider().Meter("test").Float64Histogram("dispatch")
	require.NoError(t, err)
	dispatcher := services.NewEventDispatcher(logger, tracer, meter, nil, services.PushQueueConfig{})
	monitor := &entities.HeartbeatMonitor{ID: uuid.New(), UserID: "user-1", PhoneOnline: true}
	repository := &offlineMonitorRepository{monitor: monitor}
	service := services.NewHeartbeatService(logger, tracer, nil, repository, dispatcher)
	_, routes := NewHeartbeatListener(logger, tracer, service)
	for eventType, listener := range routes {
		dispatcher.Subscribe(eventType, listener)
	}
	var notifications atomic.Int32
	dispatcher.Subscribe(events.EventTypePhoneHeartbeatOffline, func(context.Context, cloudevents.Event) error {
		notifications.Add(1)
		return nil
	})
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource("https://example.com")
	event.SetType(events.EventTypePhoneHeartbeatOffline)
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.PhoneHeartbeatOfflinePayload{
		MonitorID: monitor.ID, UserID: monitor.UserID, LastHeartbeatTimestamp: time.Now().UTC().Add(-24 * time.Hour),
	}))

	for delivery := 1; delivery <= 2; delivery++ {
		monitor.PhoneOnline = true
		require.NoError(t, dispatcher.DispatchSync(context.Background(), event))
		require.False(t, monitor.PhoneOnline)
		require.EqualValues(t, delivery, notifications.Load())
	}
}
