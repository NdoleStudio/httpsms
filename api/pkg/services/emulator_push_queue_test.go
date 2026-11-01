package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestEmulatorPushQueue_RetriesFailedHeartbeatDelivery(t *testing.T) {
	var attempts atomic.Int32
	delivered := make(chan string, 1)
	firstFailure := make(chan struct{})
	consumer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		if attempts.Add(1) < 3 {
			writer.WriteHeader(http.StatusServiceUnavailable)
			if attempts.Load() == 1 {
				close(firstFailure)
			}
			return
		}
		writer.WriteHeader(http.StatusNoContent)
		delivered <- string(body)
	}))
	t.Cleanup(consumer.Close)
	logger := &phoneNotificationLogger{}
	queue := EmulatorPushQueue(logger, telemetry.NewOtelLogger("test", logger), consumer.Client(), PushQueueConfig{})
	task := &PushQueueTask{ID: "retryable-heartbeat", URL: consumer.URL, Method: http.MethodPost, Body: []byte(`{"id":"same-heartbeat-check"}`)}
	ctx, cancel := context.WithCancel(context.Background())
	_, err := queue.Enqueue(ctx, task, 0)
	require.NoError(t, err)
	cancel()
	select {
	case <-firstFailure:
	case <-time.After(time.Second):
		t.Fatal("initial heartbeat delivery did not run")
	}
	_, err = queue.Enqueue(context.Background(), task, 0)
	require.NoError(t, err)
	select {
	case body := <-delivered:
		require.Equal(t, `{"id":"same-heartbeat-check"}`, body)
		require.Equal(t, int32(3), attempts.Load())
	case <-time.After(5 * time.Second):
		t.Fatal("transient HTTP failures dropped the heartbeat check instead of retrying it")
	}
}
