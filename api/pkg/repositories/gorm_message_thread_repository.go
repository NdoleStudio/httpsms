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

func (repository *gormMessageThreadRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.MessageThread{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete all [%T] for user with ID [%s]", &entities.MessageThread{}, userID))
	}

	return nil
}

// Delete the message thread for a user
func (repository *gormMessageThreadRepository) Delete(ctx context.Context, userID entities.UserID, messageThreadID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", messageThreadID).Delete(&entities.MessageThread{}).Error
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete message thread with ID [%s] for user with ID [%s]", messageThreadID, userID))
	}

	return nil
}

// UpdateAfterDeletedMessage updates a thread after the original message has been deleted
func (repository *gormMessageThreadRepository) UpdateAfterDeletedMessage(ctx context.Context, userID entities.UserID, messageID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Model(&entities.MessageThread{}).
		Where("user_id = ?", userID).
		Where("last_message_id = ?", messageID).
		Updates(map[string]any{
			"last_message_id":      nil,
			"last_message_content": nil,
			"status":               entities.MessageStatusDeleted,
		}).Error
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot update thread after message is deleted with userID [%s] and messageID [%s]", userID, messageID))
	}

	return nil
}

// Store a new entities.MessageThread
func (repository *gormMessageThreadRepository) Store(ctx context.Context, thread *entities.MessageThread) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(thread).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot save message thread with ID [%s]", thread.ID))
	}

	return nil
}

// Update a new entities.MessageThread
func (repository *gormMessageThreadRepository) Update(ctx context.Context, thread *entities.MessageThread) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(thread).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot update message thread thread with ID [%s]", thread.ID))
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
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, "thread with owner [%s] and contact [%s] does not exist", owner, contact))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot load thread with owner [%s] and contact [%s]", owner, contact))
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
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, "thread with id [%s] not found", ID))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "thread with id [%s]", ID))
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
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot fetch message threads with owner [%s] and params [%+#v]", owner, params))
	}

	return threads, nil
}
