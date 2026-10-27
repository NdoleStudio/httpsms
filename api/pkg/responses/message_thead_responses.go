package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// MessageThreadsResponse is the payload containing []entities.MessageThread
type MessageThreadsResponse struct {
	response
	Data []entities.MessageThread `json:"data"`
	// HasMore indicates that additional threads exist after this page.
	HasMore bool `json:"has_more" example:"true"`
}

// MessageThreadResponse is the payload containing entities.MessageThread
type MessageThreadResponse struct {
	response
	Data entities.MessageThread `json:"data"`
}
