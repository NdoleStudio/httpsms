package telemetry

import (
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
)

func TestSamplingSpanProcessor(t *testing.T) {
	t.Run("sampled trace", func(t *testing.T) {
		test := newSamplingTestSpan(t, true)
		test.finish(t, "test", "sibling")
	})
	t.Run("unsampled trace", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		test.finish(t)
	})
	t.Run("info log does not promote span", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		test.logger.Info("info")
		test.finish(t)
	})
}

func TestSamplingSpanProcessorErrors(t *testing.T) {
	t.Run("error status", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		test.span.SetStatus(codes.Error, "failed")
		test.finish(t, "test")
	})
	t.Run("exception without error status", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		test.span.RecordError(errors.New("failed"))
		test.finish(t, "test")
	})
}

func TestSamplingSpanProcessorLogs(t *testing.T) {
	logs := map[string]func(Logger){
		"warning": func(logger Logger) {
			logger.Warn(errors.New("warning"))
		},
		"decorated warning": func(logger Logger) {
			logger.WithString("key", "value").WithService("test").Warn(errors.New("warning"))
		},
		"error": func(logger Logger) {
			logger.Error(errors.New("failed"))
		},
		"nil warning": func(logger Logger) {
			logger.Warn(nil)
		},
	}
	for name, log := range logs {
		t.Run(name, func(t *testing.T) {
			test := newSamplingTestSpan(t, false)
			log(test.logger)
			test.finish(t, "test")
			exported := test.exporter.GetSpans()[0]
			require.Contains(t, exported.Attributes, attribute.Bool("httpsms.must_sample", true))
			require.Empty(t, exported.Events)
		})
	}
}

func TestSamplingSpanProcessorMustSample(t *testing.T) {
	t.Run("true retains span", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		test.span.SetAttributes(attribute.Bool("httpsms.must_sample", true))
		test.finish(t, "test")
	})
	t.Run("false does not retain span", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		test.span.SetAttributes(attribute.Bool("httpsms.must_sample", false))
		test.finish(t)
	})
}

func TestSamplingSpanProcessorGormLogs(t *testing.T) {
	t.Run("warning", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		NewGormLogger(test.tracer, test.logger).Warn(test.ctx, "warning %s", "message")
		test.finish(t, "test")
	})
	t.Run("query error", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		NewGormLogger(test.tracer, test.logger).Trace(test.ctx, time.Now(), func() (string, int64) {
			return "SELECT 1", 0
		}, errors.New("failed"))
		test.finish(t, "test")
	})
	t.Run("successful query does not promote span", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		NewGormLogger(test.tracer, test.logger).Trace(test.ctx, time.Now(), func() (string, int64) {
			return "SELECT 1", 1
		}, nil)
		test.finish(t)
	})
}

func TestSamplingSpanProcessorSlowDatabaseSpans(t *testing.T) {
	tests := []struct {
		name      string
		systemKey attribute.Key
		duration  time.Duration
		want      []string
	}{
		{name: "slow database", systemKey: "db.system.name", duration: 201 * time.Millisecond, want: []string{"test"}},
		{name: "legacy database attribute", systemKey: "db.system", duration: time.Second, want: []string{"test"}},
		{name: "at threshold", systemKey: "db.system.name", duration: 200 * time.Millisecond},
		{name: "fast database", systemKey: "db.system.name", duration: time.Millisecond},
		{name: "slow non-database", systemKey: "service.name", duration: time.Second},
	}
	for _, spec := range tests {
		t.Run(spec.name, func(t *testing.T) {
			start := time.Now()
			test := newSamplingTestSpan(t, false, trace.WithTimestamp(start))
			test.span.SetAttributes(spec.systemKey.String("postgresql"))
			test.span.End(trace.WithTimestamp(start.Add(spec.duration)))
			test.finish(t, spec.want...)
		})
	}
}

func TestSamplingSpanProcessorWarningEvent(t *testing.T) {
	test := newSamplingTestSpan(t, false)
	test.span.AddEvent("log", trace.WithAttributes(attribute.String("log.severity", "warn")))
	test.finish(t, "test")
}

