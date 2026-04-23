package validators

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

const maxWindowsPerDay = 6

// SendScheduleHandlerValidator validates send schedule HTTP requests.
type SendScheduleHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewSendScheduleHandlerValidator creates a new SendScheduleHandlerValidator.
func NewSendScheduleHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) *SendScheduleHandlerValidator {
	return &SendScheduleHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", &SendScheduleHandlerValidator{})),
		tracer: tracer,
	}
}

// ValidateStore validates a send schedule create or update request.
func (validator *SendScheduleHandlerValidator) ValidateStore(
	_ context.Context,
	request requests.SendScheduleStore,
) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"name":     []string{"required", "min:2", "max:100"},
			"timezone": []string{"required", "min:2", "max:100"},
		},
	})

	result := v.ValidateStruct()
	validator.validateWindows(result, request.Windows)

	if request.Timezone != "" {
		if _, err := time.LoadLocation(request.Timezone); err != nil {
			result.Add("timezone", "timezone must be a valid IANA timezone")
		}
	}

	return result
}

func (validator *SendScheduleHandlerValidator) validateWindows(
	result url.Values,
	windows []requests.SendScheduleWindow,
) {
	windowsPerDay := make(map[int]int)

	for index, item := range windows {
		validator.validateDayOfWeek(result, index, item, windowsPerDay)
		validator.validateStartMinute(result, index, item)
		validator.validateEndMinute(result, index, item)
		validator.validateWindowRange(result, index, item)
	}

	validator.validateOverlappingWindows(result, windows)
}

func (validator *SendScheduleHandlerValidator) validateDayOfWeek(
	result url.Values,
	index int,
	item requests.SendScheduleWindow,
	windowsPerDay map[int]int,
) {
	if item.DayOfWeek < 0 || item.DayOfWeek > 6 {
		result.Add("windows", fmt.Sprintf("windows[%d].day_of_week must be between 0 and 6", index))
		return
	}

	windowsPerDay[item.DayOfWeek]++
	if windowsPerDay[item.DayOfWeek] > maxWindowsPerDay {
		result.Add(
			"windows",
			fmt.Sprintf("day_of_week %d cannot have more than %d windows", item.DayOfWeek, maxWindowsPerDay),
		)
	}
}

func (validator *SendScheduleHandlerValidator) validateStartMinute(
	result url.Values,
	index int,
	item requests.SendScheduleWindow,
) {
	if item.StartMinute < 0 || item.StartMinute > 1439 {
		result.Add("windows", fmt.Sprintf("windows[%d].start_minute must be between 0 and 1439", index))
	}
}

func (validator *SendScheduleHandlerValidator) validateEndMinute(
	result url.Values,
	index int,
	item requests.SendScheduleWindow,
) {
	if item.EndMinute < 1 || item.EndMinute > 1440 {
		result.Add("windows", fmt.Sprintf("windows[%d].end_minute must be between 1 and 1440", index))
	}
}

func (validator *SendScheduleHandlerValidator) validateWindowRange(
	result url.Values,
	index int,
	item requests.SendScheduleWindow,
) {
	if item.EndMinute <= item.StartMinute {
		result.Add("windows", fmt.Sprintf("windows[%d].end_minute must be greater than start_minute", index))
	}
}

func (validator *SendScheduleHandlerValidator) validateOverlappingWindows(
	result url.Values,
	windows []requests.SendScheduleWindow,
) {
	grouped := make(map[int][]requests.SendScheduleWindow)

	for _, item := range windows {
		if item.DayOfWeek < 0 || item.DayOfWeek > 6 {
			continue
		}
		if item.EndMinute <= item.StartMinute {
			continue
		}
		grouped[item.DayOfWeek] = append(grouped[item.DayOfWeek], item)
	}

	for dayOfWeek, dayWindows := range grouped {
		sort.Slice(dayWindows, func(i, j int) bool {
			return dayWindows[i].StartMinute < dayWindows[j].StartMinute
		})

		for i := 1; i < len(dayWindows); i++ {
			previous := dayWindows[i-1]
			current := dayWindows[i]

			if current.StartMinute < previous.EndMinute {
				result.Add(
					"windows",
					fmt.Sprintf("day_of_week %d contains overlapping windows", dayOfWeek),
				)
				break
			}
		}
	}
}
