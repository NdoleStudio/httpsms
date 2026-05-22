package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// BulkMessagesResponse is the payload containing []*entities.BulkMessage
type BulkMessagesResponse struct {
	response
	Data []*entities.BulkMessage `json:"data"`
}
