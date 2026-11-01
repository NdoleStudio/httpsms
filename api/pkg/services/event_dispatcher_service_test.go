package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestEventDispatcherDispatchSync_RetriesOnlyHeartbeatCheckFailures(t *testing.T) {
	for _, eventType := range []string{events.EventTypePhoneHeartbeatCheck, events.EventTypePhoneHeartbeatOnline} {
		t.Run(eventType, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			meter, err := noop.NewMeterProvider().Meter("test").Float64Histogram("dispatch")
			require.NoError(t, err)
			dispatcher := NewEventDispatcher(logger, tracer, meter, nil, PushQueueConfig{})
			failure := stacktrace.NewError("database unavailable")
			dispatcher.Subscribe(eventType, func(context.Context, cloudevents.Event) error { return failure })
			event := cloudevents.NewEvent()
			event.SetID("check-1")
			event.SetSource("https://example.com")
			event.SetType(eventType)
			err = dispatcher.DispatchSync(context.Background(), event)
			if eventType == events.EventTypePhoneHeartbeatCheck {
				require.ErrorIs(t, err, failure)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEventDispatcherDispatchWithTimeout_DoesNotUseLocalFallbackForHeartbeatChecks(t *testing.T) {
	for _, eventType := range []string{events.EventTypePhoneHeartbeatCheck, events.EventTypePhoneHeartbeatOnline} {
		t.Run(eventType, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue := &heartbeatServiceQueue{err: context.DeadlineExceeded}
			dispatcher := NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{})
			event := cloudevents.NewEvent()
			event.SetID("check-1")
			event.SetSource("https://example.com")
			event.SetType(eventType)
			queueID, err := dispatcher.DispatchWithTimeout(context.Background(), event, time.Hour)
			if eventType == events.EventTypePhoneHeartbeatCheck {
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Empty(t, queueID)
			} else {
				require.NoError(t, err)
				require.Equal(t, "local-check-1", queueID)
			}
		})
	}
}
