package telemetry

import (
	"fmt"

	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

type zerologLogger struct {
	zerolog   *zerodriver.Logger
	span      trace.Span
	fields    map[string]string
	projectID string
}

// NewZerologLogger creates a new instance of the zerolog logger
func NewZerologLogger(projectID string, fields map[string]string, driver *zerodriver.Logger, span trace.Span) Logger {
	logger := &zerologLogger{
		zerolog:   driver,
		fields:    fields,
		projectID: projectID,
		span:      span,
	}

	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	return logger
}

// WithService creates a new structured zerolog logger instance with a service name
func (logger *zerologLogger) WithService(service string) Logger {
	return logger.WithString(string(semconv.ServiceNamespaceKey), service)
}

func (logger *zerologLogger) Printf(s string, i ...interface{}) {
	logger.decorateEvent(logger.zerolog.Info()).Msg(fmt.Sprintf(s, i...))
}

// WithString creates a new structured zerolog logger instance with a key value pair
func (logger *zerologLogger) WithString(key string, value string) Logger {
	return NewZerologLogger(
		logger.projectID,
		logger.addField(key, value),
		logger.zerolog,
		logger.span,
	)
}

// Info logs a new message with information level.
func (logger *zerologLogger) Info(value string) {
	logger.decorateEvent(logger.zerolog.Info()).Msg(value)
}

// Trace logs a new message with trace level.
func (logger *zerologLogger) Trace(value string) {
	logger.decorateEvent(logger.zerolog.Trace()).Msg(value)
}

// Warn logs a new message with warning level.
func (logger *zerologLogger) Warn(err error) {
	logger.markSpanForSampling()
	logger.decorateEvent(logger.zerolog.Warn()).Err(err).Send()
}

// Debug logs a new message with debug level.
func (logger *zerologLogger) Debug(value string) {
	logger.decorateEvent(logger.zerolog.Debug()).Msg(value)
}

// Fatal logs a new message with fatal level.
func (logger *zerologLogger) Fatal(err error) {
	logger.markSpanForSampling()
	logger.decorateEvent(logger.zerolog.Fatal()).Err(err).Send()
}

// Error logs an error
func (logger *zerologLogger) Error(err error) {
	logger.markSpanForSampling()
	logger.decorateEvent(logger.zerolog.Error()).Err(err).Send()
}

// WithSpan adds a span to a logger.
func (logger *zerologLogger) WithSpan(span trace.Span) Logger {
	return NewZerologLogger(
		logger.projectID,
		logger.fields,
		logger.zerolog,
		span,
	)
}

func (logger *zerologLogger) decorateEvent(event *zerodriver.Event) *zerolog.Event {
	if logger.span != nil {
		spanContext := logger.span.SpanContext()
		event.Str("trace_id", spanContext.TraceID().String())
		event.Str("span_id", spanContext.SpanID().String())
		event.Bool("trace_sampled", spanContext.IsSampled())
	}
	for key, value := range logger.fields {
		event.Str(key, value)
	}
	return event.Event
}

func (logger *zerologLogger) markSpanForSampling() {
	if logger.span != nil {
		logger.span.SetAttributes(mustSampleAttribute.Bool(true))
	}
}

func (logger *zerologLogger) addField(key string, value string) map[string]string {
	fields := map[string]string{}
	for oldKey, oldValue := range logger.fields {
		fields[oldKey] = oldValue
	}
	fields[key] = value
	return fields
}
