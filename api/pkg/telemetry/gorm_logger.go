package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NdoleStudio/stacktrace"
	gormdb "gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type gormLogger struct {
	tracer Tracer
	logger Logger
	config logger.Config
}

// NewGormLogger creates a new instance of gormLogger
func NewGormLogger(tracer Tracer, log Logger) logger.Interface {
	return NewGormLoggerWithConfig(tracer, log, logger.Config{LogLevel: logger.Info})
}

// NewGormLoggerWithConfig creates a span-aware logger with GORM query logging settings.
func NewGormLoggerWithConfig(tracer Tracer, log Logger, config logger.Config) logger.Interface {
	return &gormLogger{
		tracer: tracer,
		logger: log,
		config: config,
	}
}

// LogMode log mode
func (gorm *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	clone := *gorm
	clone.config.LogLevel = level
	return &clone
}

func (gorm *gormLogger) Info(ctx context.Context, s string, i ...any) {
	if gorm.config.LogLevel >= logger.Info {
		gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Info(fmt.Sprintf(s, i...))
	}
}

func (gorm *gormLogger) Warn(ctx context.Context, s string, i ...any) {
	if gorm.config.LogLevel >= logger.Warn {
		gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Warn(fmt.Errorf(s, i...))
	}
}

func (gorm *gormLogger) Error(ctx context.Context, s string, i ...any) {
	if gorm.config.LogLevel >= logger.Error {
		gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Error(fmt.Errorf(s, i...))
	}
}

func (gorm *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if gorm.config.LogLevel <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	l := gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).WithString("latency", elapsed.String())
	sql, rows := fc()
	msg := fmt.Sprintf("[ROWS:%d][%s]", rows, sql)

	switch {
	case err != nil && gorm.config.LogLevel >= logger.Error && (!errors.Is(err, gormdb.ErrRecordNotFound) || !gorm.config.IgnoreRecordNotFoundError):
		l.Error(stacktrace.Propagatef(err, "[ROWS:%d][%s]", rows, sql))
	case gorm.config.SlowThreshold > 0 && elapsed > gorm.config.SlowThreshold && gorm.config.LogLevel >= logger.Warn:
		l.Warn(stacktrace.NewErrorf("SLOW SQL >= %s [ROWS:%d][%s]", gorm.config.SlowThreshold, rows, sql))
	case gorm.config.LogLevel >= logger.Info:
		l.Debug(msg)
	}
}
