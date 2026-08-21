package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm/clause"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"
	"gorm.io/gorm"
)

// gormMessageThreadRepository is responsible for persisting entities.MessageThread
type gormMessageThreadRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
	now    func() time.Time
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
		now:    time.Now,
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
	return updates
}

func messageThreadDeletedUpdates(params MessageThreadDeletedUpdate) map[string]any {
	if !params.UpdateLastMessage {
		return map[string]any{}
	}
	return map[string]any{
		"last_message_id":      params.LastMessageID,
		"last_message_content": params.LastMessageContent,
		"status":               params.LastMessageStatus,
	}
}

func messageThreadStatusUpdates(params MessageThreadStatusUpdate, readAt time.Time) map[string]any {
	updates := make(map[string]any)
	if params.IsArchived != nil {
		updates["is_archived"] = *params.IsArchived
	}
	if params.UnreadCount != nil {
		updates["unread_count"] = 0
		updates["last_read_at"] = readAt
	}
	return updates
}

func lockMessageThread(tx *gorm.DB, userID entities.UserID, threadID uuid.UUID) (*entities.MessageThread, error) {
	thread := new(entities.MessageThread)
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		Where("id = ?", threadID).
		First(thread).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, stacktrace.PropagateWithCodef(
			err,
			ErrCodeNotFound,
			"message thread with ID [%s] for user [%s] does not exist",
			threadID,
			userID,
		)
	}
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot lock message thread with ID [%s] for user [%s]", threadID, userID)
	}
	return thread, nil
}

func lockMessageThreadByConversation(tx *gorm.DB, userID entities.UserID, owner string, contact string) (*entities.MessageThread, error) {
	thread := new(entities.MessageThread)
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		Where("owner = ?", owner).
		Where("contact = ?", contact).
		First(thread).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, stacktrace.PropagateWithCodef(
			err,
			ErrCodeNotFound,
			"message thread for user [%s], owner [%s], and contact [%s] does not exist",
			userID,
			owner,
			contact,
		)
	}
	if err != nil {
		return nil, stacktrace.Propagatef(
			err,
			"cannot lock message thread for user [%s], owner [%s], and contact [%s]",
			userID,
			owner,
			contact,
		)
	}
	return thread, nil
}

func insertUnreadItem(tx *gorm.DB, item entities.MessageThreadUnreadItem) (bool, error) {
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&item)
	if result.Error != nil {
		return false, stacktrace.Propagatef(
			result.Error,
			"cannot insert unread ledger item for message [%s] in thread [%s]",
			item.MessageID,
			item.MessageThreadID,
		)
	}
	return result.RowsAffected == 1, nil
}

func markUnreadItemDeleted(tx *gorm.DB, messageID uuid.UUID, threadID uuid.UUID) (bool, error) {
	result := tx.
		Model(&entities.MessageThreadUnreadItem{}).
		Where("message_id = ?", messageID).
		Where("message_thread_id = ?", threadID).
		Where("counted = ?", true).
		Update("counted", false)
	if result.Error != nil {
		return false, stacktrace.Propagatef(
			result.Error,
			"cannot mark unread ledger item deleted for message [%s] in thread [%s]",
			messageID,
			threadID,
		)
	}
	return result.RowsAffected == 1, nil
}

func applyMessageThreadActivity(tx *gorm.DB, thread *entities.MessageThread, params MessageThreadActivityUpdate) error {
	if err := tx.
		Model(thread).
		Where("user_id = ?", params.UserID).
		Where("id = ?", thread.ID).
		Updates(messageThreadActivityUpdates(params)).
		Error; err != nil {
		return stacktrace.Propagatef(
			err,
			"cannot update message activity for thread [%s] and user [%s]",
			thread.ID,
			params.UserID,
		)
	}
	if !params.CountAsUnread || !params.EventTimestamp.After(thread.LastReadAt) {
		return nil
	}

	inserted, err := insertUnreadItem(tx, entities.MessageThreadUnreadItem{
		MessageID:       params.MessageID,
		MessageThreadID: thread.ID,
	})
	if err != nil || !inserted {
		return err
	}

	if err := tx.
		Model(thread).
		Where("user_id = ?", params.UserID).
		Where("id = ?", thread.ID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + ?", 1)).
		Error; err != nil {
		return stacktrace.Propagatef(
			err,
			"cannot increment unread count for thread [%s] and user [%s]",
			thread.ID,
			params.UserID,
		)
	}
	thread.UnreadCount++
	return nil
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

	err := crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		thread, err := lockMessageThread(tx, params.UserID, params.MessageThreadID)
		if err != nil {
			return err
		}

		deleted, err := markUnreadItemDeleted(tx, params.DeletedMessageID, params.MessageThreadID)
		if err != nil {
			return err
		}
		if deleted {
			if err := tx.
				Model(thread).
				Where("user_id = ?", params.UserID).
				Where("id = ?", params.MessageThreadID).
				UpdateColumn("unread_count", gorm.Expr("GREATEST(unread_count - 1, 0)")).
				Error; err != nil {
				return stacktrace.Propagatef(
					err,
					"cannot decrement unread count for thread [%s] and user [%s]",
					params.MessageThreadID,
					params.UserID,
				)
			}
			if thread.UnreadCount > 0 {
				thread.UnreadCount--
			}
		}

		if !params.UpdateLastMessage {
			return nil
		}
		if err := tx.
			Model(thread).
			Where("user_id = ?", params.UserID).
			Where("id = ?", params.MessageThreadID).
			Updates(messageThreadDeletedUpdates(params)).
			Error; err != nil {
			return stacktrace.Propagatef(
				err,
				"cannot update deleted-message metadata for thread [%s] and user [%s]",
				params.MessageThreadID,
				params.UserID,
			)
		}
		return nil
	})
	if err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagatef(
				err,
				"cannot apply deleted message [%s] to thread [%s] for user [%s]",
				params.DeletedMessageID,
				params.MessageThreadID,
				params.UserID,
			),
		)
	}

	return nil
}

