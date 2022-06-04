package telemetry

// Logger is an interface for creating customer logger implementations
type Logger interface {
	// Error logs an error
	Error(err error)

	// WithService creates a new structured logger instance with a service name
	WithService(string) Logger

	// WithString creates a new structured logger instance with a key value pair
	WithString(key string, value string) Logger

	// WithBool creates a new structured logger instance with a key value pair
	WithBool(key string, value bool) Logger

	// Info logs a new message with information level.
	Info(value string)

	// Warn logs a new message with warning level.
	Warn(err error)

	// Trace logs a new message with trace level.
	Trace(value string)

	// Debug logs a new message with debug level.
	Debug(value string)

	// Fatal logs a new message with fatal level.
	Fatal(err error)
}
