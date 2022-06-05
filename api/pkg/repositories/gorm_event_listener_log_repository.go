package repositories

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormEventListenerLogRepository is responsible for persisting entities.EventListenerLog
type gormEventListenerLogRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormEventListenerLogRepository creates the GORM version of the EventListenerLogRepository
func NewGormEventListenerLogRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) EventListenerLogRepository {
	return &gormEventListenerLogRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormEventRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// Save a new entities.Message
func (repository *gormEventListenerLogRepository) Store(ctx context.Context, message *entities.EventListenerLog) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.Create(message).Error; err != nil {
		msg := fmt.Sprintf("cannot save message with ID [%s]", message.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Has checks if an event has been handled
func (repository *gormEventListenerLogRepository) Has(ctx context.Context, eventID string, handler string) (bool, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	var exists bool
	err := repository.db.Model(&entities.EventListenerLog{}).
		Select("count(*) > 0").
		Where("event_id = ?", eventID).
		Where("handler = ?", handler).
		Find(&exists).
		Error
	if err != nil {
		msg := fmt.Sprintf("cannot check if log exists with event ID [%s] and handler [%s]", eventID, handler)
		return exists, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return exists, nil
}
