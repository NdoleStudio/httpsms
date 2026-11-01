package telemetry

import (
	"context"
	"crypto/sha256"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	mustSampleAttribute       attribute.Key = "httpsms.must_sample"
	slowDatabaseSpanThreshold               = 200 * time.Millisecond
)

type samplingSpanProcessor struct {
	sdktrace.SpanProcessor
}

// NewSamplingSpanProcessor exports warning/error spans, database spans over 200ms,
// and 50% of other spans. Truncated spans are retained conservatively.
// The provider must record all spans so the decision can be made after they end.
func NewSamplingSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return &samplingSpanProcessor{
		SpanProcessor: sdktrace.NewBatchSpanProcessor(exporter),
	}
}

func (processor *samplingSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	if isSampledTrace(span.SpanContext().TraceID()) || shouldRetainSpan(span) {
		processor.SpanProcessor.OnEnd(span)
	}
}

// SampledTraceExemplarFilter links metrics only to the stable trace-ID sample.
func SampledTraceExemplarFilter(ctx context.Context) bool {
	spanContext := trace.SpanContextFromContext(ctx)
	return spanContext.IsValid() && spanContext.IsSampled() && isSampledTrace(spanContext.TraceID())
}

func isSampledTrace(traceID trace.TraceID) bool {
	// UUID version/variant bits must not bias the trace-ID decision.
	hash := sha256.Sum256(traceID[:])
	return hash[0] < 128
}

func shouldRetainSpan(span sdktrace.ReadOnlySpan) bool {
	// Drop counters survive buffer limits, unlike severity attributes and events.
	return span.DroppedEvents() > 0 ||
		span.DroppedAttributes() > 0 ||
		span.Status().Code == codes.Error ||
		hasRetentionFlag(span.Attributes()) ||
		hasExceptionalEvents(span.Events()) ||
		hasTruncatedLinks(span) ||
		isSlowDatabaseSpan(span)
}

func hasTruncatedLinks(span sdktrace.ReadOnlySpan) bool {
	if span.DroppedLinks() > 0 {
		return true
	}
	for _, link := range span.Links() {
		if link.DroppedAttributeCount > 0 {
			return true
		}
	}
	return false
}

func isSlowDatabaseSpan(span sdktrace.ReadOnlySpan) bool {
	if span.EndTime().Sub(span.StartTime()) <= slowDatabaseSpanThreshold {
		return false
	}
	for _, value := range span.Attributes() {
		switch value.Key {
		case "db.system.name", "db.system":
			if value.Value.AsString() != "" {
				return true
			}
		}
	}
	return false
}

func hasExceptionalEvents(events []sdktrace.Event) bool {
	for _, event := range events {
		if event.Name == "exception" || event.DroppedAttributeCount > 0 {
			return true
		}
	}
	return false
}

func hasRetentionFlag(attributes []attribute.KeyValue) bool {
	for _, value := range attributes {
		if value.Key == mustSampleAttribute && value.Value.AsBool() {
			return true
		}
	}
	return false
}
