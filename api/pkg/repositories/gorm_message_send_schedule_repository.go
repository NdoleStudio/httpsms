package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormMessageSendScheduleRepository persists and loads entities.MessageSendSchedule using GORM.
type gormMessageSendScheduleRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormMessageSendScheduleRepository creates a new GORM-backed MessageSendScheduleRepository.
func NewGormMessageSendScheduleRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) MessageSendScheduleRepository {
	return &gormMessageSendScheduleRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormMessageSendScheduleRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// Store saves a new message send schedule.
func (r *gormMessageSendScheduleRepository) Store(
	ctx context.Context,
	schedule *entities.MessageSendSchedule,
) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()

	if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
		return r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot store send schedule [%s]", schedule.ID))
	}

	return nil
}

// Update persists changes to an existing message send schedule.
func (r *gormMessageSendScheduleRepository) Update(
	ctx context.Context,
	schedule *entities.MessageSendSchedule,
) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()

	if err := r.db.WithContext(ctx).Save(schedule).Error; err != nil {
		return r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot update send schedule [%s]", schedule.ID))
	}

	return nil
}

// Load fetches a message send schedule by user ID and schedule ID.
func (r *gormMessageSendScheduleRepository) Load(
	ctx context.Context,
	userID entities.UserID,
	scheduleID uuid.UUID,
) (*entities.MessageSendSchedule, error) {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()

	item := new(entities.MessageSendSchedule)
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", scheduleID).
		First(item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, r.tracer.WrapErrorSpan(
			span,
			stacktrace.PropagateWithCode(err, ErrCodeNotFound, "send schedule [%s] not found for user with ID [%s]", scheduleID, userID),
		)
	}
	if err != nil {
		return nil, r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot load send schedule [%s]", scheduleID))
	}

	return item, nil
}

// Index lists all message send schedules owned by the given user.
func (r *gormMessageSendScheduleRepository) Index(
	ctx context.Context,
	userID entities.UserID,
) ([]entities.MessageSendSchedule, error) {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()

	items := make([]entities.MessageSendSchedule, 0)
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot index send schedules for user [%s]", userID))
	}

	return items, nil
}

// Delete removes a message send schedule owned by the given user.
func (r *gormMessageSendScheduleRepository) Delete(
	ctx context.Context,
	userID entities.UserID,
	scheduleID uuid.UUID,
) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", scheduleID).
		Delete(&entities.MessageSendSchedule{}).Error
	if err != nil {
		return r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete send schedule [%s]", scheduleID))
	}

	return nil
}

// DeleteAllForUser removes all message send schedules owned by the given user.
func (r *gormMessageSendScheduleRepository) DeleteAllForUser(
	ctx context.Context,
	userID entities.UserID,
) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&entities.MessageSendSchedule{}).Error
	if err != nil {
		return r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete send schedules for user [%s]", userID))
	}

	return nil
}

// CountByUser returns the number of schedules owned by a user.
func (r *gormMessageSendScheduleRepository) CountByUser(
	ctx context.Context,
	userID entities.UserID,
) (int, error) {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()

	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.MessageSendSchedule{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot count send schedules for user [%s]", userID))
	}

	return int(count), nil
}
