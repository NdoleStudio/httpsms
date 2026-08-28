package listeners

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/pusher/pusher-http-go/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketListenerRegistersMissedCalls(t *testing.T) {
	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	_, routes := NewWebsocketListener(logger, tracer, &pusher.Client{})

	assert.Contains(t, routes, events.MessageCallMissed)
}

func TestWebsocketListenerPublishesReceivedMessageID(t *testing.T) {
	userID := "user-id"
	messageID := uuid.New()
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetType(events.EventTypeMessagePhoneReceived)
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.MessagePhoneReceivedPayload{
		MessageID: messageID,
		UserID:    entities.UserID(userID),
	}))

	payload := captureWebsocketPayload(t, event, userID)

	assert.Equal(t, messageID.String(), payload.MessageID)
}

func TestWebsocketListenerPublishesMissedCallMessageID(t *testing.T) {
	userID := "user-id"
	messageID := uuid.New()
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetType(events.MessageCallMissed)
	require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.MessageCallMissedPayload{
		MessageID: messageID,
		UserID:    entities.UserID(userID),
	}))

	payload := captureWebsocketPayload(t, event, userID)

	assert.Equal(t, messageID.String(), payload.MessageID)
}

func captureWebsocketPayload(t *testing.T, event cloudevents.Event, userID string) websocketMessagePayload {
	t.Helper()

	requestBody := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		requestBody <- body
		writer.Header().Set("Content-Type", "application/json")
		_, err = writer.Write([]byte(`{}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	client := &pusher.Client{
		AppID:      "app-id",
		Key:        "key",
		Secret:     "secret",
		Host:       strings.TrimPrefix(server.URL, "http://"),
		HTTPClient: server.Client(),
	}
	_, routes := NewWebsocketListener(logger, tracer, client)

	require.NoError(t, routes[event.Type()](context.Background(), event))

	var trigger struct {
		Channels []string `json:"channels"`
		Data     string   `json:"data"`
	}
	require.NoError(t, json.Unmarshal(<-requestBody, &trigger))
	assert.Equal(t, []string{userID}, trigger.Channels)

	var payload websocketMessagePayload
	require.NoError(t, json.Unmarshal([]byte(trigger.Data), &payload))
	return payload
}
