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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEmulatorPushQueue_DeduplicatesOfflineEvents(t *testing.T) {
	deliveries := make(chan struct{}, 2)
	consumer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		deliveries <- struct{}{}
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(consumer.Close)
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	dispatcher := NewEventDispatcher(logger, tracer, nil,
		EmulatorPushQueue(logger, tracer, consumer.Client(), PushQueueConfig{}),
		PushQueueConfig{ConsumerEndpoint: consumer.URL})
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetType(events.EventTypePhoneHeartbeatOffline)
	event.SetSource("https://example.com")
	require.NoError(t, dispatcher.Dispatch(context.Background(), event))
	require.NoError(t, dispatcher.Dispatch(context.Background(), event))
	select {
	case <-deliveries:
	case <-time.After(time.Second):
		t.Fatal("offline alert was not delivered")
	}
	select {
	case <-deliveries:
		t.Fatal("offline alert was delivered twice")
	case <-time.After(100 * time.Millisecond):
	}
}
