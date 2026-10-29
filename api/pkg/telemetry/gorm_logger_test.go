package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
			queryLogger.Trace(context.Background(), time.Now().Add(-test.elapsed), func() (string, int64) {
				return "SELECT 1", 1
			}, test.err)

			if test.want == "" {
				require.Empty(t, output.String())
				return
			}
			var record struct {
				Level string `json:"level"`
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, test.want, record.Level)
			require.Contains(t, record.Error, "SELECT 1")
			if test.err != nil {
				require.Contains(t, record.Error, test.err.Error())
			} else {
				require.Contains(t, record.Error, "SLOW SQL")
			}
		})
	}
}
