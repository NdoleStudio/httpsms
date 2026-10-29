package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestLoggerWithSpanRetainsWarnings(t *testing.T) {
	test := newSamplingTestSpan(t, false)
	var output bytes.Buffer
	driver := zerolog.New(&output)
	logger := NewZerologLogger("", nil, &zerodriver.Logger{Logger: &driver}, nil).WithSpan(test.span)
	logger.WithString("field", "value").WithService("test").Warn(errors.New("warning"))
	test.finish(t, "test")

	var record struct {
		TraceID   string `json:"trace_id"`
		SpanID    string `json:"span_id"`
		Sampled   bool   `json:"trace_sampled"`
		Field     string `json:"field"`
		Namespace string `json:"service.namespace"`
		Error     string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(output.Bytes(), &record))
	require.Equal(t, test.span.SpanContext().TraceID().String(), record.TraceID)
	require.Equal(t, test.span.SpanContext().SpanID().String(), record.SpanID)
	require.True(t, record.Sampled)
	require.Equal(t, "value", record.Field)
	require.Equal(t, "test", record.Namespace)
	require.Equal(t, "warning", record.Error)
}

func TestZerologLoggerPreservesCaller(t *testing.T) {
	for _, withSpan := range []bool{false, true} {
		for _, severity := range []string{"warn", "error", "info"} {
			t.Run(fmt.Sprintf("with_span=%t/%s", withSpan, severity), func(t *testing.T) {
				logger, output := newCallerTestLogger(withSpan)
				wantCaller := emitLogAtCaller(logger, severity)
				assertCallerLog(t, output.Bytes(), wantCaller, severity)
			})
		}
	}
}

func newCallerTestLogger(withSpan bool) (Logger, *bytes.Buffer) {
	output := &bytes.Buffer{}
	driver := zerolog.New(output).With().CallerWithSkipFrameCount(3).Logger()
	logger := NewZerologLogger("", nil, &zerodriver.Logger{Logger: &driver}, nil)
	if withSpan {
		logger = logger.WithSpan(trace.SpanFromContext(context.Background()))
	}
	return logger.WithString("field", "value").WithService("test"), output
}

func assertCallerLog(t *testing.T, output []byte, wantCaller string, severity string) {
	t.Helper()
	var record struct {
		Caller  string `json:"caller"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(output, &record))
	require.Equal(t, wantCaller, record.Caller)
	if severity == "info" {
		require.Equal(t, "original message", record.Message)
		return
	}
	require.Equal(t, "original error", record.Error)
}

func emitLogAtCaller(logger Logger, severity string) string {
	switch severity {
	case "warn":
		_, file, line, _ := runtime.Caller(0)
		logger.Warn(errors.New("original error"))
		return fmt.Sprintf("%s:%d", file, line+1)
	case "error":
		_, file, line, _ := runtime.Caller(0)
		logger.Error(errors.New("original error"))
		return fmt.Sprintf("%s:%d", file, line+1)
	default:
		_, file, line, _ := runtime.Caller(0)
		logger.Info("original message")
		return fmt.Sprintf("%s:%d", file, line+1)
	}
}
