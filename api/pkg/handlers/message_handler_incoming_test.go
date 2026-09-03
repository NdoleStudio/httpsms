package handlers

import (
	"context"
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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

type messageIncomingRepositoryStub struct {
	searchUserID entities.UserID
	searchOwners []string
	searchTypes  []entities.MessageType
	searchStatus []entities.MessageStatus
	searchParams repositories.IndexParams
}

func (stub *messageIncomingRepositoryStub) Store(context.Context, *entities.Message) error {
	return nil
}

func (stub *messageIncomingRepositoryStub) Update(context.Context, *entities.Message) error {
	return nil
}

func (stub *messageIncomingRepositoryStub) Load(context.Context, entities.UserID, uuid.UUID) (*entities.Message, error) {
	return nil, nil
}

func (stub *messageIncomingRepositoryStub) Index(context.Context, entities.UserID, string, string, repositories.IndexParams) (*[]entities.Message, error) {
	return nil, nil
}

func (stub *messageIncomingRepositoryStub) LastMessage(context.Context, entities.UserID, string, string) (*entities.Message, error) {
	return nil, nil
}

func (stub *messageIncomingRepositoryStub) Search(_ context.Context, userID entities.UserID, owners []string, types []entities.MessageType, statuses []entities.MessageStatus, params repositories.IndexParams) ([]*entities.Message, error) {
	stub.searchUserID = userID
	stub.searchOwners = owners
	stub.searchTypes = types
	stub.searchStatus = statuses
	stub.searchParams = params
	return []*entities.Message{}, nil
}

func (stub *messageIncomingRepositoryStub) GetBulkMessages(context.Context, entities.UserID, int) ([]*entities.BulkMessage, error) {
	return nil, nil
}

func (stub *messageIncomingRepositoryStub) GetOutstanding(context.Context, entities.UserID, uuid.UUID, []string) (*entities.Message, error) {
	return nil, nil
}

func (stub *messageIncomingRepositoryStub) Delete(context.Context, entities.UserID, uuid.UUID) error {
	return nil
}

func (stub *messageIncomingRepositoryStub) DeleteByOwnerAndContact(context.Context, entities.UserID, string, string) error {
	return nil
}

func (stub *messageIncomingRepositoryStub) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}

func TestMessageHandlerIncoming_ForcesMobileOriginatedType(t *testing.T) {
	logger := &messageIncomingNoopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	repository := &messageIncomingRepositoryStub{}
	service := services.NewMessageService(logger, tracer, repository, nil, nil, nil, "http://localhost")
	validator := validators.NewMessageHandlerValidator(logger, tracer, nil, nil)
	handler := NewMessageHandler(logger, tracer, validator, nil, service)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
		return c.Next()
	})
	handler.RegisterRoutes(app)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages/incoming?owners=%2B18005550199&limit=25&skip=0", nil)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, []entities.MessageType{entities.MessageTypeMobileOriginated}, repository.searchTypes)
	require.Equal(t, entities.UserID("user-id"), repository.searchUserID)
	require.Equal(t, []string{"+18005550199"}, repository.searchOwners)
}

func TestMessageHandlerIncoming_ReturnsUnprocessableEntityForInvalidOwner(t *testing.T) {
	logger := &messageIncomingNoopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	repository := &messageIncomingRepositoryStub{}
	service := services.NewMessageService(logger, tracer, repository, nil, nil, nil, "http://localhost")
	validator := validators.NewMessageHandlerValidator(logger, tracer, nil, nil)
	handler := NewMessageHandler(logger, tracer, validator, nil, service)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
		return c.Next()
	})
	handler.RegisterRoutes(app)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages/incoming?owners=not-a-phone-number&limit=25&skip=0", nil)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

type messageIncomingNoopLogger struct{}

var _ telemetry.Logger = (*messageIncomingNoopLogger)(nil)

func (logger *messageIncomingNoopLogger) Error(_ error)                         {}
func (logger *messageIncomingNoopLogger) WithService(_ string) telemetry.Logger { return logger }

func (logger *messageIncomingNoopLogger) WithString(_, _ string) telemetry.Logger { return logger }

func (logger *messageIncomingNoopLogger) WithSpan(_ trace.SpanContext) telemetry.Logger {
	return logger
}
func (logger *messageIncomingNoopLogger) Trace(_ string)                    {}
func (logger *messageIncomingNoopLogger) Info(_ string)                     {}
func (logger *messageIncomingNoopLogger) Warn(_ error)                      {}
func (logger *messageIncomingNoopLogger) Debug(_ string)                    {}
func (logger *messageIncomingNoopLogger) Fatal(_ error)                     {}
func (logger *messageIncomingNoopLogger) Printf(_ string, _ ...interface{}) {}
