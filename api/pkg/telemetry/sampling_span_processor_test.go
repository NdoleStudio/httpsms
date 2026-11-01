package telemetry

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceSamplingSupportsUUIDTraceIDs(t *testing.T) {
	sampled := 0
	for index := range 1024 {
		var traceID trace.TraceID
		binary.BigEndian.PutUint64(traceID[8:], uint64(index))
		traceID[6] = 0x40
		traceID[8] |= 0x80
		if isSampledTrace(traceID) {
			sampled++
		}
	}
	require.InDelta(t, 512, sampled, 64)
}

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

func TestSamplingSpanProcessorTruncatedTelemetry(t *testing.T) {
	scenarios := map[string]func(*samplingTestSpan){
		"evicted exception": func(test *samplingTestSpan) {
			test.span.RecordError(errors.New("failed"))
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
		"dropped retention flag": func(test *samplingTestSpan) {
			test.fillAttributes()
			test.span.SetAttributes(attribute.Bool("httpsms.must_sample", true))
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

func TestSamplingSpanProcessorTruncatedLinks(t *testing.T) {
	link := trace.Link{SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{15: 2},
		SpanID:  trace.SpanID{7: 2},
	})}
	t.Run("dropped links", func(t *testing.T) {
		test := newSamplingTestSpan(t, false)
		for range 200 {
			test.span.AddLink(link)
		}
		test.finish(t, "test")
	})
	t.Run("dropped link attributes", func(t *testing.T) {
		var attributes []attribute.KeyValue
		for index := range 200 {
			attributes = append(attributes, attribute.Int(fmt.Sprintf("attribute.%d", index), index))
		}
		link.Attributes = attributes
		test := newSamplingTestSpan(t, false, trace.WithLinks(link))
		test.finish(t, "test")
	})
}

func TestMetricExemplarsOnlyReferenceSampledTraces(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithExemplarFilter(SampledTraceExemplarFilter),
	)
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	histogram, err := provider.Meter("test").Float64Histogram("duration")
	require.NoError(t, err)

	for _, sampled := range []bool{false, true} {
		traceID := trace.TraceID{6: 0x40, 8: 0x80, 15: 1}
		if sampled {
			traceID[15] = 2
		}
		ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     trace.SpanID{7: 1},
			TraceFlags: trace.FlagsSampled,
		}))
		histogram.Record(ctx, 10, metric.WithAttributes(attribute.Bool("sampled", sampled)))
	}

	var result metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &result))
	data, ok := result.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, data.DataPoints, 2)
	for _, point := range data.DataPoints {
		require.Equal(t, uint64(1), point.Count)
		require.Equal(t, float64(10), point.Sum)
		sampled, _ := point.Attributes.Value("sampled")
		if sampled.AsBool() {
			require.Len(t, point.Exemplars, 1)
		} else {
			require.Empty(t, point.Exemplars)
		}
	}
}

func TestSamplingSpanProcessorTruncatedEventAttributes(t *testing.T) {
	test := newSamplingTestSpan(t, false)
	var attributes []attribute.KeyValue
	for index := range 200 {
		attributes = append(attributes, attribute.Int(fmt.Sprintf("attribute.%d", index), index))
	}
	test.span.AddEvent("event", trace.WithAttributes(attributes...))
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

	// Fixed UUID-shaped IDs exercise both outcomes of the stable hash decision.
	traceID := trace.TraceID{6: 0x40, 8: 0x80, 15: 1}
	if sampled {
		traceID[15] = 2
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
