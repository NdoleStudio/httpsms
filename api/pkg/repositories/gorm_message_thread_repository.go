package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"gorm.io/gorm/clause"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
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

// Store a new entities.MessageThread
func (repository *gormMessageThreadRepository) Store(ctx context.Context, thread *entities.MessageThread) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(thread).Error; err != nil {
		msg := fmt.Sprintf("cannot save message thread with ID [%s]", thread.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Update a new entities.MessageThread
func (repository *gormMessageThreadRepository) Update(ctx context.Context, thread *entities.MessageThread) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(thread).Error; err != nil {
		msg := fmt.Sprintf("cannot update message thread thread with ID [%s]", thread.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
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
