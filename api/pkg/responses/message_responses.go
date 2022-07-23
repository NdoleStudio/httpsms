package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// MessageResponse is the payload containing an entities.Message
type MessageResponse struct {
	response
	Data entities.Message `json:"data"`
}

// MessagesResponse is the payload containing []entities.Message
type MessagesResponse struct {
	response
	Data []entities.Message `json:"data"`
}
