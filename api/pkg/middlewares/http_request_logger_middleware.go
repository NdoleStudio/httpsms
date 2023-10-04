package middlewares

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

const (
	clientVersionHeader = "X-Client-Version"
)

// HTTPRequestLogger adds a trace for an HTTP request
func HTTPRequestLogger(tracer telemetry.Tracer, logger telemetry.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, span, ctxLogger := tracer.StartFromFiberCtxWithLogger(c, logger)
		defer span.End()

		ctxLogger.WithString("http.method", c.Method()).
			WithString("http.path", c.Path()).
			WithString("client.version", c.Get(clientVersionHeader)).
			Trace(fmt.Sprintf("%s %s", c.Method(), c.OriginalURL()))

		response := c.Next()

		statusCode := c.Response().StatusCode()
		span.AddEvent(fmt.Sprintf("finished handling request with traceID: [%s], statusCode: [%d]", span.SpanContext().TraceID().String(), statusCode))
		if statusCode >= 300 && len(c.Request().Body()) > 0 {
			ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("http.status [%d], body [%s]", statusCode, string(c.Request().Body()))))
		}

		return response
	}
}
