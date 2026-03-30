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

type gormSendScheduleRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

func NewGormSendScheduleRepository(logger telemetry.Logger, tracer telemetry.Tracer, db *gorm.DB) SendScheduleRepository {
	return &gormSendScheduleRepository{logger: logger.WithService(fmt.Sprintf("%T", &gormSendScheduleRepository{})), tracer: tracer, db: db}
}

func (r *gormSendScheduleRepository) Store(ctx context.Context, schedule *entities.SendSchedule) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()
	if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
		return r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot store send schedule [%s]", schedule.ID)))
	}
	return nil
}

func (r *gormSendScheduleRepository) Update(ctx context.Context, schedule *entities.SendSchedule) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()
	if err := r.db.WithContext(ctx).Save(schedule).Error; err != nil {
		return r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot update send schedule [%s]", schedule.ID)))
	}
	return nil
}

func (r *gormSendScheduleRepository) Load(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) (*entities.SendSchedule, error) {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()
	item := new(entities.SendSchedule)
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", scheduleID).First(item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, r.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, fmt.Sprintf("send schedule [%s] not found", scheduleID)))
	}
	if err != nil {
		return nil, r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot load send schedule [%s]", scheduleID)))
	}
	return item, nil
}

func (r *gormSendScheduleRepository) Index(ctx context.Context, userID entities.UserID) ([]entities.SendSchedule, error) {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()
	items := make([]entities.SendSchedule, 0)
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot index send schedules for user [%s]", userID)))
	}
	return items, nil
}

func (r *gormSendScheduleRepository) Delete(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", scheduleID).Delete(&entities.SendSchedule{}).Error; err != nil {
		return r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot delete send schedule [%s]", scheduleID)))
	}
	return nil
}

func (r *gormSendScheduleRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := r.tracer.Start(ctx)
	defer span.End()
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.SendSchedule{}).Error; err != nil {
		return r.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot delete send schedules for user [%s]", userID)))
	}
	return nil
}
