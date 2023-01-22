package telemetry

import (
	"fmt"

	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

type zerologLogger struct {
	zerolog     *zerodriver.Logger
	spanContext *trace.SpanContext
	fields      map[string]string
	projectID   string
	level       zerolog.Level
}

// NewZerologLogger creates a new instance of the zerolog logger
func NewZerologLogger(projectID string, fields map[string]string, driver *zerodriver.Logger, span *trace.SpanContext) Logger {
	logger := &zerologLogger{
		zerolog:     driver,
		fields:      fields,
		projectID:   projectID,
		spanContext: span,
	}

	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	return logger
}

// WithService creates a new structured zerolog logger instance with a service name
func (logger *zerologLogger) WithService(service string) Logger {
	return NewZerologLogger(
		logger.projectID,
		logger.addField(string(semconv.ServiceNameKey), service),
		logger.zerolog,
		logger.spanContext,
	)
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
		logger.spanContext,
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
	logger.decorateEvent(logger.zerolog.Warn()).Err(err).Send()
}

// Debug logs a new message with debug level.
func (logger *zerologLogger) Debug(value string) {
	logger.decorateEvent(logger.zerolog.Debug()).Msg(value)
}

// Fatal logs a new message with fatal level.
func (logger *zerologLogger) Fatal(err error) {
	logger.decorateEvent(logger.zerolog.Fatal()).Err(err).Send()
}

// Error logs an error
func (logger *zerologLogger) Error(err error) {
	logger.decorateEvent(logger.zerolog.Error()).Err(err).Send()
}

// WithSpan adds a spanContext to a logger
func (logger *zerologLogger) WithSpan(spanContext trace.SpanContext) Logger {
	return NewZerologLogger(
		logger.projectID,
		logger.fields,
		logger.zerolog,
		&spanContext,
	)
}

func (logger *zerologLogger) decorateEvent(event *zerodriver.Event) *zerolog.Event {
	if logger.spanContext != nil {
		event.TraceContext(logger.spanContext.TraceID().String(), logger.spanContext.SpanID().String(), logger.spanContext.IsSampled(), logger.projectID)
	}
	for key, value := range logger.fields {
		event.Str(key, value)
	}
	return event.Event
}

func (logger *zerologLogger) addField(key string, value string) map[string]string {
	fields := map[string]string{}
	for oldKey, oldValue := range logger.fields {
		fields[oldKey] = oldValue
	}
	fields[key] = value
	return fields
}
