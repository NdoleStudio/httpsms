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

type SendScheduleService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.SendScheduleRepository
}

func NewSendScheduleService(logger telemetry.Logger, tracer telemetry.Tracer, repository repositories.SendScheduleRepository) *SendScheduleService {
	return &SendScheduleService{logger: logger.WithService(fmt.Sprintf("%T", &SendScheduleService{})), tracer: tracer, repository: repository}
}

type SendScheduleUpsertParams struct {
	UserID   entities.UserID
	Name     string
	Timezone string
	IsActive bool
	Windows  []entities.SendScheduleWindow
}

func (service *SendScheduleService) Index(ctx context.Context, userID entities.UserID) ([]entities.SendSchedule, error) {
	return service.repository.Index(ctx, userID)
}

func (service *SendScheduleService) Load(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) (*entities.SendSchedule, error) {
	return service.repository.Load(ctx, userID, scheduleID)
}

func (service *SendScheduleService) Store(ctx context.Context, params *SendScheduleUpsertParams) (*entities.SendSchedule, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()
	schedule := &entities.SendSchedule{ID: uuid.New(), UserID: params.UserID, Name: params.Name, Timezone: params.Timezone, IsActive: params.IsActive, Windows: sanitizeWindows(params.Windows), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := service.repository.Store(ctx, schedule); err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot store send schedule [%s]", schedule.ID)))
	}
	return schedule, nil
}

func (service *SendScheduleService) Update(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID, params *SendScheduleUpsertParams) (*entities.SendSchedule, error) {
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
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot update send schedule [%s]", schedule.ID)))
	}
	return schedule, nil
}

func (service *SendScheduleService) Delete(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) error {
	return service.repository.Delete(ctx, userID, scheduleID)
}

func (service *SendScheduleService) ResolveScheduledSendTime(ctx context.Context, schedule *entities.SendSchedule, current time.Time) (time.Time, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()
	if schedule == nil || !schedule.IsActive || len(schedule.Windows) == 0 {
		return current.UTC(), nil
	}
	location, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return current.UTC(), service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot load location [%s]", schedule.Timezone)))
	}

	base := current.In(location)
	type candidate struct{ t time.Time }
	best := time.Time{}
	for dayOffset := 0; dayOffset <= 7; dayOffset++ {
		day := base.AddDate(0, 0, dayOffset)
		weekday := int(day.Weekday())
		for _, window := range schedule.Windows {
			if window.DayOfWeek != weekday {
				continue
			}
			start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, location).Add(time.Duration(window.StartMinute) * time.Minute)
			end := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, location).Add(time.Duration(window.EndMinute) * time.Minute)
			var candidateTime time.Time
			switch {
			case dayOffset == 0 && base.Before(start):
				candidateTime = start
			case (dayOffset == 0 && (base.Equal(start) || (base.After(start) && base.Before(end)))) || (dayOffset > 0):
				candidateTime = base
				if dayOffset > 0 {
					candidateTime = start
				}
			default:
				continue
			}
			if best.IsZero() || candidateTime.Before(best) {
				best = candidateTime
			}
		}
		if !best.IsZero() {
			break
		}
	}
	if best.IsZero() {
		return current.UTC(), nil
	}
	return best.UTC(), nil
}

func sanitizeWindows(windows []entities.SendScheduleWindow) []entities.SendScheduleWindow {
	result := make([]entities.SendScheduleWindow, 0, len(windows))
	for _, item := range windows {
		result = append(result, entities.SendScheduleWindow{DayOfWeek: item.DayOfWeek, StartMinute: item.StartMinute, EndMinute: item.EndMinute})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].DayOfWeek == result[j].DayOfWeek {
			return result[i].StartMinute < result[j].StartMinute
		}
		return result[i].DayOfWeek < result[j].DayOfWeek
	})
	return result
}
