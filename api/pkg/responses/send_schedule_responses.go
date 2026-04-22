package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

type SendSchedulesResponse struct {
	response
	Data []entities.MessageSendSchedule `json:"data"`
}

type SendScheduleResponse struct {
	response
	Data entities.MessageSendSchedule `json:"data"`
}