func TestSamplingSpanProcessorUppercaseSeverity(t *testing.T) {
	for _, severity := range []string{"WARN", "WARNING", "ERROR", "FATAL"} {
		t.Run(severity, func(t *testing.T) {
			test := newSamplingTestSpan(t, false)
			test.span.AddEvent("log", trace.WithAttributes(attribute.String("log.severity", severity)))
			test.finish(t, "test")
		})
	}
}

func TestSamplingSpanProcessorTruncatedTelemetry(t *testing.T) {
	scenarios := map[string]func(*samplingTestSpan){
		"evicted exception": func(test *samplingTestSpan) {
			test.span.RecordError(errors.New("failed"))
			test.fillEvents()
		},
		"evicted warning event": func(test *samplingTestSpan) {
			test.span.AddEvent("log", trace.WithAttributes(attribute.String("log.severity", "warn")))
			test.fillEvents()
		},
		"warning after event limit": func(test *samplingTestSpan) {
			test.logger.Warn(errors.New("warning"))
			test.fillEvents()
		},
		"warning after attribute exhaustion and event eviction": func(test *samplingTestSpan) {
			test.fillAttributes()
			test.logger.Warn(errors.New("warning"))
			test.fillEvents()
		},
		"dropped warning attribute": func(test *samplingTestSpan) {
			test.fillAttributes()
			test.span.SetAttributes(attribute.String("log.severity", "warn"))
		},
	}
	for name, record := range scenarios {
		t.Run(name, func(t *testing.T) {
			test := newSamplingTestSpan(t, false)
			record(test)
			test.finish(t, "test")
		})
	}
}

func TestSamplingSpanProcessorTruncatedEventAttributes(t *testing.T) {
	test := newSamplingTestSpan(t, false)
	var attributes []attribute.KeyValue
	for index := range 200 {
		attributes = append(attributes, attribute.Int(fmt.Sprintf("attribute.%d", index), index))
	}
	attributes = append(attributes, attribute.String("log.severity", "warn"))
	test.span.AddEvent("log", trace.WithAttributes(attributes...))
	test.finish(t, "test")
}

type samplingTestSpan struct {
	ctx      context.Context
	span     trace.Span
	logger   Logger
	tracer   Tracer
	provider *sdktrace.TracerProvider
	exporter *tracetest.InMemoryExporter
}

func newSamplingTestSpan(t *testing.T, sampled bool, options ...trace.SpanStartOption) *samplingTestSpan {
	t.Helper()
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
	if sampled {
		traceID[8] = 0
	}
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  trace.SpanID{7: 1},
	}))
	ctx, span := provider.Tracer("test").Start(ctx, "test", options...)
	driver := zerolog.Nop()
	logger := NewZerologLogger("", nil, &zerodriver.Logger{Logger: &driver}, nil)
	tracer := NewOtelLogger("", logger)
	return &samplingTestSpan{
		ctx:      ctx,
		span:     span,
		logger:   tracer.CtxLogger(logger, span),
		tracer:   tracer,
		provider: provider,
		exporter: exporter,
	}
}

func (test *samplingTestSpan) finish(t *testing.T, want ...string) {
	t.Helper()
	test.span.End()
	// Normal siblings share the same 50% decision, but are not promoted by errors.
	_, sibling := test.provider.Tracer("test").Start(test.ctx, "sibling")
	sibling.End()
	require.NoError(t, test.provider.ForceFlush(context.Background()))

	var names []string
	for _, span := range test.exporter.GetSpans() {
		names = append(names, span.Name)
	}
	require.Equal(t, want, names)
}

func (test *samplingTestSpan) fillEvents() {
	for range 200 {
		test.span.AddEvent("info")
	}
}

func (test *samplingTestSpan) fillAttributes() {
	for index := range 200 {
		test.span.SetAttributes(attribute.Int(fmt.Sprintf("attribute.%d", index), index))
	}
}
