package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
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
	logger.recordLog("warn", err)
	logger.Logger.Warn(err)
}

func (logger *spanLogger) Error(err error) {
	logger.recordLog("error", err)
	logger.Logger.Error(err)
}

func (logger *spanLogger) Fatal(err error) {
	logger.recordLog("fatal", err)
	logger.Logger.Fatal(err)
}

func (logger *spanLogger) recordLog(severity string, err error) {
	attributes := []attribute.KeyValue{attribute.String("log.severity", severity)}
	logger.span.SetAttributes(attributes...)
	if err != nil {
		attributes = append(attributes, attribute.String("log.message", err.Error()))
	}
	logger.span.AddEvent("log", trace.WithAttributes(attributes...))
}
