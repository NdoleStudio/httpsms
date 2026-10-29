package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type samplingSpanProcessor struct {
	sdktrace.SpanProcessor
	sampler sdktrace.Sampler
}

// NewSamplingSpanProcessor exports warning/error spans and 50% of other spans.
// The provider must record all spans so the decision can be made after they end.
func NewSamplingSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return &samplingSpanProcessor{
		SpanProcessor: sdktrace.NewBatchSpanProcessor(exporter),
		sampler:       sdktrace.TraceIDRatioBased(0.5),
	}
}

func (processor *samplingSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	if processor.sampler.ShouldSample(sdktrace.SamplingParameters{
		TraceID: span.SpanContext().TraceID(),
	}).Decision == sdktrace.RecordAndSample || isWarningOrErrorSpan(span) {
		processor.SpanProcessor.OnEnd(span)
	}
}

func isWarningOrErrorSpan(span sdktrace.ReadOnlySpan) bool {
	if span.Status().Code == codes.Error || hasWarningOrErrorSeverity(span.Attributes()) {
		return true
	}
	for _, event := range span.Events() {
		if event.Name == "exception" || hasWarningOrErrorSeverity(event.Attributes) {
			return true
		}
	}
	return false
}

func hasWarningOrErrorSeverity(attributes []attribute.KeyValue) bool {
	for _, value := range attributes {
		if value.Key == "log.severity" {
			switch value.Value.AsString() {
			case "warn", "warning", "error", "fatal":
				return true
			}
		}
	}
	return false
}
