package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// MessageThreadsResponse is the payload containing []entities.MessageThread
type MessageThreadsResponse struct {
	response
	Data []entities.MessageThread `json:"data"`
}
