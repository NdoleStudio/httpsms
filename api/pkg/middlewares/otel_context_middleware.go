package middlewares

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
		// Get the Context from the request
		spanContext, errors := spanContextFromHeader(c.Get(header))
		if len(errors) != 0 {
			logger.Error(stacktrace.NewError(strings.Join(errors, "\n")))
		}

		if !spanContext.IsValid() {
			if c.Get(header) != "" {
				logger.Error(stacktrace.NewError("invalid trace context %s creating new context", c.Get(header)))
			}
			otelTracer := otel.Tracer(namespace)
			_, span := otelTracer.Start(context.Background(), fmt.Sprintf("%s %s", c.Method(), c.OriginalURL()))
			defer span.End()
			spanContext = span.SpanContext()
		}

		logger.WithSpan(spanContext).
			WithString("http.method", c.Method()).
			WithString("client.version", c.Get(clientVersionHeader)).
			Trace(c.OriginalURL())

		newCtx, span := otel.Tracer(namespace).Start(trace.ContextWithRemoteSpanContext(context.Background(), spanContext), "middlewares.OtelTraceContext")
		defer span.End()

		ctxLogger := tracer.CtxLogger(logger, span)
		traceID := spanContext.TraceID().String()
		span.SetAttributes(attribute.Key("traceID").String(traceID))
		span.SetAttributes(attribute.Key("SpanID").String(span.SpanContext().SpanID().String()))
		span.SetAttributes(attribute.Key("traceFlags").String(spanContext.TraceFlags().String()))
		span.SetAttributes(attribute.Key("clientVersion").String(c.Get(clientVersionHeader)))

		c.Locals(telemetry.TracerContextKey, trace.ContextWithSpan(newCtx, span))

		// Go to next middleware:
		response := c.Next()

		statusCode := c.Response().StatusCode()
		span.AddEvent(fmt.Sprintf("finished handling request with traceID: [%s], statusCode: [%d]", traceID, statusCode))

		if statusCode >= 300 && len(c.Request().Body()) > 0 {
			ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("http.status [%d], body [%s]", statusCode, string(c.Request().Body()))))
		}

		return response
	}
}

func spanContextFromHeader(parentContext string) (trace.SpanContext, []string) {
	result := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{},
		SpanID:     trace.SpanID{},
		TraceState: trace.TraceState{},
		Remote:     true,
	})

	parts := strings.Split(parentContext, "/")
	var errors []string
	if len(parts) == 2 {
		traceID, err := trace.TraceIDFromHex(parts[0])
		if err != nil {
			errors = append(errors, fmt.Sprintf("could not get trace id %v", err))
			return result, errors
		}
		result = result.WithTraceID(traceID)

		spanParts := strings.Split(parts[1], ";")
		if len(spanParts) == 1 {
			spanParts = append(spanParts, "")
		}

		if len(spanParts) == 2 {
			val, err := strconv.ParseUint(spanParts[0], 10, 64)
			if err != nil {
				errors = append(errors, fmt.Sprintf("could not get trace id %v", err))
				return result, errors
			}

			spanID, err := trace.SpanIDFromHex(fmt.Sprintf("%016x", val))
			if err != nil {
				errors = append(errors, fmt.Sprintf("could not get span trace id %v", err))
				return result, errors
			}
			result = result.WithSpanID(spanID)

			if spanParts[1] == "o=1" {
				result = result.WithTraceFlags(trace.FlagsSampled)
			}
		}
	}

	return result, errors
}
