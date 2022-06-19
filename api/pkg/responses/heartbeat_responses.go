package responses

import "github.com/NdoleStudio/http-sms-manager/pkg/entities"

// HeartbeatsResponse is the payload containing []entities.Heartbeat
type HeartbeatsResponse struct {
	response
	Data []entities.Heartbeat `json:"data"`
}
