package middlewares

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	clientVersionHeader          = "X-Client-Version"
	metricNameHTTPServerDuration = "http.server.duration"
)

// OtelTraceContext adds a trace for an HTTP request
func OtelTraceContext(tracer telemetry.Tracer, logger telemetry.Logger, resources *resource.Resource, namespace string) fiber.Handler {
	httpServerDuration, err := otel.GetMeterProvider().Meter(namespace).Int64Histogram(metricNameHTTPServerDuration, metric.WithUnit("ms"), metric.WithDescription("measures the duration inbound HTTP requests"))
	if err != nil {
		otel.Handle(err)
	}
	return func(c *fiber.Ctx) error {
		start := time.Now()
		otelTracer := otel.Tracer(namespace)
		ctx, span := otelTracer.Start(context.Background(), fmt.Sprintf("%s %s", c.Method(), fixURL(c.OriginalURL())), trace.WithSpanKind(trace.SpanKindServer))
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

		defer func() {
			attributes := append([]attribute.KeyValue{
				semconv.HTTPMethod(c.Method()),
				semconv.HTTPURL(fixURL(c.OriginalURL())),
			}, resources.Attributes()...)
			httpServerDuration.Record(ctx, time.Since(start).Milliseconds(), metric.WithAttributes(attributes...))
		}()

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

func fixURL(url string) string {
	url = strings.Split(url, "?")[0]
	parts := strings.Split(url, "/")
	var result []string
	for _, part := range parts {
		if _, err := uuid.Parse(part); err == nil {
			result = append(result, ":id")
		} else {
			result = append(result, part)
		}
	}
	return strings.Join(result, "/")
}
