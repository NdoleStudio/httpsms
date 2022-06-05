package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm/clause"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"
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

// GetOutstanding fetches messages that still to be sent to the phone
func (repository *gormMessageRepository) GetOutstanding(ctx context.Context, take int) (*[]entities.Message, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	messages := new([]entities.Message)
	err := crdbgorm.ExecuteTx(ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			return tx.Model(messages).
				Clauses(clause.Returning{}).
				Where(
					"id IN (?)",
					tx.Model(&entities.Message{}).
						Where("status = ?", entities.MessageStatusPending).
						Order("request_received_at ASC").
						Select("id").
						Limit(take),
				).
				Update("status", "sending").Error
		},
	)
	if err != nil {
		msg := fmt.Sprintf("cannot fetch [%d] outstanding messages", take)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return messages, nil
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
