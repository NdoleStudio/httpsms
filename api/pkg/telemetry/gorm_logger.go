package telemetry

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/NdoleStudio/stacktrace"
	gormdb "gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type gormLogger struct {
	gormLogConfig
	tracer Tracer
	logger Logger
}

type gormLogConfig struct {
	logger.Config
}

// NewGormLogger creates a new instance of gormLogger
func NewGormLogger(tracer Tracer, log Logger) logger.Interface {
	return NewGormLoggerWithConfig(tracer, log, logger.Config{LogLevel: logger.Info})
}

// NewGormLoggerWithConfig creates a span-aware logger with GORM query logging settings.
func NewGormLoggerWithConfig(tracer Tracer, log Logger, config logger.Config) logger.Interface {
	return &gormLogger{
		gormLogConfig: gormLogConfig{Config: config},
		tracer:        tracer,
		logger:        log,
	}
}

// LogMode log mode
func (gorm *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	clone := *gorm
	clone.LogLevel = level
	return &clone
}

func (gorm *gormLogger) Info(ctx context.Context, s string, i ...any) {
	if gorm.LogLevel >= logger.Info {
		gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Info(fmt.Sprintf(s, i...))
	}
}

func (gorm *gormLogger) Warn(ctx context.Context, s string, i ...any) {
	if gorm.LogLevel >= logger.Warn {
		gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Warn(fmt.Errorf(s, i...))
	}
}

func (gorm *gormLogger) Error(ctx context.Context, s string, i ...any) {
	if gorm.LogLevel >= logger.Error {
		gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Error(fmt.Errorf(s, i...))
	}
}

func (gorm *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	level := gorm.queryLevel(elapsed, err)
	if level == logger.Silent {
		return
	}
	l := gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).WithString("latency", elapsed.String())
	sql, rows := fc()
	msg := fmt.Sprintf("[ROWS:%d][%s]", rows, sql)

	switch level {
	case logger.Error:
		l.Error(stacktrace.Propagatef(err, "[ROWS:%d][%s]", rows, sql))
	case logger.Warn:
		l.Warn(stacktrace.NewErrorf("SLOW SQL >= %s [ROWS:%d][%s]", gorm.SlowThreshold, rows, sql))
	default:
		l.Debug(msg)
	}
}

func (config gormLogConfig) queryLevel(elapsed time.Duration, err error) logger.LogLevel {
	if config.LogLevel <= logger.Silent {
		return logger.Silent
	}
	switch {
	case config.shouldLogError(err):
		return logger.Error
	case config.shouldLogSlowQuery(elapsed):
		return logger.Warn
	case config.LogLevel >= logger.Info:
		return logger.Info
	default:
		return logger.Silent
	}
}

func (config gormLogConfig) shouldLogError(err error) bool {
	if err == nil || config.LogLevel < logger.Error {
		return false
	}
	return !config.IgnoreRecordNotFoundError || !errors.Is(err, gormdb.ErrRecordNotFound)
}

func (config gormLogConfig) shouldLogSlowQuery(elapsed time.Duration) bool {
	return config.SlowThreshold > 0 &&
		elapsed > config.SlowThreshold &&
		config.LogLevel >= logger.Warn
}

func (config gormLogConfig) ParamsFilter(_ context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if config.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

type gormSpanLogger struct {
	logger.Interface
	gormLogConfig
	tracer Tracer
}

// NewGormSpanLogger marks warning/error spans while keeping GORM's standard output format.
func NewGormSpanLogger(tracer Tracer, writer logger.Writer, config logger.Config) logger.Interface {
	return &gormSpanLogger{
		Interface:     logger.New(&gormSourceWriter{Writer: writer}, config),
		gormLogConfig: gormLogConfig{Config: config},
		tracer:        tracer,
	}
}

func (gorm *gormSpanLogger) LogMode(level logger.LogLevel) logger.Interface {
	clone := *gorm
	clone.Interface = gorm.Interface.LogMode(level)
	clone.LogLevel = level
	return &clone
}

func (gorm *gormSpanLogger) Warn(ctx context.Context, message string, args ...any) {
	if gorm.LogLevel >= logger.Warn {
		gorm.tracer.Span(ctx).SetAttributes(mustSampleAttribute.Bool(true))
	}
	gorm.Interface.Warn(ctx, message, args...)
}

func (gorm *gormSpanLogger) Error(ctx context.Context, message string, args ...any) {
	if gorm.LogLevel >= logger.Error {
		gorm.tracer.Span(ctx).SetAttributes(mustSampleAttribute.Bool(true))
	}
	gorm.Interface.Error(ctx, message, args...)
}

func (gorm *gormSpanLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	gorm.Interface.Trace(ctx, begin, func() (string, int64) {
		switch gorm.queryLevel(time.Since(begin), err) {
		case logger.Error, logger.Warn:
			gorm.tracer.Span(ctx).SetAttributes(mustSampleAttribute.Bool(true))
		}
		return fc()
	}, err)
}

type gormSourceWriter struct {
	logger.Writer
}

func (writer *gormSourceWriter) Printf(format string, args ...interface{}) {
	if len(args) > 0 {
		if source, ok := args[0].(string); ok {
			args[0] = gormQueryCaller(source)
		}
	}
	writer.Writer.Printf(format, args...)
}

func gormQueryCaller(fallback string) string {
	_, adapterFile, _, _ := runtime.Caller(0)
	var callers [32]uintptr
	frames := runtime.CallersFrames(callers[:runtime.Callers(2, callers[:])])
	for {
		frame, more := frames.Next()
		if frame.File != adapterFile && !strings.HasPrefix(frame.Function, "gorm.io/gorm") {
			return fmt.Sprintf("%s:%d", frame.File, frame.Line)
		}
		if !more {
			return fallback
		}
	}
}
