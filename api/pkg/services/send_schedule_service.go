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

type SendScheduleWindowParams struct {
	DayOfWeek int
	StartTime string
	EndTime   string
}

type SendScheduleUpsertParams struct {
	Name      string
	Timezone  string
	IsDefault bool
	IsActive  bool
	Windows   []SendScheduleWindowParams
}

type SendScheduleService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.SendScheduleRepository
}

func NewSendScheduleService(logger telemetry.Logger, tracer telemetry.Tracer, repository repositories.SendScheduleRepository) *SendScheduleService {
	return &SendScheduleService{logger: logger.WithService(fmt.Sprintf("%T", &SendScheduleService{})), tracer: tracer, repository: repository}
}

func (s *SendScheduleService) Index(ctx context.Context, userID entities.UserID) ([]*entities.SendSchedule, error) {
	return s.repository.Index(ctx, userID)
}
func (s *SendScheduleService) Load(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) (*entities.SendSchedule, error) {
	return s.repository.Load(ctx, userID, scheduleID)
}
func (s *SendScheduleService) Delete(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) error {
	return s.repository.Delete(ctx, userID, scheduleID)
}

func (s *SendScheduleService) Create(ctx context.Context, userID entities.UserID, params SendScheduleUpsertParams) (*entities.SendSchedule, error) {
	if _, err := time.LoadLocation(params.Timezone); err != nil {
		return nil, stacktrace.Propagate(err, "invalid schedule timezone")
	}
	windows, err := s.buildWindows(uuid.New(), params.Windows)
	if err != nil {
		return nil, err
	}
	schedule := &entities.SendSchedule{ID: uuid.New(), UserID: userID, Name: params.Name, Timezone: params.Timezone, IsDefault: params.IsDefault, IsActive: params.IsActive, Windows: windows}
	if schedule.IsDefault {
		if err := s.repository.ClearDefault(ctx, userID); err != nil {
			return nil, err
		}
	}
	for i := range schedule.Windows {
		schedule.Windows[i].ScheduleID = schedule.ID
	}
	if err := s.repository.Store(ctx, schedule); err != nil {
		return nil, err
	}
	return s.repository.Load(ctx, userID, schedule.ID)
}

func (s *SendScheduleService) Update(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID, params SendScheduleUpsertParams) (*entities.SendSchedule, error) {
	if _, err := time.LoadLocation(params.Timezone); err != nil {
		return nil, stacktrace.Propagate(err, "invalid schedule timezone")
	}
	current, err := s.repository.Load(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	windows, err := s.buildWindows(scheduleID, params.Windows)
	if err != nil {
		return nil, err
	}
	current.Name = params.Name
	current.Timezone = params.Timezone
	current.IsDefault = params.IsDefault
	current.IsActive = params.IsActive
	current.Windows = windows
	if current.IsDefault {
		if err := s.repository.ClearDefault(ctx, userID); err != nil {
			return nil, err
		}
		current.IsDefault = true
	}
	if err := s.repository.Update(ctx, current); err != nil {
		return nil, err
	}
	return s.repository.Load(ctx, userID, scheduleID)
}

func (s *SendScheduleService) SetDefault(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) (*entities.SendSchedule, error) {
	item, err := s.repository.Load(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	if err := s.repository.ClearDefault(ctx, userID); err != nil {
		return nil, err
	}
	item.IsDefault = true
	if !item.IsActive {
		item.IsActive = true
	}
	if err := s.repository.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.repository.Load(ctx, userID, scheduleID)
}

func (s *SendScheduleService) ResolveScheduledSendTime(ctx context.Context, userID entities.UserID, sendAt *time.Time) (*time.Time, error) {
	schedule, err := s.repository.Default(ctx, userID)
	if err != nil {
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			return sendAt, nil
		}
		return nil, err
	}
	if !schedule.IsActive || len(schedule.Windows) == 0 {
		return sendAt, nil
	}
	if sendAt != nil {
		if !s.IsAllowedAt(schedule, *sendAt) {
			return nil, stacktrace.NewError("send_at falls outside the default send schedule")
		}
		return sendAt, nil
	}
	now := time.Now().UTC()
	if s.IsAllowedAt(schedule, now) {
		return nil, nil
	}
	next, err := s.NextAllowedTime(schedule, now)
	if err != nil {
		return nil, err
	}
	return &next, nil
}

func (s *SendScheduleService) IsAllowedAt(schedule *entities.SendSchedule, ts time.Time) bool {
	if schedule == nil || !schedule.IsActive || len(schedule.Windows) == 0 {
		return true
	}
	location, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		location = time.UTC
	}
	local := ts.In(location)
	minute := local.Hour()*60 + local.Minute()
	weekday := int(local.Weekday())
	for _, window := range schedule.Windows {
		if window.DayOfWeek == weekday && minute >= window.StartMinute && minute < window.EndMinute {
			return true
		}
	}
	return false
}

