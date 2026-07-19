package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm/clause"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"
	"github.com/google/uuid"
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

func (repository *gormMessageRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.Message{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all [%T] for user with ID [%s]", &entities.Message{}, userID))
	}

	return nil
}

// DeleteByOwnerAndContact deletes all the messages between and owner and a contact
func (repository *gormMessageRepository) DeleteByOwnerAndContact(ctx context.Context, userID entities.UserID, owner string, contact string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("owner = ?", owner).
		Where("contact = ?", contact).
		Delete(&entities.Message{}).
		Error
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete messages between owner [%s] and contact [%s] for user with ID [%s]", owner, contact, userID))
	}

	return nil
}

// Delete a message by the ID
func (repository *gormMessageRepository) Delete(ctx context.Context, userID entities.UserID, messageID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", messageID).Delete(&entities.Message{}).Error
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete message with ID [%s] for user with ID [%s]", messageID, userID))
	}

	return nil
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
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch messges with owner [%s] and contact [%s] and params [%+#v]", owner, contact, params))
	}

	return messages, nil
}

func (repository *gormMessageRepository) LastMessage(ctx context.Context, userID entities.UserID, owner string, contact string) (*entities.Message, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Where("owner = ?", owner).
		Where("contact =  ?", contact)

	message := new(entities.Message)

	err := query.Order("order_timestamp DESC").First(&message).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "cannot get last message for [%s] with owner [%s] and contact [%s]", userID, owner, contact))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot get last message for [%s] with owner [%s] and contact [%s]", userID, owner, contact))
	}

	return message, nil
}

func (repository *gormMessageRepository) Search(ctx context.Context, userID entities.UserID, owners []string, types []entities.MessageType, statuses []entities.MessageStatus, params IndexParams) ([]*entities.Message, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID)

	if len(owners) > 0 {
		query = query.Where("owner IN ?", owners)
	}
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		subQuery := repository.db.Where("content ILIKE ?", queryPattern).
			Or("contact ILIKE ?", queryPattern).
			Or("failure_reason ILIKE ?", queryPattern).
			Or("request_id ILIKE ?", queryPattern)

		if _, err := uuid.Parse(params.Query); err == nil {
			subQuery = subQuery.Or("id = ?", params.Query)
		}

		query = query.Where(subQuery)
	}

	messages := make([]*entities.Message, 0, params.Limit)
	err := query.Order(repository.order(params, "created_at")).
		Limit(params.Limit).
		Offset(params.Skip).
		Find(&messages).
		Error
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot search messages with for user [%s] params [%+#v]", userID, params))
	}

	return messages, nil
}

// GetBulkMessages fetches the last bulk message summaries for a user
func (repository *gormMessageRepository) GetBulkMessages(ctx context.Context, userID entities.UserID, limit int) ([]*entities.BulkMessage, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	orders := make([]*entities.BulkMessage, 0)
	err := repository.db.WithContext(ctx).Raw(`
		SELECT
			request_id,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'scheduled') as scheduled_count,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
			COUNT(*) FILTER (WHERE status = 'expired') as expired_count,
			COUNT(*) FILTER (WHERE status = 'sent') as sent_count,
			COUNT(*) FILTER (WHERE status = 'delivered') as delivered_count,
			MIN(created_at) as created_at
		FROM messages
		WHERE user_id = ? AND request_id LIKE 'bulk-%'
		GROUP BY request_id
		ORDER BY MIN(created_at) DESC
		LIMIT ?
	`, userID, limit).Scan(&orders).Error
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch bulk message orders for user [%s]", userID))
	}

	return orders, nil
}

// Store a new entities.Message
func (repository *gormMessageRepository) Store(ctx context.Context, message *entities.Message) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Create(message).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot save message with ID [%s]", message.ID))
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
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "message with ID [%s] and userID [%s] does not exist", messageID, userID))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load message with ID [%s]", messageID))
	}

	return message, nil
}

// Update an entities.Message
func (repository *gormMessageRepository) Update(ctx context.Context, message *entities.Message) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(message).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update message with ID [%s]", message.ID))
	}

	return nil
}

// GetOutstanding fetches messages that still to be sent to the phone
func (repository *gormMessageRepository) GetOutstanding(ctx context.Context, userID entities.UserID, messageID uuid.UUID, phoneNumbers []string) (*entities.Message, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	message := new(entities.Message)
	err := crdbgorm.ExecuteTx(
		ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			query := tx.WithContext(ctx).Model(message).
				Clauses(clause.Returning{}).
				Where("user_id = ?", userID).
				Where("id = ?", messageID)

			if len(phoneNumbers) > 0 {
				query = query.Where("owner IN ?", phoneNumbers)
			}

			return query.Where(repository.db.Where("status = ?", entities.MessageStatusScheduled).Or("status = ?", entities.MessageStatusPending).Or("status = ?", entities.MessageStatusExpired)).
				Update("status", entities.MessageStatusSending).Error
		},
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "outstanding message with ID [%s] and userID [%s] does not exist", messageID, userID))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch outstanding message with userID [%s] and messageID [%s]", userID, messageID))
	}

	if message == nil || message.ID == uuid.Nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.NewErrorWithCodef(ErrCodeNotFound, "outstanding message with ID [%s] and userID [%s] does not exist", messageID, userID))
	}

	return message, nil
}

func (repository *gormMessageRepository) order(params IndexParams, defaultSortBy string) string {
	sortBy := defaultSortBy
	if len(params.SortBy) > 0 {
		sortBy = params.SortBy
	}

	direction := "ASC"
	if params.SortDescending {
		direction = "DESC"
	}

	return fmt.Sprintf("%s %s", sortBy, direction)
}
