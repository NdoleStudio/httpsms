package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdlog "log"
	"runtime"
	"testing"
	"time"

	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestGormWarningLogger(t *testing.T) {
	tests := []struct {
		name     string
		elapsed  time.Duration
		err      error
		logLevel gormlogger.LogLevel
		want     string
	}{
		{name: "fast query", elapsed: time.Millisecond, logLevel: gormlogger.Warn},
		{name: "slow query", elapsed: time.Second, logLevel: gormlogger.Warn, want: "warn"},
		{name: "query error", elapsed: time.Millisecond, err: errors.New("query failed"), logLevel: gormlogger.Warn, want: "error"},
		{name: "record not found", elapsed: time.Millisecond, err: gorm.ErrRecordNotFound, logLevel: gormlogger.Warn, want: "error"},
		{name: "silent query error", elapsed: time.Second, err: errors.New("query failed"), logLevel: gormlogger.Silent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			driver := zerolog.New(&output)
			logger := NewZerologLogger("", nil, &zerodriver.Logger{Logger: &driver}, nil)
			tracer := NewOtelLogger("", logger)
			queryLogger := NewGormLoggerWithConfig(tracer, logger, gormlogger.Config{
				SlowThreshold: 200 * time.Millisecond,
				LogLevel:      gormlogger.Warn,
			}).LogMode(test.logLevel)
			formatted := false
			queryLogger.Trace(context.Background(), time.Now().Add(-test.elapsed), func() (string, int64) {
				formatted = true
				return "SELECT 1", 1
			}, test.err)

			require.Equal(t, test.want != "", formatted)
			assertGormQueryLog(t, output.Bytes(), test.want, test.err)
		})
	}
}

func assertGormQueryLog(t *testing.T, output []byte, wantLevel string, err error) {
	t.Helper()
	if wantLevel == "" {
		require.Empty(t, output)
		return
	}
	var record struct {
		Level string `json:"level"`
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(output, &record))
	require.Equal(t, wantLevel, record.Level)
	require.Contains(t, record.Error, "SELECT 1")
	if err != nil {
		require.Contains(t, record.Error, err.Error())
		return
	}
	require.Contains(t, record.Error, "SLOW SQL")
}

func TestGormSpanLoggerPreservesDelegate(t *testing.T) {
	tests := []struct {
		name    string
		elapsed time.Duration
		err     error
		level   gormlogger.LogLevel
		message string
		spans   []string
	}{
		{name: "fast query", elapsed: time.Millisecond, level: gormlogger.Warn},
		{name: "slow query", elapsed: time.Second, level: gormlogger.Warn, message: "SLOW SQL", spans: []string{"test"}},
		{name: "query error", err: errors.New("original query error"), level: gormlogger.Warn, message: "original query error", spans: []string{"test"}},
		{name: "info query", elapsed: time.Millisecond, level: gormlogger.Info, message: "SELECT 1"},
		{name: "silent error", err: errors.New("query error"), level: gormlogger.Silent},
	}
	for _, spec := range tests {
		t.Run(spec.name, func(t *testing.T) {
			test := newSamplingTestSpan(t, false)
			queryLogger, output := newDelegatedGormLogger(test)
			formatted := false
			queryLogger.LogMode(spec.level).Trace(test.ctx, time.Now().Add(-spec.elapsed), func() (string, int64) {
				formatted = true
				return "SELECT 1", 1
			}, spec.err)

			test.finish(t, spec.spans...)
			require.Equal(t, spec.message != "", formatted)
			assertDelegatedGormOutput(t, output.String(), spec.message)
		})
	}
}

