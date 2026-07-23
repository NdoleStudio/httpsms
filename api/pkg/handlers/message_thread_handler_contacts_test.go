package handlers

import (
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type messageThreadHandlerIndexRepositoryStub struct {
	repositories.MessageThreadRepository
	threads []entities.MessageThread
}

func (stub *messageThreadHandlerIndexRepositoryStub) Index(context.Context, entities.UserID, string, bool, repositories.IndexParams) (*[]entities.MessageThread, error) {
	threads := make([]entities.MessageThread, len(stub.threads))
	copy(threads, stub.threads)
	return &threads, nil
}

type messageThreadHandlerContactProviderStub struct {
	contacts map[string]*entities.Contact
	calls    int
}

func (stub *messageThreadHandlerContactProviderStub) GetContactMap(context.Context, entities.UserID) (map[string]*entities.Contact, error) {
	stub.calls++
	return stub.contacts, nil
}

func TestMessageThreadHandlerIndex_ParsesContactsQuery(t *testing.T) {
	contact := &entities.Contact{ID: uuid.New(), Name: "Alice", PhoneNumbers: []string{"+18005550100"}}
	repository := &messageThreadHandlerIndexRepositoryStub{threads: []entities.MessageThread{{Contact: "+18005550100"}}}
	provider := &messageThreadHandlerContactProviderStub{contacts: map[string]*entities.Contact{"+18005550100": contact}}
	logger := &messageThreadHandlerNoopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	service := services.NewMessageThreadService(logger, tracer, repository, nil, nil, provider)
	handler := NewMessageThreadHandler(logger, tracer, validators.NewMessageThreadHandlerValidator(logger, tracer), service)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
		return c.Next()
	})
	handler.RegisterRoutes(app)

	req := httptest.NewRequest(http.MethodGet, "/v1/message-threads?owner=%2B18005550199&contacts=true", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, provider.calls)

	var payload struct {
		Data []entities.MessageThread `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Len(t, payload.Data, 1)
	require.NotNil(t, payload.Data[0].ContactDetails)
	assert.Equal(t, contact.ID, payload.Data[0].ContactDetails.ID)
	assert.Equal(t, "Alice", payload.Data[0].ContactDetails.Name)
}
