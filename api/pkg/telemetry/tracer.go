package telemetry

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerContextKey stores the fiber trace context
	TracerContextKey = "tracer.context.key"
)

// Tracer is used for tracing
type Tracer interface {
	// StartFromFiberCtx creates a spanContext and a context.Context containing the newly-created spanContext.
	StartFromFiberCtx(c *fiber.Ctx, name ...string) (context.Context, trace.Span)

	// Start creates a spanContext and a context.Context containing the newly-created spanContext.
	Start(c context.Context, name ...string) (context.Context, trace.Span)

	StartWithLogger(c context.Context, logger Logger, name ...string) (context.Context, trace.Span, Logger)

	// CtxLogger creates a telemetry.Logger with spanContext attributes in the structured logger
	CtxLogger(logger Logger, span trace.Span) Logger

	// WrapErrorSpan sets a spanContext as error
	WrapErrorSpan(span trace.Span, err error) error

	// Span returns the trace.Span from context.Context
	Span(ctx context.Context) trace.Span
}