func TestGormSpanLoggerDirectWarningsAndErrors(t *testing.T) {
	logs := map[string]func(gormlogger.Interface, context.Context){
		"warning": func(logger gormlogger.Interface, ctx context.Context) {
			logger.Warn(ctx, "original %s", "message")
		},
		"error": func(logger gormlogger.Interface, ctx context.Context) {
			logger.Error(ctx, "original %s", "message")
		},
	}
	for name, log := range logs {
		t.Run(name, func(t *testing.T) {
			test := newSamplingTestSpan(t, false)
			logger, output := newDelegatedGormLogger(test)
			log(logger, test.ctx)
			test.finish(t, "test")
			require.Contains(t, output.String(), "original message")
		})
	}
}

func newDelegatedGormLogger(test *samplingTestSpan) (gormlogger.Interface, *bytes.Buffer) {
	output := &bytes.Buffer{}
	config := gormlogger.Config{SlowThreshold: 200 * time.Millisecond, LogLevel: gormlogger.Warn}
	return NewGormSpanLogger(test.tracer, stdlog.New(output, "", 0), config), output
}

func assertDelegatedGormOutput(t *testing.T, output string, message string) {
	t.Helper()
	if message == "" {
		require.Empty(t, output)
		return
	}
	require.Contains(t, output, message)
	require.Contains(t, output, "SELECT 1")
}

func TestGormSpanLoggerPreservesCaller(t *testing.T) {
	for _, severity := range []string{"warn", "error", "slow query", "query error"} {
		t.Run(severity, func(t *testing.T) {
			test := newSamplingTestSpan(t, false)
			logger, output := newDelegatedGormLogger(test)
			caller := emitGormLogAtCaller(logger, test.ctx, severity)
			require.Contains(t, output.String(), caller)
			test.finish(t, "test")
		})
	}
}

func emitGormLogAtCaller(logger gormlogger.Interface, ctx context.Context, severity string) string {
	query := func() (string, int64) { return "SELECT 1", 1 }
	switch severity {
	case "warn":
		_, file, line, _ := runtime.Caller(0)
		logger.Warn(ctx, "warning")
		return fmt.Sprintf("%s:%d", file, line+1)
	case "error":
		_, file, line, _ := runtime.Caller(0)
		logger.Error(ctx, "error")
		return fmt.Sprintf("%s:%d", file, line+1)
	case "slow query":
		_, file, line, _ := runtime.Caller(0)
		logger.Trace(ctx, time.Now().Add(-time.Second), query, nil)
		return fmt.Sprintf("%s:%d", file, line+1)
	default:
		_, file, line, _ := runtime.Caller(0)
		logger.Trace(ctx, time.Now(), query, errors.New("query error"))
		return fmt.Sprintf("%s:%d", file, line+1)
	}
}

func TestGormLoggerParameterizedQueries(t *testing.T) {
	for _, delegated := range []bool{false, true} {
		for _, parameterized := range []bool{false, true} {
			t.Run(fmt.Sprintf("delegated=%t/parameterized=%t", delegated, parameterized), func(t *testing.T) {
				test := newSamplingTestSpan(t, false)
				config := gormlogger.Config{ParameterizedQueries: parameterized, LogLevel: gormlogger.Warn}
				logger := newParameterFilterLogger(test, config, delegated)
				filter, ok := logger.(interface {
					ParamsFilter(context.Context, string, ...interface{}) (string, []interface{})
				})
				require.True(t, ok)
				sql, params := filter.ParamsFilter(test.ctx, "SELECT ?", "example-value")
				require.Equal(t, "SELECT ?", sql)
				assertGormQueryParams(t, params, parameterized)
				test.finish(t)
			})
		}
	}
}

func newParameterFilterLogger(test *samplingTestSpan, config gormlogger.Config, delegated bool) gormlogger.Interface {
	if delegated {
		return NewGormSpanLogger(test.tracer, stdlog.New(&bytes.Buffer{}, "", 0), config)
	}
	return NewGormLoggerWithConfig(test.tracer, test.logger, config)
}

func assertGormQueryParams(t *testing.T, params []interface{}, parameterized bool) {
	t.Helper()
	if parameterized {
		require.Empty(t, params)
		return
	}
	require.Equal(t, []interface{}{"example-value"}, params)
}
