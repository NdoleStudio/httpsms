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

func TestSpanLoggerPreservesCaller(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		for _, severity := range []string{"warn", "error", "info"} {
			t.Run(fmt.Sprintf("wrapped=%t/%s", wrapped, severity), func(t *testing.T) {
				logger, output := newCallerTestLogger(wrapped)
				wantCaller := emitLogAtCaller(logger, severity)
				assertCallerLog(t, output.Bytes(), wantCaller, severity)
			})
		}
	}
}

func newCallerTestLogger(wrapped bool) (Logger, *bytes.Buffer) {
	output := &bytes.Buffer{}
	driver := zerolog.New(output).With().CallerWithSkipFrameCount(3).Logger()
	logger := NewZerologLogger("", nil, &zerodriver.Logger{Logger: &driver}, nil)
	if wrapped {
		logger = NewOtelLogger("", logger).CtxLogger(logger, trace.SpanFromContext(context.Background()))
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
