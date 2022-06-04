package telemetry

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
)

// Tracer is used for tracing
type Tracer interface {
	// StartFromFiberCtx creates a span and a context.Context containing the newly-created span.
	StartFromFiberCtx(c *fiber.Ctx, name ...string) (context.Context, trace.Span)

	// Start creates a span and a context.Context containing the newly-created span.
	Start(c context.Context, name ...string) (context.Context, trace.Span)

	// CtxLogger creates a telemetry.Logger with span attributes in the structured logger
	CtxLogger(logger Logger, span trace.Span) Logger

	// WrapErrorSpan sets a span as error
	WrapErrorSpan(span trace.Span, err error) error
}
