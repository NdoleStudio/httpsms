package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/middlewares"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestEventsHandler_UsesDeliveredTaskQueueID(t *testing.T) {
	for _, test := range []struct {
		name    string
		taskID  string
		queueID string
	}{
		{name: "cloud task", taskID: "task-123", queueID: "projects/test/locations/us/queues/heartbeats/tasks/task-123"},
		{name: "missing task header"},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := &messageThreadHandlerNoopLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			meter, err := noop.NewMeterProvider().Meter("test").Float64Histogram("events")
			require.NoError(t, err)
			config := services.PushQueueConfig{
				Name: "projects/test/locations/us/queues/heartbeats", UserID: "queue-user",
			}

			dispatcher := services.NewEventDispatcher(logger, tracer, meter, nil, config)
			var delivered cloudevents.Event
			dispatcher.Subscribe(events.EventTypePhoneHeartbeatCheck, func(_ context.Context, event cloudevents.Event) error {
				delivered = event
				return nil
			})
			handler := NewEventsHandler(logger, tracer, config, dispatcher)
			app := fiber.New()
			app.Use(func(c fiber.Ctx) error {
				c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: config.UserID, Email: "queue@example.com"})
				return c.Next()
			})
			handler.RegisterRoutes(app)
			event := cloudevents.NewEvent()
			event.SetID("event-123")
			event.SetSource("https://example.com")
			event.SetType(events.EventTypePhoneHeartbeatCheck)
			event.SetExtension("queueid", "untrusted-body-queue-id")
			require.NoError(t, event.SetData(cloudevents.ApplicationJSON, events.PhoneHeartbeatCheckPayload{}))
			body, err := json.Marshal(event)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-CloudTasks-TaskName", test.taskID)

			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, http.StatusNoContent, response.StatusCode)
			assert.Equal(t, test.queueID, delivered.Extensions()["queueid"])
		})
	}
}

func TestEventsHandler_RetriesFailedHeartbeatChecks(t *testing.T) {
	for _, eventType := range []string{events.EventTypePhoneHeartbeatCheck, events.EventTypePhoneUpdated} {
		t.Run(eventType, func(t *testing.T) {
			logger := &messageThreadHandlerNoopLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			meter, err := noop.NewMeterProvider().Meter("test").Float64Histogram("events")
			require.NoError(t, err)
			config := services.PushQueueConfig{Name: "projects/test/locations/us/queues/heartbeats", UserID: "queue-user"}
			dispatcher := services.NewEventDispatcher(logger, tracer, meter, nil, config)
			attempts := 0
			dispatcher.Subscribe(eventType, func(context.Context, cloudevents.Event) error {
				attempts++
				if attempts == 1 {
					return errors.New("database unavailable")
				}
				return nil
			})
			app := fiber.New()
			app.Use(func(c fiber.Ctx) error {
				c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: config.UserID, Email: "queue@example.com"})
				return c.Next()
			})
			NewEventsHandler(logger, tracer, config, dispatcher).RegisterRoutes(app)
			event := cloudevents.NewEvent()
			event.SetID("event-123")
			event.SetSource("https://example.com")
			event.SetType(eventType)
			body, err := json.Marshal(event)
			require.NoError(t, err)

			statuses := []int{http.StatusNoContent, http.StatusNoContent}
			if eventType == events.EventTypePhoneHeartbeatCheck {
				statuses[0] = http.StatusInternalServerError
			}
			for _, status := range statuses {
				request := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("X-CloudTasks-TaskName", "task-123")
				response, err := app.Test(request)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				assert.Equal(t, status, response.StatusCode)
			}
			assert.Equal(t, 2, attempts)
		})
	}
}