// Store a new entities.MessageThread
func (repository *gormMessageThreadRepository) Store(ctx context.Context, params MessageThreadStoreParams) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		candidate := *params.Thread
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&candidate)
		if result.Error != nil {
			return stacktrace.Propagatef(result.Error, "cannot insert message thread with ID [%s]", params.Thread.ID)
		}
		if result.RowsAffected == 0 {
			thread, err := lockMessageThreadByConversation(
				tx,
				params.Thread.UserID,
				params.Thread.Owner,
				params.Thread.Contact,
			)
			if err != nil {
				return err
			}
			if params.Thread.LastMessageID == nil {
				return stacktrace.NewErrorf(
					"cannot apply conflicting thread [%s] without a last message ID",
					params.Thread.ID,
				)
			}
			content := ""
			if params.Thread.LastMessageContent != nil {
				content = *params.Thread.LastMessageContent
			}
			return applyMessageThreadActivity(tx, thread, MessageThreadActivityUpdate{
				MessageThreadID: thread.ID,
				UserID:          params.Thread.UserID,
				Timestamp:       params.Thread.OrderTimestamp,
				MessageID:       *params.Thread.LastMessageID,
				Content:         content,
				Status:          params.Thread.Status,
				CountAsUnread:   params.CountAsUnread,
				EventTimestamp:  params.EventTimestamp,
			})
		}
		if !params.CountAsUnread {
			return nil
		}
		if params.Thread.LastMessageID == nil {
			return stacktrace.NewErrorf(
				"cannot store unread ledger item for new thread [%s] without a last message ID",
				params.Thread.ID,
			)
		}

		inserted, err := insertUnreadItem(tx, entities.MessageThreadUnreadItem{
			MessageID:       *params.Thread.LastMessageID,
			MessageThreadID: params.Thread.ID,
		})
		if err != nil {
			return err
		}
		if !inserted {
			return stacktrace.NewErrorf(
				"unread ledger item for message [%s] was not inserted for new thread [%s]",
				*params.Thread.LastMessageID,
				params.Thread.ID,
			)
		}
		return nil
	})
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot save message thread with ID [%s]", params.Thread.ID))
	}

	return nil
}

// UpdateActivity persists the last-message activity fields for a thread
func (repository *gormMessageThreadRepository) UpdateActivity(ctx context.Context, params MessageThreadActivityUpdate) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		thread, err := lockMessageThread(tx, params.UserID, params.MessageThreadID)
		if err != nil {
			return err
		}

		return applyMessageThreadActivity(tx, thread, params)
	})
	if err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagatef(
				err,
				"cannot update message activity for thread [%s] and user [%s]",
				params.MessageThreadID,
				params.UserID,
			),
		)
	}

	return nil
}

// UpdateStatus persists archive/unread status fields for a thread
func (repository *gormMessageThreadRepository) UpdateStatus(
	ctx context.Context,
	userID entities.UserID,
	messageThreadID uuid.UUID,
	params MessageThreadStatusUpdate,
) (*entities.MessageThread, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if params.UnreadCount != nil && *params.UnreadCount != 0 {
		return nil, repository.tracer.WrapErrorSpan(
			span,
			stacktrace.NewErrorf(
				"cannot set unread count to [%d] for thread [%s] and user [%s]: only zero is supported",
				*params.UnreadCount,
				messageThreadID,
				userID,
			),
		)
	}

	var thread *entities.MessageThread
	err := crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		thread = nil
		lockedThread, err := lockMessageThread(tx, userID, messageThreadID)
		if err != nil {
			return err
		}

		var readAt time.Time
		if params.UnreadCount != nil {
			readAt = repository.now().UTC()
		}
		updates := messageThreadStatusUpdates(params, readAt)
		if len(updates) > 0 {
			if err := tx.
				Model(lockedThread).
				Clauses(clause.Returning{}).
				Where("user_id = ?", userID).
				Where("id = ?", messageThreadID).
				Updates(updates).
				Error; err != nil {
				return stacktrace.Propagatef(
					err,
					"cannot update status for thread [%s] and user [%s]",
					messageThreadID,
					userID,
				)
			}
		}
		if params.IsArchived != nil {
			lockedThread.IsArchived = *params.IsArchived
		}
		if params.UnreadCount == nil {
			thread = lockedThread
			return nil
		}

		lockedThread.UnreadCount = *params.UnreadCount
		lockedThread.LastReadAt = readAt
		if err := tx.
			Where("message_thread_id = ?", messageThreadID).
			Delete(&entities.MessageThreadUnreadItem{}).
			Error; err != nil {
			return stacktrace.Propagatef(
				err,
				"cannot clear unread ledger items for thread [%s] and user [%s]",
				messageThreadID,
				userID,
			)
		}
		thread = lockedThread
		return nil
	})
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagatef(err, "cannot update status for thread [%s] and user [%s]", messageThreadID, userID),
		)
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
