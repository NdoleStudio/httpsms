package requests

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageThreadUpdate is the payload for updating a message thread
type MessageThreadUpdate struct {
	request
	IsArchived  *bool `json:"is_archived,omitempty" example:"true"`
	UnreadCount *uint `json:"unread_count,omitempty" example:"0" minimum:"0" maximum:"0"`

	MessageThreadID string `json:"messageThreadID" swaggerignore:"true"` // used internally for validation
}

// ToUpdateParams converts MessageThreadUpdate to services.MessageThreadStatusParams
func (input *MessageThreadUpdate) ToUpdateParams(userID entities.UserID) services.MessageThreadStatusParams {
	return services.MessageThreadStatusParams{
		UserID:          userID,
		MessageThreadID: uuid.MustParse(input.MessageThreadID),
		IsArchived:      input.IsArchived,
		UnreadCount:     input.UnreadCount,
	}
}
