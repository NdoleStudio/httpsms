package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// MessageSendScheduleService manages message send schedules for a user.
type MessageSendScheduleService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.MessageSendScheduleRepository
	dispatcher *EventDispatcher
}

// NewMessageSendScheduleService creates a new MessageSendScheduleService.
func NewMessageSendScheduleService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.MessageSendScheduleRepository,
	dispatcher *EventDispatcher,
) *MessageSendScheduleService {
	return &MessageSendScheduleService{
		logger:     logger.WithService(fmt.Sprintf("%T", &MessageSendScheduleService{})),
		tracer:     tracer,
		repository: repository,
		dispatcher: dispatcher,
	}
}

// MessageSendScheduleUpsertParams contains the fields required to create or update a message send schedule.
type MessageSendScheduleUpsertParams struct {
	UserID   entities.UserID
	Name     string
	Timezone string
	Windows  []entities.MessageSendScheduleWindow
}

// Index returns all message send schedules for a user.
func (service *MessageSendScheduleService) Index(
	ctx context.Context,
	userID entities.UserID,
) ([]entities.MessageSendSchedule, error) {
	return service.repository.Index(ctx, userID)
}

// CountByUser returns the number of schedules owned by a user.
func (service *MessageSendScheduleService) CountByUser(
	ctx context.Context,
	userID entities.UserID,
) (int, error) {
	return service.repository.CountByUser(ctx, userID)
}

// Load returns a single message send schedule for a user.
func (service *MessageSendScheduleService) Load(
	ctx context.Context,
	userID entities.UserID,
	scheduleID uuid.UUID,
) (*entities.MessageSendSchedule, error) {
	return service.repository.Load(ctx, userID, scheduleID)
}

// Store creates a new message send schedule.
func (service *MessageSendScheduleService) Store(
	ctx context.Context,
	params *MessageSendScheduleUpsertParams,
) (*entities.MessageSendSchedule, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	schedule := &entities.MessageSendSchedule{
		ID:        uuid.New(),
		UserID:    params.UserID,
		Name:      params.Name,
		Timezone:  params.Timezone,
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
func (service *MessageSendScheduleService) Update(
	ctx context.Context,
	userID entities.UserID,
	scheduleID uuid.UUID,
	params *MessageSendScheduleUpsertParams,
) (*entities.MessageSendSchedule, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	schedule, err := service.repository.Load(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}

	schedule.Name = params.Name
	schedule.Timezone = params.Timezone
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
func (service *MessageSendScheduleService) Delete(
	ctx context.Context,
	userID entities.UserID,
	scheduleID uuid.UUID,
) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if err := service.repository.Delete(ctx, userID, scheduleID); err != nil {
		msg := fmt.Sprintf("cannot delete message send schedule with ID [%s] for user [%s]", scheduleID, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	event, err := service.createEvent(events.EventTypeMessageSendScheduleDeleted, fmt.Sprintf("%T", service), events.MessageSendScheduleDeletedPayload{
		ScheduleID: scheduleID,
		UserID:     userID,
		Timestamp:  time.Now().UTC(),
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create [%s] event for schedule [%s]", events.EventTypeMessageSendScheduleDeleted, scheduleID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for schedule [%s]", event.Type(), scheduleID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
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
func (service *MessageSendScheduleService) DeleteAllForUser(
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
