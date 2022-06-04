package telemetry

import (
	"github.com/rs/zerolog"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

type zerologLogger struct {
	zerolog zerolog.Logger
	writers zerolog.LevelWriter
	context zerolog.Context
}

// NewZerologLogger creates a new instance of the zerolog logger
func NewZerologLogger(ctx zerolog.Context) Logger {
	return &zerologLogger{
		zerolog: ctx.Logger(),
	}
}

// WithService creates a new structured zerolog logger instance with a service name
func (logger *zerologLogger) WithService(service string) Logger {
	return NewZerologLogger(logger.zerolog.With().Str(string(semconv.ServiceNameKey), service))
}

// Info logs a new message with information level.
func (logger *zerologLogger) Info(value string) {
	logger.zerolog.Info().Msg(value)
}

// Warn logs a new message with warning level.
func (logger *zerologLogger) Warn(value string) {
	logger.zerolog.Warn().Msg(value)
}

// Trace logs a new message with trace level.
func (logger *zerologLogger) Trace(value string) {
	logger.zerolog.Trace().Msg(value)
}

// Debug logs a new message with debug level.
func (logger *zerologLogger) Debug(value string) {
	logger.zerolog.Debug().Msg(value)
}

// Error logs an error
func (logger *zerologLogger) Error(err error) {
	logger.zerolog.Error().Err(err).Send()
}

// WithString creates a new structured logger instance with a key value pair
func (logger *zerologLogger) WithString(key string, value string) Logger {
	return NewZerologLogger(logger.zerolog.With().Str(key, value))
}
