package repositories

import (
	"context"
	"errors"
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

// NewGormMessageRepository creates the GORM version of the MessageRepository
func NewGormMessageRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) MessageRepository {
	return &gormMessageRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormMessageRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// Index entities.Message between 2 parties
func (repository *gormMessageRepository) Index(ctx context.Context, userID entities.UserID, owner string, contact string, params IndexParams) (*[]entities.Message, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Where("owner = ?", owner).
		Where("contact =  ?", contact)
	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query.Where("content ILIKE ?", queryPattern)
	}

	messages := new([]entities.Message)
	if err := query.Order("order_timestamp DESC").Limit(params.Limit).Offset(params.Skip).Find(&messages).Error; err != nil {
		msg := fmt.Sprintf("cannot fetch messges with owner [%s] and contact [%s] and params [%+#v]", owner, contact, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return messages, nil
}

// Store a new entities.Message
func (repository *gormMessageRepository) Store(ctx context.Context, message *entities.Message) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Create(message).Error; err != nil {
		msg := fmt.Sprintf("cannot save message with ID [%s]", message.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Load an entities.Message by ID
func (repository *gormMessageRepository) Load(ctx context.Context, userID entities.UserID, messageID uuid.UUID) (*entities.Message, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	message := new(entities.Message)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", messageID).First(message).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("message with ID [%s] does not exist", message.ID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load message with ID [%s]", messageID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return message, nil
}

// Update an entities.Message
func (repository *gormMessageRepository) Update(ctx context.Context, message *entities.Message) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(message).Error; err != nil {
		msg := fmt.Sprintf("cannot update message with ID [%s]", message.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// GetOutstanding fetches messages that still to be sent to the phone
func (repository *gormMessageRepository) GetOutstanding(ctx context.Context, userID entities.UserID, messageID uuid.UUID) (*entities.Message, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	message := new(entities.Message)
	err := crdbgorm.ExecuteTx(ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			return tx.WithContext(ctx).Model(message).
				Clauses(clause.Returning{}).
				Where("user_id = ?", userID).
				Where("id = ?", messageID).
				Where("status = ?", entities.MessageStatusPending).
				Update("status", entities.MessageStatusSending).Error
		},
	)
	if err != nil {
		msg := fmt.Sprintf("cannot fetch outstanding message with userID [%s] and messageID [%s]", userID, messageID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return message, nil
}
