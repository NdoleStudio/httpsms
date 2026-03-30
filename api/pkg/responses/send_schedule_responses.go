package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

type SendSchedulesResponse struct {
	response
	Data []entities.SendSchedule `json:"data"`
}

type SendScheduleResponse struct {
	response
	Data entities.SendSchedule `json:"data"`
}
