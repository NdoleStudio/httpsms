package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestHTTPRequestLoggerRedactsFCMTokenFromFailedRequestBody(t *testing.T) {
	logger := &requestLoggerRecordingLogger{}
	app := fiber.New()
	app.Use(HTTPRequestLogger(telemetry.NewOtelLogger("test", logger), logger))
	app.Put("/v1/phones", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusUnprocessableEntity)
	})
	body := `{"phone_number":"+18005550199","fcm_token":"https://adapter.example.com/secret?token=customer-secret"}`
	request := httptest.NewRequest(http.MethodPut, "/v1/phones", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request)

	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	logged := strings.Join(logger.warnings, "\n")
	assert.Contains(t, logged, "[redacted]")
	assert.NotContains(t, logged, "adapter.example.com")
	assert.NotContains(t, logged, "customer-secret")
}

type requestLoggerRecordingLogger struct {
	warnings []string
}

func (logger *requestLoggerRecordingLogger) Error(error) {}
func (logger *requestLoggerRecordingLogger) WithService(string) telemetry.Logger {
	return logger
}
func (logger *requestLoggerRecordingLogger) WithString(string, string) telemetry.Logger {
	return logger
}
func (logger *requestLoggerRecordingLogger) WithSpan(trace.SpanContext) telemetry.Logger {
	return logger
}
func (logger *requestLoggerRecordingLogger) Trace(string) {}
func (logger *requestLoggerRecordingLogger) Info(string)  {}
func (logger *requestLoggerRecordingLogger) Warn(err error) {
	logger.warnings = append(logger.warnings, err.Error())
}
func (logger *requestLoggerRecordingLogger) Debug(string)                  {}
func (logger *requestLoggerRecordingLogger) Fatal(error)                   {}
func (logger *requestLoggerRecordingLogger) Printf(string, ...interface{}) {}
