package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"gorm.io/gorm/clause"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"gorm.io/gorm"
)

// gormMessageThreadRepository is responsible for persisting entities.MessageThread
type gormMessageThreadRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormMessageThreadRepository creates the GORM version of the MessageRepository
func NewGormMessageThreadRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) MessageThreadRepository {
	return &gormMessageThreadRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormMessageThreadRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func messageThreadActivityUpdates(params MessageThreadActivityUpdate) map[string]any {
	updates := map[string]any{
		"order_timestamp":      params.Timestamp,
		"last_message_id":      params.MessageID,
		"last_message_content": params.Content,
		"status":               params.Status,
	}
	if params.Unarchive {
		updates["is_archived"] = false
	}
	if params.MarkAsUnread {
		updates["is_read"] = gorm.Expr(
			"CASE WHEN last_read_at < ? THEN ? ELSE is_read END",
			params.EventTimestamp,
			false,
		)
	}
	return updates
}

func messageThreadDeletedUpdates(params MessageThreadDeletedUpdate) map[string]any {
	return map[string]any{
		"last_message_id":      params.LastMessageID,
		"last_message_content": params.LastMessageContent,
		"status":               params.LastMessageStatus,
	}
}

func messageThreadStatusUpdates(params MessageThreadStatusUpdate) map[string]any {
	updates := make(map[string]any)
	if params.IsArchived != nil {
		updates["is_archived"] = *params.IsArchived
	}
	if params.IsRead != nil {
		updates["is_read"] = *params.IsRead
		if *params.IsRead {
			updates["last_read_at"] = params.ReadAt
		}
	}
	return updates
}

func (repository *gormMessageThreadRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.MessageThread{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all [%T] for user with ID [%s]", &entities.MessageThread{}, userID))
	}

	return nil
}

// Delete the message thread for a user
func (repository *gormMessageThreadRepository) Delete(ctx context.Context, userID entities.UserID, messageThreadID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", messageThreadID).Delete(&entities.MessageThread{}).Error
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete message thread with ID [%s] for user with ID [%s]", messageThreadID, userID))
	}

	return nil
}

// UpdateAfterDeletedMessage updates a thread after the original message has been deleted
func (repository *gormMessageThreadRepository) UpdateAfterDeletedMessage(ctx context.Context, params MessageThreadDeletedUpdate) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	result := repository.db.WithContext(ctx).
		Model(&entities.MessageThread{}).
		Where("user_id = ?", params.UserID).
		Where("id = ?", params.MessageThreadID).
		Updates(messageThreadDeletedUpdates(params))
	if result.Error != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(result.Error, "cannot update deleted-message metadata for thread [%s]", params.MessageThreadID))
	}

	return nil
}

// Store a new entities.MessageThread
func (repository *gormMessageThreadRepository) Store(ctx context.Context, thread *entities.MessageThread) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	isRead := thread.IsRead
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(thread)
		thread.IsRead = isRead
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 || isRead {
			return nil
		}

		return tx.Model(&entities.MessageThread{}).
			Where("user_id = ?", thread.UserID).
			Where("id = ?", thread.ID).
			UpdateColumn("is_read", false).
			Error
	})
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot save message thread with ID [%s]", thread.ID))
	}

	return nil
}

// UpdateActivity persists the last-message activity fields for a thread
func (repository *gormMessageThreadRepository) UpdateActivity(ctx context.Context, params MessageThreadActivityUpdate) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	result := repository.db.WithContext(ctx).
		Model(&entities.MessageThread{}).
		Where("user_id = ?", params.UserID).
		Where("id = ?", params.MessageThreadID).
		Updates(messageThreadActivityUpdates(params))
	if result.Error != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(result.Error, "cannot update message activity for thread [%s]", params.MessageThreadID))
	}
	if result.RowsAffected == 0 {
		return repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(gorm.ErrRecordNotFound, ErrCodeNotFound, "thread with id [%s] not found", params.MessageThreadID))
	}

	return nil
}

// UpdateStatus persists archive/read status fields for a thread
func (repository *gormMessageThreadRepository) UpdateStatus(
	ctx context.Context,
	userID entities.UserID,
	messageThreadID uuid.UUID,
	params MessageThreadStatusUpdate,
) (*entities.MessageThread, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	thread := new(entities.MessageThread)
	result := repository.db.WithContext(ctx).
		Model(thread).
		Clauses(clause.Returning{}).
		Where("user_id = ?", userID).
		Where("id = ?", messageThreadID).
		Updates(messageThreadStatusUpdates(params))
	if result.Error != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(result.Error, "cannot update status for thread [%s] and user [%s]", messageThreadID, userID))
	}
	if result.RowsAffected == 0 {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(gorm.ErrRecordNotFound, ErrCodeNotFound, "thread with id [%s] not found for user with ID [%s]", messageThreadID, userID))
	}

	return thread, nil
}

// LoadByOwnerContact a thread between 2 users
func (repository *gormMessageThreadRepository) LoadByOwnerContact(ctx context.Context, userID entities.UserID, owner string, contact string) (*entities.MessageThread, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	thread := new(entities.MessageThread)

	err := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Where("owner = ?", owner).
		Where("contact = ?", contact).
		First(thread).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "thread with owner [%s] and contact [%s] does not exist", owner, contact))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load thread with owner [%s] and contact [%s]", owner, contact))
	}

	return thread, nil
}

// Load an entities.MessageThread by ID
func (repository *gormMessageThreadRepository) Load(ctx context.Context, userID entities.UserID, ID uuid.UUID) (*entities.MessageThread, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	thread := new(entities.MessageThread)

	err := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", ID).
		First(thread).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "thread with id [%s] not found", ID))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "thread with id [%s]", ID))
	}

	return thread, nil
}

// Index message threads for an owner
func (repository *gormMessageThreadRepository) Index(ctx context.Context, userID entities.UserID, owner string, isArchived bool, params IndexParams) (*[]entities.MessageThread, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Where("owner = ?", owner)

	if isArchived {
		query.Where("is_archived = ?", isArchived)
	} else {
		query.Where(repository.db.Where("is_archived = ?", isArchived).Or("is_archived IS NULL"))
	}

	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query.Where(
			repository.db.Where("last_message_content ILIKE ?", queryPattern).
				Or("owner ILIKE ?", queryPattern).
				Or("contact ILIKE ?", queryPattern),
		)
	}

	threads := new([]entities.MessageThread)
	if err := query.Order("order_timestamp DESC").Limit(params.Limit).Offset(params.Skip).Find(&threads).Error; err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch message threads with owner [%s] and params [%+#v]", owner, params))
	}

	return threads, nil
}
