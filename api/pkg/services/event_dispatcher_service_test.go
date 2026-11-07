package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

type deadlineEventQueue struct{}

func (queue *deadlineEventQueue) Enqueue(context.Context, *PushQueueTask, time.Duration) (string, error) {
	return "", context.DeadlineExceeded
}

func TestEventDispatcher_LocalFallbackIncludesQueueID(t *testing.T) {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	meter, err := noop.NewMeterProvider().Meter("test").Float64Histogram("events")
	require.NoError(t, err)
	dispatcher := NewEventDispatcher(logger, tracer, meter, &deadlineEventQueue{}, PushQueueConfig{})
	delivered := make(chan cloudevents.Event, 1)
	dispatcher.Subscribe(events.EventTypePhoneHeartbeatCheck, func(_ context.Context, event cloudevents.Event) error {
		delivered <- event
		return nil
	})
	event := cloudevents.NewEvent()
	event.SetID("event-123")
	event.SetSource("https://example.com")
	event.SetType(events.EventTypePhoneHeartbeatCheck)

	queueID, err := dispatcher.DispatchWithTimeout(context.Background(), event, time.Millisecond)
	require.NoError(t, err)
	select {
	case received := <-delivered:
		assert.Equal(t, queueID, received.Extensions()["queueid"])
	case <-time.After(time.Second):
		t.Fatal("local fallback did not deliver the event")
	}
}

func TestEmulatorPushQueue_DeliversTaskNameMatchingQueueID(t *testing.T) {
	delivered := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		delivered <- request.Header.Get("X-CloudTasks-TaskName")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	logger := &phoneNotificationLogger{}
	queue := EmulatorPushQueue(logger, telemetry.NewOtelLogger("test", logger), server.Client(),
		PushQueueConfig{Name: "projects/test/locations/us/queues/heartbeats"})

	queueID, err := queue.Enqueue(context.Background(), &PushQueueTask{
		Method: http.MethodPost, URL: server.URL, Body: []byte(`{}`),
	}, time.Millisecond)
	require.NoError(t, err)
	select {
	case taskName := <-delivered:
		require.NotEmpty(t, taskName)
		assert.Equal(t, "projects/test/locations/us/queues/heartbeats/tasks/"+taskName, queueID)
	case <-time.After(time.Second):
		t.Fatal("emulator did not deliver the task")
	}
}
