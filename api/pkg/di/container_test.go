package di

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/middlewares"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimitDisabledDoesNotInitializeResources(t *testing.T) {
	t.Setenv("ENV", "local")
	t.Setenv("RATE_LIMIT_ENABLED", "false")
	container := NewLiteContainer()
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: "user", Email: "user@example.com"})
		return c.Next()
	})
	app.Use(container.AuthenticatedMiddleware())
	app.Get("/", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Nil(t, container.rateLimitService)
	require.Nil(t, container.rateLimitMiddleware)
	require.Nil(t, container.rateLimitRedisClient)
}

func TestRateLimitRedisClientBoundsStalledConnections(t *testing.T) {
	t.Setenv("ENV", "local")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	accepted := make(chan net.Conn, 1)
	go func() {
		connection, err := listener.Accept()
		if err == nil {
			accepted <- connection
		}
	}()
	t.Cleanup(func() {
		select {
		case connection := <-accepted:
			connection.Close()
		case <-time.After(time.Second):
		}
	})
	t.Setenv("REDIS_URL", "redis://"+listener.Addr().String())
	container := NewLiteContainer()
	container.RateLimitService()
	t.Cleanup(container.Close)
	t.Cleanup(func() { container.redisClient.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	err = container.rateLimitRedisClient.Ping(ctx).Err()
	require.Error(t, err)
	require.Less(t, time.Since(start), 1500*time.Millisecond)
}

func TestNotificationHTTPClientUsesOTelRoundTripperWithoutRetries(t *testing.T) {
	t.Setenv("ENV", "local")
	client := NewLiteContainer().NotificationHTTPClient()

	assert.Zero(t, client.Timeout)
	assert.Equal(t, "*otelroundtripper.otelRoundTripper", reflect.TypeOf(client.Transport).String())
	assert.Nil(t, client.CheckRedirect)
}

func TestPhoneNotificationClientsMapsConfiguredTransports(t *testing.T) {
	t.Setenv("ENV", "local")
	t.Setenv("FCM_ENDPOINT", "http://localhost")

	clients := NewLiteContainer().PhoneNotificationClients()

	require.Len(t, clients, 2)
	assert.IsType(t, &services.EmulatorFCMClient{}, clients[entities.NotificationTransportFCM])
	httpSender, ok := clients[entities.NotificationTransportHTTP].(*services.HTTPNotificationSender)
	require.True(t, ok)
	client := reflect.ValueOf(httpSender).Elem().FieldByName("client").Elem()
	transport := client.FieldByName("Transport").Elem()

	assert.Equal(t, "*otelroundtripper.otelRoundTripper", transport.Type().String())
}
