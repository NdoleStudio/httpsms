package telemetry

import (
	"go.opentelemetry.io/otel/trace"
)

type spanLogger struct {
	Logger
	span trace.Span
}

func (logger *spanLogger) WithService(service string) Logger {
	return &spanLogger{Logger: logger.Logger.WithService(service), span: logger.span}
}

func (logger *spanLogger) WithString(key string, value string) Logger {
	return &spanLogger{Logger: logger.Logger.WithString(key, value), span: logger.span}
}

func (logger *spanLogger) Warn(err error) {
	logger.span.SetAttributes(mustSampleAttribute.Bool(true))
	logger.callerLogger().Warn(err)
}

func (logger *spanLogger) Error(err error) {
	logger.span.SetAttributes(mustSampleAttribute.Bool(true))
	logger.callerLogger().Error(err)
}

func (logger *spanLogger) Fatal(err error) {
	logger.span.SetAttributes(mustSampleAttribute.Bool(true))
	logger.callerLogger().Fatal(err)
}

func (logger *spanLogger) callerLogger() Logger {
	if callerLogger, ok := logger.Logger.(interface{ withCallerSkipFrame(int) Logger }); ok {
		return callerLogger.withCallerSkipFrame(1)
	}
	return logger.Logger
}
