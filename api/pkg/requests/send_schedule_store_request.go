package requests

import (
	"sort"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

type SendScheduleWindow struct {
	DayOfWeek   int `json:"day_of_week"`
	StartMinute int `json:"start_minute"`
	EndMinute   int `json:"end_minute"`
}

type SendScheduleStore struct {
	request
	Name     string               `json:"name"`
	Timezone string               `json:"timezone"`
	IsActive bool                 `json:"is_active"`
	Windows  []SendScheduleWindow `json:"windows"`
}

func (input *SendScheduleStore) Sanitize() SendScheduleStore {
	input.Name = strings.TrimSpace(input.Name)
	input.Timezone = strings.TrimSpace(input.Timezone)
	windows := make([]SendScheduleWindow, 0, len(input.Windows))
	for _, item := range input.Windows {
		windows = append(windows, SendScheduleWindow{DayOfWeek: item.DayOfWeek, StartMinute: item.StartMinute, EndMinute: item.EndMinute})
	}
	sort.SliceStable(windows, func(i, j int) bool {
		if windows[i].DayOfWeek == windows[j].DayOfWeek {
			return windows[i].StartMinute < windows[j].StartMinute
		}
		return windows[i].DayOfWeek < windows[j].DayOfWeek
	})
	input.Windows = windows
	return *input
}

func (input *SendScheduleStore) ToParams(user entities.AuthContext) *services.SendScheduleUpsertParams {
	windows := make([]entities.SendScheduleWindow, 0, len(input.Windows))
	for _, item := range input.Windows {
		windows = append(windows, entities.SendScheduleWindow{DayOfWeek: item.DayOfWeek, StartMinute: item.StartMinute, EndMinute: item.EndMinute})
	}
	return &services.SendScheduleUpsertParams{UserID: user.ID, Name: input.Name, Timezone: input.Timezone, IsActive: input.IsActive, Windows: windows}
}
