package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// SendScheduleService manages message send schedules for a user.
type SendScheduleService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.SendScheduleRepository
}

// NewSendScheduleService creates a new SendScheduleService.
func NewSendScheduleService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.SendScheduleRepository,
) *SendScheduleService {
	return &SendScheduleService{
		logger:     logger.WithService(fmt.Sprintf("%T", &SendScheduleService{})),
		tracer:     tracer,
		repository: repository,
	}
}

// SendScheduleUpsertParams contains the fields required to create or update a message send schedule.
type SendScheduleUpsertParams struct {
	UserID   entities.UserID
	Name     string
	Timezone string
	IsActive bool
	Windows  []entities.MessageSendScheduleWindow
}

// Index returns all message send schedules for a user.
func (service *SendScheduleService) Index(
	ctx context.Context,
	userID entities.UserID,
) ([]entities.MessageSendSchedule, error) {
	return service.repository.Index(ctx, userID)
}

// CountByUser returns the number of schedules owned by a user.
func (service *SendScheduleService) CountByUser(
	ctx context.Context,
	userID entities.UserID,
) (int, error) {
	return service.repository.CountByUser(ctx, userID)
}

// Load returns a single message send schedule for a user.
func (service *SendScheduleService) Load(
	ctx context.Context,
	userID entities.UserID,
	scheduleID uuid.UUID,
) (*entities.MessageSendSchedule, error) {
	return service.repository.Load(ctx, userID, scheduleID)
}

// Store creates a new message send schedule.
func (service *SendScheduleService) Store(
	ctx context.Context,
	params *SendScheduleUpsertParams,
) (*entities.MessageSendSchedule, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	schedule := &entities.MessageSendSchedule{
		ID:        uuid.New(),
		UserID:    params.UserID,
		Name:      params.Name,
		Timezone:  params.Timezone,
		IsActive:  params.IsActive,
		Windows:   sanitizeWindows(params.Windows),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := service.repository.Store(ctx, schedule); err != nil {
		return nil, service.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				fmt.Sprintf("cannot store message send schedule [%s]", schedule.ID),
			),
		)
	}

	return schedule, nil
}

// Update updates an existing message send schedule.
func (service *SendScheduleService) Update(
	ctx context.Context,
	userID entities.UserID,
	scheduleID uuid.UUID,
	params *SendScheduleUpsertParams,
) (*entities.MessageSendSchedule, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	schedule, err := service.repository.Load(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}

	schedule.Name = params.Name
	schedule.Timezone = params.Timezone
	schedule.IsActive = params.IsActive
	schedule.Windows = sanitizeWindows(params.Windows)
	schedule.UpdatedAt = time.Now().UTC()

	if err = service.repository.Update(ctx, schedule); err != nil {
		return nil, service.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				fmt.Sprintf("cannot update message send schedule [%s]", schedule.ID),
			),
		)
	}

	return schedule, nil
}

// Delete removes a message send schedule for a user.
func (service *SendScheduleService) Delete(
	ctx context.Context,
	userID entities.UserID,
	scheduleID uuid.UUID,
) error {
	return service.repository.Delete(ctx, userID, scheduleID)
}

// sanitizeWindows normalizes and sorts schedule windows by day and start minute.
func sanitizeWindows(
	windows []entities.MessageSendScheduleWindow,
) []entities.MessageSendScheduleWindow {
	result := make([]entities.MessageSendScheduleWindow, 0, len(windows))

	for _, item := range windows {
		result = append(result, entities.MessageSendScheduleWindow{
			DayOfWeek:   item.DayOfWeek,
			StartMinute: item.StartMinute,
			EndMinute:   item.EndMinute,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].DayOfWeek == result[j].DayOfWeek {
			return result[i].StartMinute < result[j].StartMinute
		}
		return result[i].DayOfWeek < result[j].DayOfWeek
	})

	return result
}

// DeleteAllForUser removes all message send schedules owned by a user.
func (service *SendScheduleService) DeleteAllForUser(
	ctx context.Context,
	userID entities.UserID,
) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if err := service.repository.DeleteAllForUser(ctx, userID); err != nil {
		return service.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				fmt.Sprintf("cannot delete message send schedules for user [%s]", userID),
			),
		)
	}

	return nil
}
