package repositories

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormMessageRepository is responsible for persisting entities.Message
type gormMessageRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormMessageRepository creates the GORM version of the MessageRepository
func NewGormMessageRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) MessageRepository {
	return &gormMessageRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormEventRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// Save a new entities.Message
func (repository *gormMessageRepository) Save(ctx context.Context, message *entities.Message) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.Create(message).Error; err != nil {
		msg := fmt.Sprintf("cannot save message with ID [%s]", message.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Load an entities.Message by ID
func (repository *gormMessageRepository) Load(ctx context.Context, messageID uuid.UUID) (*entities.Message, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	message := new(entities.Message)
	if err := repository.db.First(message, messageID).Error; err != nil {
		msg := fmt.Sprintf("cannot load message with ID [%s]", messageID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return message, nil
}
