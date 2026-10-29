package telemetry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	gormlogger "gorm.io/gorm/logger"
)

func TestSamplingSpanProcessor(t *testing.T) {
	tests := []struct {
		name     string
		sampled  bool
		record   func(context.Context, trace.Span, Logger, Tracer)
		exported bool
	}{
		{name: "sampled trace", sampled: true, exported: true},
		{name: "unsampled trace"},
		{
			name: "error status",
			record: func(_ context.Context, span trace.Span, _ Logger, _ Tracer) {
				span.SetStatus(codes.Error, "failed")
			},
			exported: true,
		},
		{
			name: "exception without error status",
			record: func(_ context.Context, span trace.Span, _ Logger, _ Tracer) {
				span.RecordError(errors.New("failed"))
			},
			exported: true,
		},
		{
			name: "evicted exception",
			record: func(_ context.Context, span trace.Span, _ Logger, _ Tracer) {
				span.RecordError(errors.New("failed"))
				for range 200 {
					span.AddEvent("info")
				}
			},
			exported: true,
		},
		{
			name: "warning log",
			record: func(_ context.Context, _ trace.Span, logger Logger, _ Tracer) {
				logger.Warn(errors.New("warning"))
			},
			exported: true,
		},
		{
			name: "decorated warning log",
			record: func(_ context.Context, _ trace.Span, logger Logger, _ Tracer) {
				logger.WithString("key", "value").WithService("test").Warn(errors.New("warning"))
			},
			exported: true,
		},
		{
			name: "error log",
			record: func(_ context.Context, _ trace.Span, logger Logger, _ Tracer) {
				logger.Error(errors.New("failed"))
			},
			exported: true,
		},
		{
			name: "warning after event limit",
			record: func(_ context.Context, span trace.Span, logger Logger, _ Tracer) {
				logger.Warn(errors.New("warning"))
				for range 200 {
					span.AddEvent("info")
				}
			},
			exported: true,
		},
		{
			name: "warning after attribute exhaustion and event eviction",
			record: func(_ context.Context, span trace.Span, logger Logger, _ Tracer) {
				for index := range 200 {
					span.SetAttributes(attribute.Int(fmt.Sprintf("attribute.%d", index), index))
				}
				logger.Warn(errors.New("warning"))
				for range 200 {
					span.AddEvent("info")
				}
			},
			exported: true,
		},
		{
			name: "nil warning",
			record: func(_ context.Context, _ trace.Span, logger Logger, _ Tracer) {
				logger.Warn(nil)
			},
			exported: true,
		},
		{
			name: "gorm warning",
			record: func(ctx context.Context, _ trace.Span, logger Logger, tracer Tracer) {
				NewGormLogger(tracer, logger).Warn(ctx, "warning %s", "message")
			},
			exported: true,
		},
		{
			name: "gorm query error",
			record: func(ctx context.Context, _ trace.Span, logger Logger, tracer Tracer) {
				NewGormLogger(tracer, logger).Trace(ctx, time.Now(), func() (string, int64) {
					return "SELECT 1", 0
				}, errors.New("failed"))
			},
			exported: true,
		},
		{
			name: "gorm slow query warning",
			record: func(ctx context.Context, _ trace.Span, logger Logger, tracer Tracer) {
				NewGormLoggerWithConfig(tracer, logger, gormlogger.Config{
					SlowThreshold: 200 * time.Millisecond,
					LogLevel:      gormlogger.Warn,
				}).Trace(ctx, time.Now().Add(-time.Second), func() (string, int64) {
					return "SELECT 1", 1
				}, nil)
			},
			exported: true,
		},
		{
			name: "info log",
			record: func(_ context.Context, _ trace.Span, logger Logger, _ Tracer) {
				logger.Info("info")
			},
		},
		{
			name: "warning event",
			record: func(_ context.Context, span trace.Span, _ Logger, _ Tracer) {
				span.AddEvent("log", trace.WithAttributes(attribute.String("log.severity", "warn")))
			},
			exported: true,
		},
		{
			name: "evicted warning event",
			record: func(_ context.Context, span trace.Span, _ Logger, _ Tracer) {
				span.AddEvent("log", trace.WithAttributes(attribute.String("log.severity", "warn")))
				for range 200 {
					span.AddEvent("info")
				}
			},
			exported: true,
		},
		{
			name: "dropped warning attribute",
			record: func(_ context.Context, span trace.Span, _ Logger, _ Tracer) {
				for index := range 200 {
					span.SetAttributes(attribute.Int(fmt.Sprintf("attribute.%d", index), index))
				}
				span.SetAttributes(attribute.String("log.severity", "warn"))
			},
			exported: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithSpanProcessor(NewSamplingSpanProcessor(exporter)),
			)
			t.Cleanup(func() {
				require.NoError(t, provider.Shutdown(context.Background()))
			})

			// Ratio sampling uses the lower half of the trace ID.
			traceID := trace.TraceID{8: 0xff, 15: 1}
			if test.sampled {
				traceID[8] = 0
			}
			ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: traceID,
				SpanID:  trace.SpanID{7: 1},
			}))
			ctx, span := provider.Tracer("test").Start(ctx, "test")
			var output bytes.Buffer
			driver := zerolog.New(&output)
			logger := NewZerologLogger("", nil, &zerodriver.Logger{Logger: &driver}, nil)
			tracer := NewOtelLogger("", logger)
			if test.record != nil {
				test.record(ctx, span, tracer.CtxLogger(logger, span), tracer)
			}
			span.End()

			// Normal siblings share the same 50% decision, but are not promoted by errors.
			_, sibling := provider.Tracer("test").Start(ctx, "sibling")
			sibling.End()
			require.NoError(t, provider.ForceFlush(context.Background()))

			want := 0
			if test.exported {
				want++
			}
			if test.sampled {
				want++
			}
			spans := exporter.GetSpans()
			require.Len(t, spans, want)
			if test.exported {
				require.Equal(t, "test", spans[0].Name)
			}
		})
	}
}
