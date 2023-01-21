package telemetry

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/palantir/stacktrace"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type otelTracer struct {
	projectID string
	logger    Logger
}

// NewOtelLogger creates a new Tracer
func NewOtelLogger(projectID string, logger Logger) Tracer {
	return &otelTracer{
		projectID: projectID,
		logger:    logger,
	}
}

func (tracer *otelTracer) StartFromFiberCtxWithLogger(c *fiber.Ctx, logger Logger, name ...string) (context.Context, trace.Span, Logger) {
	ctx, span := tracer.StartFromFiberCtx(c, getName(name...))
	return ctx, span, tracer.CtxLogger(logger, span)
}

func (tracer *otelTracer) StartFromFiberCtx(c *fiber.Ctx, name ...string) (context.Context, trace.Span) {
	parentCtx, ok := c.Locals(TracerContextKey).(context.Context)
	if !ok {
		tracer.logger.Error(stacktrace.NewError(fmt.Sprintf("could not get trace from context with key [%s] url[%s] method [%s]", TracerContextKey, c.OriginalURL(), c.Method())))
		ctx, span := trace.NewNoopTracerProvider().Tracer("").Start(context.Background(), "")
		defer span.End()
		parentCtx = ctx
	}
	return tracer.Start(parentCtx, getName(name...))
}

func (tracer *otelTracer) CtxLogger(logger Logger, span trace.Span) Logger {
	return logger.WithSpan(span.SpanContext())
}

func (tracer *otelTracer) StartWithLogger(c context.Context, logger Logger, name ...string) (context.Context, trace.Span, Logger) {
	ctx, span := tracer.Start(c, getName(name...))
	return ctx, span, tracer.CtxLogger(logger, span)
}

func (tracer *otelTracer) Start(c context.Context, name ...string) (context.Context, trace.Span) {
	parentSpan := trace.SpanFromContext(c)
	ctx, span := parentSpan.TracerProvider().Tracer("").Start(c, getName(name...))

	span.SetAttributes(attribute.Key("traceID").String(parentSpan.SpanContext().TraceID().String()))
	span.SetAttributes(attribute.Key("SpanID").String(span.SpanContext().SpanID().String()))
	span.SetAttributes(attribute.Key("traceFlags").String(parentSpan.SpanContext().TraceFlags().String()))

	return ctx, span
}

// Span returns the trace.Span from context.Context
func (tracer *otelTracer) Span(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func (tracer *otelTracer) WrapErrorSpan(span trace.Span, err error) error {
	if err == nil {
		return nil
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, strings.Split(err.Error(), "\n")[0])

	return err
}

func getName(name ...string) string {
	if len(name) > 0 {
		return name[0]
	}
	return functionName()
}

func functionName() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(4, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()

	return strings.ReplaceAll(frame.Function, "github.com/NdoleStudio/http-sms-manager/", "")
}
