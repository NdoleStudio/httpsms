package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// MessageSendSchedulesResponse represents a collection of message send schedules.
type MessageSendSchedulesResponse struct {
	response
	Data []entities.MessageSendSchedule `json:"data"`
}

// MessageSendScheduleResponse represents a single message send schedule.
type MessageSendScheduleResponse struct {
	response
	Data entities.MessageSendSchedule `json:"data"`
}
