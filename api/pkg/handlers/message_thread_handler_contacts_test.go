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
	threads    []entities.MessageThread
	unreadOnly bool
	limit      int
}

func (stub *messageThreadHandlerIndexRepositoryStub) Index(_ context.Context, _ entities.UserID, _ string, _ bool, unreadOnly bool, params repositories.IndexParams) (*[]entities.MessageThread, error) {
	stub.unreadOnly = unreadOnly
	stub.limit = params.Limit
	threads := make([]entities.MessageThread, len(stub.threads))
	copy(threads, stub.threads)
	return &threads, nil
}

type messageThreadHandlerContactProviderStub struct {
	contacts map[string]*entities.Contact
	calls    int
}

func (stub *messageThreadHandlerContactProviderStub) GetContactMap(context.Context, entities.UserID, []string) (map[string]*entities.Contact, error) {
	stub.calls++
	return stub.contacts, nil
}

func TestMessageThreadHandlerIndex_ParsesContactsAndUnreadQuery(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/v1/message-threads?owner=%2B18005550199&contacts=true&unread_only=true", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, provider.calls)
	assert.True(t, repository.unreadOnly)

	var payload struct {
		Data []entities.MessageThread `json:"data"`
	}

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Len(t, payload.Data, 1)
	require.NotNil(t, payload.Data[0].ContactDetails)
	assert.Equal(t, contact.ID, payload.Data[0].ContactDetails.ID)
	assert.Equal(t, "Alice", payload.Data[0].ContactDetails.Name)
}

func TestMessageThreadHandlerIndexRejectsInvalidUnreadFilter(t *testing.T) {
	repository := &messageThreadHandlerIndexRepositoryStub{}
	logger := &messageThreadHandlerNoopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	service := services.NewMessageThreadService(logger, tracer, repository, nil, nil, nil)
	handler := NewMessageThreadHandler(logger, tracer, validators.NewMessageThreadHandlerValidator(logger, tracer), service)
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
		return c.Next()
	})
	handler.RegisterRoutes(app)

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/message-threads?owner=%2B18005550199&unread_only=invalid", nil))

	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
}

func TestMessageThreadIndexReportsWhetherAnotherPageExists(t *testing.T) {
	for _, test := range []struct {
		name    string
		count   int
		wantLen int
		more    bool
	}{
		{name: "empty", count: 0, wantLen: 0},
		{name: "partial page", count: 19, wantLen: 19},
		{name: "exactly one page", count: 20, wantLen: 20},
		{name: "another page exists", count: 21, wantLen: 20, more: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &messageThreadHandlerIndexRepositoryStub{threads: make([]entities.MessageThread, test.count)}
			logger := &messageThreadHandlerNoopLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			service := services.NewMessageThreadService(logger, tracer, repository, nil, nil, nil)
			handler := NewMessageThreadHandler(logger, tracer, validators.NewMessageThreadHandlerValidator(logger, tracer), service)
			app := fiber.New()
			app.Use(func(c fiber.Ctx) error {
				c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID("user-id"), Email: "user@example.com"})
				return c.Next()
			})
			handler.RegisterRoutes(app)

			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/message-threads?owner=%2B18005550199&limit=20", nil))

			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, http.StatusOK, response.StatusCode)
			var body struct {
				Data    []entities.MessageThread `json:"data"`
				HasMore bool                     `json:"has_more"`
			}
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			assert.Len(t, body.Data, test.wantLen)
			assert.Equal(t, test.more, body.HasMore)
			assert.Equal(t, 21, repository.limit)
		})
	}
}
