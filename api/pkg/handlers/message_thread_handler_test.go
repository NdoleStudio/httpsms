package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/middlewares"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/NdoleStudio/stacktrace"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type messageThreadHandlerRepositoryStub struct {
	load func(context.Context, entities.UserID, uuid.UUID) (*entities.MessageThread, error)
}

func (stub *messageThreadHandlerRepositoryStub) Store(context.Context, *entities.MessageThread) error {
	return nil
}

func (stub *messageThreadHandlerRepositoryStub) UpdateActivity(context.Context, repositories.MessageThreadActivityUpdate) error {
	return nil
}

func (stub *messageThreadHandlerRepositoryStub) UpdateStatus(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error) {
	return nil, stacktrace.PropagateWithCodef(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
}

func (stub *messageThreadHandlerRepositoryStub) LoadByOwnerContact(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
	return nil, nil
}

func (stub *messageThreadHandlerRepositoryStub) Load(ctx context.Context, userID entities.UserID, id uuid.UUID) (*entities.MessageThread, error) {
	if stub.load != nil {
		return stub.load(ctx, userID, id)
	}
	return nil, nil
}

func (stub *messageThreadHandlerRepositoryStub) Index(context.Context, entities.UserID, string, bool, bool, repositories.IndexParams) (*[]entities.MessageThread, error) {
	return nil, nil
}

func (stub *messageThreadHandlerRepositoryStub) UpdateAfterDeletedMessage(context.Context, repositories.MessageThreadDeletedUpdate) error {
	return nil
}

func (stub *messageThreadHandlerRepositoryStub) Delete(context.Context, entities.UserID, uuid.UUID) error {
	return nil
}

func (stub *messageThreadHandlerRepositoryStub) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}

func TestMessageThreadHandlerUpdate_ReturnsNotFoundWhenThreadIsMissing(t *testing.T) {
	logger := &messageThreadHandlerNoopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	service := services.NewMessageThreadService(logger, tracer, &messageThreadHandlerRepositoryStub{}, nil, nil, nil)
	handler := NewMessageThreadHandler(logger, tracer, validators.NewMessageThreadHandlerValidator(logger, tracer), service)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
		return c.Next()
	})
	handler.RegisterRoutes(app)

	messageThreadID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/v1/message-threads/"+messageThreadID.String(), bytes.NewBufferString(`{"unread_count":0}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var payload struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Equal(t, "cannot find message thread with ID ["+messageThreadID.String()+"]", payload.Message)
}

func TestMessageThreadHandlerGetLoadsIndividualUserScopedThread(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
	}{
		{name: "existing", status: http.StatusOK},
		{name: "missing", status: http.StatusNotFound},
		{name: "another user", status: http.StatusNotFound},
		{name: "invalid UUID", status: http.StatusUnprocessableEntity},
		{name: "repository failure", status: http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			id := uuid.New()
			contact := &entities.Contact{ID: uuid.New(), Name: "Alice", PhoneNumbers: []string{"+18005550100"}}
			repository := &messageThreadHandlerRepositoryStub{
				load: func(_ context.Context, userID entities.UserID, threadID uuid.UUID) (*entities.MessageThread, error) {
					require.Equal(t, entities.UserID("user-id"), userID)
					require.Equal(t, id, threadID)
					if test.name == "missing" || test.name == "another user" {
						return nil, stacktrace.PropagateWithCodef(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
					}
					if test.name == "repository failure" {
						return nil, stacktrace.NewError("database unavailable")
					}
					return &entities.MessageThread{ID: id, UserID: userID, Owner: "+18005550199", Contact: "+18005550100"}, nil
				},
			}
			logger := &messageThreadHandlerNoopLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			provider := &messageThreadHandlerContactProviderStub{contacts: map[string]*entities.Contact{"+18005550100": contact}}
			service := services.NewMessageThreadService(logger, tracer, repository, nil, nil, provider)
			handler := NewMessageThreadHandler(logger, tracer, validators.NewMessageThreadHandlerValidator(logger, tracer), service)
			app := fiber.New()
			app.Use(func(c fiber.Ctx) error {
				c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
				return c.Next()
			})
			handler.RegisterRoutes(app)
			pathID := id.String()
			if test.name == "invalid UUID" {
				pathID = "invalid"
			}

			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/message-threads/"+pathID, nil))

			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, test.status, response.StatusCode)
			if test.status == http.StatusOK {
				var payload struct {
					Data entities.MessageThread `json:"data"`
				}
				require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
				require.Equal(t, id, payload.Data.ID)
				require.NotNil(t, payload.Data.ContactDetails)
				require.Equal(t, "Alice", payload.Data.ContactDetails.Name)
			}
		})
	}
}

type messageThreadHandlerNoopLogger struct{}

var _ telemetry.Logger = (*messageThreadHandlerNoopLogger)(nil)

func (logger *messageThreadHandlerNoopLogger) Error(_ error)                         {}
func (logger *messageThreadHandlerNoopLogger) WithService(_ string) telemetry.Logger { return logger }

func (logger *messageThreadHandlerNoopLogger) WithString(_, _ string) telemetry.Logger { return logger }

func (logger *messageThreadHandlerNoopLogger) WithSpan(_ trace.SpanContext) telemetry.Logger {
	return logger
}
func (logger *messageThreadHandlerNoopLogger) Trace(_ string)                    {}
func (logger *messageThreadHandlerNoopLogger) Info(_ string)                     {}
func (logger *messageThreadHandlerNoopLogger) Warn(_ error)                      {}
func (logger *messageThreadHandlerNoopLogger) Debug(_ string)                    {}
func (logger *messageThreadHandlerNoopLogger) Fatal(_ error)                     {}
func (logger *messageThreadHandlerNoopLogger) Printf(_ string, _ ...interface{}) {}
