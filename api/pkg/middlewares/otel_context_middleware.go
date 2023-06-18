package middlewares

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	clientVersionHeader = "X-Client-Version"
)

// OtelTraceContext adds a trace for an HTTP request
func OtelTraceContext(tracer telemetry.Tracer, logger telemetry.Logger, header string, namespace string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		otelTracer := otel.Tracer(namespace)
		ctx, span := otelTracer.Start(context.Background(), fmt.Sprintf("%s %s", c.Method(), c.OriginalURL()), trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		spanContext := span.SpanContext()

		logger.WithSpan(spanContext).
			WithString("http.method", c.Method()).
			WithString("client.version", c.Get(clientVersionHeader)).
			Trace(c.OriginalURL())

		ctxLogger := tracer.CtxLogger(logger, span)
		span.SetAttributes(attribute.Key("traceID").String(span.SpanContext().TraceID().String()))
		span.SetAttributes(attribute.Key("SpanID").String(span.SpanContext().SpanID().String()))
		span.SetAttributes(attribute.Key("traceFlags").String(spanContext.TraceFlags().String()))
		span.SetAttributes(attribute.Key("clientVersion").String(c.Get(clientVersionHeader)))

		c.Locals(telemetry.TracerContextKey, trace.ContextWithSpan(ctx, span))

		// Go to next middleware:
		response := c.Next()

		statusCode := c.Response().StatusCode()
		span.AddEvent(fmt.Sprintf("finished handling request with traceID: [%s], statusCode: [%d]", span.SpanContext().TraceID().String(), statusCode))

		if statusCode >= 300 && len(c.Request().Body()) > 0 {
			ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("http.status [%d], body [%s]", statusCode, string(c.Request().Body()))))
		}

		return response
	}
}
