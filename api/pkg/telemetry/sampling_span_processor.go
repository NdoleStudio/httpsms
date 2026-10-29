package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const mustSampleAttribute attribute.Key = "httpsms.must_sample"

type samplingSpanProcessor struct {
	sdktrace.SpanProcessor
	sampler sdktrace.Sampler
}

// NewSamplingSpanProcessor exports warning/error spans and 50% of other spans.
// Spans with dropped telemetry are retained because their severity may be lost.
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
	}).Decision == sdktrace.RecordAndSample || shouldRetainSpan(span) {
		processor.SpanProcessor.OnEnd(span)
	}
}

func shouldRetainSpan(span sdktrace.ReadOnlySpan) bool {
	// Drop counters survive buffer limits, unlike severity attributes and events.
	return span.DroppedEvents() > 0 ||
		span.DroppedAttributes() > 0 ||
		span.Status().Code == codes.Error ||
		hasRetentionAttributes(span.Attributes()) ||
		hasExceptionalEvents(span.Events())
}

func hasExceptionalEvents(events []sdktrace.Event) bool {
	for _, event := range events {
		if event.Name == "exception" || event.DroppedAttributeCount > 0 || hasRetentionAttributes(event.Attributes) {
			return true
		}
	}
	return false
}

func hasRetentionAttributes(attributes []attribute.KeyValue) bool {
	for _, value := range attributes {
		if isRetentionAttribute(value) {
			return true
		}
	}
	return false
}

func isRetentionAttribute(value attribute.KeyValue) bool {
	if value.Key == mustSampleAttribute {
		return value.Value.AsBool()
	}
	if value.Key != "log.severity" {
		return false
	}
	switch value.Value.AsString() {
	case "warn", "warning", "error", "fatal":
		return true
	}
	return false
}
