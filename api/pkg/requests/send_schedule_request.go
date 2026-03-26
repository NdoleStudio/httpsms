package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

type SendScheduleWindow struct {
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type SendScheduleUpsert struct {
	request
	Name      string               `json:"name"`
	Timezone  string               `json:"timezone"`
	IsDefault bool                 `json:"is_default"`
	IsActive  bool                 `json:"is_active"`
	Windows   []SendScheduleWindow `json:"windows"`
}

func (input *SendScheduleUpsert) Sanitize() SendScheduleUpsert {
	input.Name = strings.TrimSpace(input.Name)
	input.Timezone = strings.TrimSpace(input.Timezone)
	for i := range input.Windows {
		input.Windows[i].StartTime = strings.TrimSpace(input.Windows[i].StartTime)
		input.Windows[i].EndTime = strings.TrimSpace(input.Windows[i].EndTime)
	}
	return *input
}

func (input *SendScheduleUpsert) ToUpsertParams() services.SendScheduleUpsertParams {
	windows := make([]services.SendScheduleWindowParams, 0, len(input.Windows))
	for _, item := range input.Windows {
		windows = append(windows, services.SendScheduleWindowParams{DayOfWeek: item.DayOfWeek, StartTime: item.StartTime, EndTime: item.EndTime})
	}
	return services.SendScheduleUpsertParams{Name: input.Name, Timezone: input.Timezone, IsDefault: input.IsDefault, IsActive: input.IsActive, Windows: windows}
}
