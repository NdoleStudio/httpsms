package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// HeartbeatsResponse is the payload containing []entities.Heartbeat
type HeartbeatsResponse struct {
	response
	Data []entities.Heartbeat `json:"data"`
}

// HeartbeatResponse is the payload containing entities.Heartbeat
type HeartbeatResponse struct {
	response
	Data entities.Heartbeat `json:"data"`
}
