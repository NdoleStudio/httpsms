package telemetry

import (
	"go.opentelemetry.io/otel/trace"
)

// Logger is an interface for creating customer logger implementations
type Logger interface {
	// Error logs an error
	Error(err error)

	// WithService creates a new structured logger instance with a service name
	WithService(string) Logger

	// WithString creates a new structured logger instance with a string
	WithString(key string, value string) Logger

	// WithSpan creates a new structured logger instance for a spanContext
	WithSpan(span trace.SpanContext) Logger

	// Trace logs a new message with trace level.
	Trace(value string)

	// Info logs a new message with information level.
	Info(value string)

	// Warn logs a new message with warning level.
	Warn(err error)

	// Debug logs a new message with debug level.
	Debug(value string)

	// Fatal logs a new message with fatal level.
	Fatal(err error)

	// Printf makes the logger compatible with retryablehttp.Logger
	Printf(string, ...interface{})
}
