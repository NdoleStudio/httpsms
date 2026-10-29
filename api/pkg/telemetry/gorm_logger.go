package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/stacktrace"
	"gorm.io/gorm/logger"
)

type gormLogger struct {
	tracer Tracer
	logger Logger
}

// NewGormLogger creates a new instance of gormLogger
func NewGormLogger(tracer Tracer, logger Logger) logger.Interface {
	return &gormLogger{
		tracer: tracer,
		logger: logger,
	}
}

// LogMode log mode
func (gorm *gormLogger) LogMode(_ logger.LogLevel) logger.Interface {
	return gorm
}

func (gorm *gormLogger) Info(ctx context.Context, s string, i ...any) {
	gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Info(fmt.Sprintf(s, i...))
}

func (gorm *gormLogger) Warn(ctx context.Context, s string, i ...any) {
	gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Warn(fmt.Errorf(s, i...))
}

func (gorm *gormLogger) Error(ctx context.Context, s string, i ...any) {
	gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).Error(fmt.Errorf(s, i...))
}

func (gorm *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	l := gorm.tracer.CtxLogger(gorm.logger, gorm.tracer.Span(ctx)).WithString("latency", elapsed.String())
	sql, rows := fc()
	msg := fmt.Sprintf("[ROWS:%d][%s]", rows, sql)

	if err != nil {
		l.Error(stacktrace.Propagatef(err, "[ROWS:%d][%s]", rows, sql))
		return
	}

	l.Debug(msg)
}
