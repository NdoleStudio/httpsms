package requests

import (
	"sort"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageSendScheduleWindow represents a single request window for a message send schedule.
type MessageSendScheduleWindow struct {
	DayOfWeek   int `json:"day_of_week"`
	StartMinute int `json:"start_minute"`
	EndMinute   int `json:"end_minute"`
}

// MessageSendScheduleStore contains the payload used to create or update a message send schedule.
type MessageSendScheduleStore struct {
	request
	Name     string                      `json:"name"`
	Timezone string                      `json:"timezone"`
	Windows  []MessageSendScheduleWindow `json:"windows"`
}

// Sanitize trims and sorts the message send schedule payload before validation.
func (input *MessageSendScheduleStore) Sanitize() MessageSendScheduleStore {
	input.Name = strings.TrimSpace(input.Name)
	input.Timezone = strings.TrimSpace(input.Timezone)
	windows := make([]MessageSendScheduleWindow, 0, len(input.Windows))
	for _, item := range input.Windows {
		windows = append(windows, MessageSendScheduleWindow{DayOfWeek: item.DayOfWeek, StartMinute: item.StartMinute, EndMinute: item.EndMinute})
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

// ToParams converts the request payload into message send schedule service params.
func (input *MessageSendScheduleStore) ToParams(user entities.AuthContext) *services.MessageSendScheduleUpsertParams {
	windows := make([]entities.MessageSendScheduleWindow, 0, len(input.Windows))
	for _, item := range input.Windows {
		windows = append(windows, entities.MessageSendScheduleWindow{DayOfWeek: item.DayOfWeek, StartMinute: item.StartMinute, EndMinute: item.EndMinute})
	}
	return &services.MessageSendScheduleUpsertParams{UserID: user.ID, Name: input.Name, Timezone: input.Timezone, Windows: windows}
}