func (s *SendScheduleService) NextAllowedTime(schedule *entities.SendSchedule, ts time.Time) (time.Time, error) {
	location, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return time.Time{}, stacktrace.Propagate(err, "invalid timezone")
	}
	local := ts.In(location)
	for dayOffset := 0; dayOffset <= 7; dayOffset++ {
		day := local.AddDate(0, 0, dayOffset)
		weekday := int(day.Weekday())
		minuteNow := day.Hour()*60 + day.Minute()
		windows := make([]entities.SendScheduleWindow, 0)
		for _, w := range schedule.Windows {
			if w.DayOfWeek == weekday {
				windows = append(windows, w)
			}
		}
		sort.Slice(windows, func(i, j int) bool { return windows[i].StartMinute < windows[j].StartMinute })
		for _, window := range windows {
			if dayOffset == 0 && minuteNow < window.StartMinute {
				t := time.Date(day.Year(), day.Month(), day.Day(), window.StartMinute/60, window.StartMinute%60, 0, 0, location)
				return t.UTC(), nil
			}
			if dayOffset > 0 {
				t := time.Date(day.Year(), day.Month(), day.Day(), window.StartMinute/60, window.StartMinute%60, 0, 0, location)
				return t.UTC(), nil
			}
		}
	}
	return time.Time{}, stacktrace.NewError("no future availability found")
}

func (s *SendScheduleService) buildWindows(scheduleID uuid.UUID, params []SendScheduleWindowParams) ([]entities.SendScheduleWindow, error) {
	result := make([]entities.SendScheduleWindow, 0, len(params))
	perDay := map[int][]entities.SendScheduleWindow{}
	for _, item := range params {
		start, err := parseClock(item.StartTime)
		if err != nil {
			return nil, err
		}
		end, err := parseClock(item.EndTime)
		if err != nil {
			return nil, err
		}
		if item.DayOfWeek < 0 || item.DayOfWeek > 6 {
			return nil, stacktrace.NewError("day_of_week must be between 0 and 6")
		}
		if end <= start {
			return nil, stacktrace.NewError("end_time must be after start_time")
		}
		window := entities.SendScheduleWindow{ID: uuid.New(), ScheduleID: scheduleID, DayOfWeek: item.DayOfWeek, StartMinute: start, EndMinute: end}
		perDay[item.DayOfWeek] = append(perDay[item.DayOfWeek], window)
		result = append(result, window)
	}
	for _, windows := range perDay {
		sort.Slice(windows, func(i, j int) bool { return windows[i].StartMinute < windows[j].StartMinute })
		for i := 1; i < len(windows); i++ {
			if windows[i].StartMinute < windows[i-1].EndMinute {
				return nil, stacktrace.NewError("schedule windows cannot overlap")
			}
		}
	}
	return result, nil
}

func parseClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, stacktrace.Propagate(err, fmt.Sprintf("invalid time [%s]", value))
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}
