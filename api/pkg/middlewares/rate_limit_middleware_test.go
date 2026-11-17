package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v3"
	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestRateLimitMiddleware_WeightedCost(t *testing.T) {
	for _, test := range []struct {
		name   string
		method string
		query  string
		cost   int
	}{
		{"requested page", http.MethodGet, "?limit=10", 10},
		{"search maximum", http.MethodGet, "?limit=200", 200},
		{"capped", http.MethodGet, "?limit=300", 200},
		{"missing limit", http.MethodGet, "", 1},
		{"invalid limit", http.MethodGet, "?limit=invalid", 1},
		{"negative limit", http.MethodGet, "?limit=-10", 1},
		{"write request", http.MethodPost, "?limit=10", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &rateLimitUserRepository{}
			counter := &recordingRateLimitCounter{}
			app := rateLimitTestApp(repo, counter, rateLimitTestAuth, nil)
			response, err := app.Test(httptest.NewRequest(test.method, "/v1/messages/search"+test.query, nil))
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Equal(t, []int{test.cost}, counter.costs)
		})
	}
}

func TestRateLimitMiddleware_RouteAuthentication(t *testing.T) {
	for _, phoneKey := range []bool{false, true} {
		t.Run(map[bool]string{false: "3cx bearer key", true: "phone key"}[phoneKey], func(t *testing.T) {
			repo := &rateLimitUserRepository{}
			counter := &recordingRateLimitCounter{}
			logger, tracer := rateLimitTestTelemetry()
			auth := BearerAPIKeyAuth(logger, tracer, repo)
			request := httptest.NewRequest(http.MethodGet, "/v1/messages/outstanding", nil)
			request.Header.Set("Authorization", "Bearer test-key")
			if phoneKey {
				auth = PhoneAPIKeyAuth(logger, tracer, &rateLimitPhoneRepository{})
				request.Header.Set("x-api-key", "pk_test-key")
			}
			app := rateLimitTestApp(repo, counter, auth, nil)
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Equal(t, []entities.UserID{"user"}, counter.users)
		})
	}
}

func TestRateLimitMiddleware_CachesPlansPerUser(t *testing.T) {
	repo := &rateLimitUserRepository{}
	counter := &recordingRateLimitCounter{}
	app := rateLimitTestApp(repo, counter, rateLimitTestAuth, nil)
	for _, user := range []string{"user", "user", "other", "other", "user"} {
		request := httptest.NewRequest(http.MethodGet, "/v1/messages/search", nil)
		request.Header.Set("test-user", user)
		response, err := app.Test(request)
		require.NoError(t, err)
		response.Body.Close()
	}
	require.Equal(t, 2, repo.loads)
	require.Len(t, counter.costs, 5)
}

func TestRateLimitMiddleware_Exclusions(t *testing.T) {
	repo := &rateLimitUserRepository{}
	counter := &recordingRateLimitCounter{}
	app := rateLimitTestApp(repo, counter, rateLimitTestAuth, nil)
	for _, path := range []string{"/v1/events", "/v1/events/dispatch", "/v1/events-other"} {
		response, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		require.NoError(t, err)
		response.Body.Close()
	}
	require.Equal(t, []int{1}, counter.costs)

	anonymous := rateLimitTestApp(repo, counter, func(c fiber.Ctx) error { return c.Next() }, nil)
	response, err := anonymous.Test(httptest.NewRequest(http.MethodGet, "/v1/messages/search", nil))
	require.NoError(t, err)
	response.Body.Close()
	require.Len(t, counter.costs, 1)
}

func TestRateLimitMiddleware_FailsOpenAndPreservesHandlerErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		repoError  error
		countError error
		nextError  error
		status     int
	}{
		{"plan lookup fails", errors.New("database unavailable"), nil, nil, http.StatusOK},
		{"tracking fails", nil, errors.New("redis unavailable"), nil, http.StatusOK},
		{"handler fails", nil, nil, fiber.ErrTeapot, http.StatusTeapot},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := rateLimitTestApp(
				&rateLimitUserRepository{err: test.repoError},
				&recordingRateLimitCounter{err: test.countError},
				rateLimitTestAuth, test.nextError,
			)
			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/messages/search", nil))
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, test.status, response.StatusCode)
		})
	}
}

type rateLimitUserRepository struct {
	repositories.UserRepository
	loads int
	err   error
}

func (repo *rateLimitUserRepository) Load(context.Context, entities.UserID) (*entities.User, error) {
	repo.loads++
	return &entities.User{SubscriptionName: entities.SubscriptionNameFree}, repo.err
}

func (*rateLimitUserRepository) LoadAuthContext(context.Context, string) (entities.AuthContext, error) {
	return entities.AuthContext{ID: "user", Email: "user@example.com"}, nil
}

type rateLimitPhoneRepository struct {
	repositories.PhoneAPIKeyRepository
}

func (*rateLimitPhoneRepository) LoadAuthContext(context.Context, string) (entities.AuthContext, error) {
	return entities.AuthContext{ID: "user", Email: "user@example.com"}, nil
}

type recordingRateLimitCounter struct {
	costs []int
	users []entities.UserID
	err   error
}

func (counter *recordingRateLimitCounter) Increment(_ context.Context, user entities.UserID, _ entities.SubscriptionName, cost int) (int64, bool, error) {
	counter.costs = append(counter.costs, cost)
	counter.users = append(counter.users, user)
	return 0, false, counter.err
}

func rateLimitTestTelemetry() (telemetry.Logger, telemetry.Tracer) {
	driver := zerolog.Nop()
	logger := telemetry.NewZerologLogger("", nil, &zerodriver.Logger{Logger: &driver}, nil)
	return logger, telemetry.NewOtelLogger("", logger)
}

func rateLimitTestAuth(c fiber.Ctx) error {
	user := c.Get("test-user", "user")
	c.Locals(ContextKeyAuthUserID, entities.AuthContext{ID: entities.UserID(user), Email: "user@example.com"})
	return c.Next()
}

func rateLimitTestApp(repo repositories.UserRepository, counter rateLimitCounter, auth fiber.Handler, handlerError error) *fiber.App {
	logger, tracer := rateLimitTestTelemetry()
	app := fiber.New()
	app.Use(RateLimit(tracer, logger, counter, repo, []string{"/v1/events"}))
	app.Use(auth)
	app.Use(func(c fiber.Ctx) error {
		if handlerError != nil {
			return handlerError
		}
		return c.SendStatus(http.StatusOK)
	})
	return app
}
