package responses

import "github.com/NdoleStudio/http-sms-manager/pkg/entities"

// MessageThreadsResponse is the payload containing []entities.MessageThread
type MessageThreadsResponse struct {
	response
	Data []entities.MessageThread `json:"data"`
}
