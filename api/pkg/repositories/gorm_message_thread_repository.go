package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"gorm.io/gorm/clause"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
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
	return map[string]any{
		"order_timestamp":      params.Timestamp,
		"last_message_id":      params.MessageID,
		"last_message_content": params.Content,
		"status":               string(params.Status),
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
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.MessageThread{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Delete the message thread for a user
func (repository *gormMessageThreadRepository) Delete(ctx context.Context, userID entities.UserID, messageThreadID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", messageThreadID).Delete(&entities.MessageThread{}).Error
	if err != nil {
		msg := fmt.Sprintf("cannot delete message thread with ID [%s] for user with ID [%s]", messageThreadID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
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
		Updates(map[string]any{
			"last_message_id":      params.LastMessageID,
			"last_message_content": params.LastMessageContent,
			"status":               string(params.LastMessageStatus),
		})
	if result.Error != nil {
		msg := fmt.Sprintf("cannot update deleted-message metadata for thread [%s]", params.MessageThreadID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(result.Error, msg))
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
		msg := fmt.Sprintf("cannot save message thread with ID [%s]", thread.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// UpdateActivity persists the last-message activity fields for a thread
func (repository *gormMessageThreadRepository) UpdateActivity(ctx context.Context, params MessageThreadActivityUpdate) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&entities.MessageThread{}).
			Where("user_id = ?", params.UserID).
			Where("id = ?", params.MessageThreadID)

		result := query.Updates(messageThreadActivityUpdates(params))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return stacktrace.PropagateWithCode(
				gorm.ErrRecordNotFound,
				ErrCodeNotFound,
				fmt.Sprintf("thread with id [%s] not found", params.MessageThreadID),
			)
		}

		if !params.MarksUnread {
			return nil
		}

		return tx.Model(&entities.MessageThread{}).
			Where("user_id = ?", params.UserID).
			Where("id = ?", params.MessageThreadID).
			Where("last_read_at < ?", params.EventTimestamp).
			Update("is_read", false).
			Error
	})
	if err != nil {
		msg := fmt.Sprintf("cannot update message activity for thread [%s]", params.MessageThreadID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	return nil
}

// UpdateStatus persists archive/read status fields for a thread
func (repository *gormMessageThreadRepository) UpdateStatus(
	ctx context.Context,
	userID entities.UserID,
	messageThreadID uuid.UUID,
	params MessageThreadStatusUpdate,
) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	result := repository.db.WithContext(ctx).
		Model(&entities.MessageThread{}).
		Where("user_id = ?", userID).
		Where("id = ?", messageThreadID).
		Updates(messageThreadStatusUpdates(params))
	if result.Error != nil {
		msg := fmt.Sprintf("cannot update status for thread [%s]", messageThreadID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(result.Error, msg))
	}
	if result.RowsAffected == 0 {
		msg := fmt.Sprintf("thread with id [%s] not found", messageThreadID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(gorm.ErrRecordNotFound, ErrCodeNotFound, msg))
	}

	return nil
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
		msg := fmt.Sprintf("thread with owner [%s] and contact [%s] does not exist", owner, contact)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load thread with owner [%s] and contact [%s]", owner, contact)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
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
		msg := fmt.Sprintf("thread with id [%s] not found", ID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("thread with id [%s]", ID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
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
		msg := fmt.Sprintf("cannot fetch message threads with owner [%s] and params [%+#v]", owner, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return threads, nil
}
