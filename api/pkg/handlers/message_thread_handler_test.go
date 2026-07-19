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
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type messageThreadHandlerRepositoryStub struct{}

func (stub *messageThreadHandlerRepositoryStub) Store(context.Context, *entities.MessageThread) error {
	return nil
}

func (stub *messageThreadHandlerRepositoryStub) UpdateActivity(context.Context, repositories.MessageThreadActivityUpdate) error {
	return nil
}

func (stub *messageThreadHandlerRepositoryStub) UpdateStatus(context.Context, entities.UserID, uuid.UUID, repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error) {
	return nil, stacktrace.PropagateWithCode(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found")
}

func (stub *messageThreadHandlerRepositoryStub) LoadByOwnerContact(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
	return nil, nil
}

func (stub *messageThreadHandlerRepositoryStub) Load(context.Context, entities.UserID, uuid.UUID) (*entities.MessageThread, error) {
	return nil, nil
}

func (stub *messageThreadHandlerRepositoryStub) Index(context.Context, entities.UserID, string, bool, repositories.IndexParams) (*[]entities.MessageThread, error) {
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
	service := services.NewMessageThreadService(logger, tracer, &messageThreadHandlerRepositoryStub{}, nil, nil)
	handler := NewMessageThreadHandler(logger, tracer, validators.NewMessageThreadHandlerValidator(logger, tracer), service)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
		return c.Next()
	})
	handler.RegisterRoutes(app)

	messageThreadID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/v1/message-threads/"+messageThreadID.String(), bytes.NewBufferString(`{"is_read":true}`))
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
