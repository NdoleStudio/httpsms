package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// MessageThreadAPIDeleted is emitted when a new message is deleted
const MessageThreadAPIDeleted = "message-thread.api.deleted"

// MessageThreadAPIDeletedPayload is the payload of the MessageThreadAPIDeleted event
type MessageThreadAPIDeletedPayload struct {
	MessageThreadID uuid.UUID              `json:"message_thread_id"`
	UserID          entities.UserID        `json:"user_id"`
	Owner           string                 `json:"owner"`
	Contact         string                 `json:"contact"`
	IsArchived      bool                   `json:"is_archived"`
	Color           string                 `json:"color"`
	Status          entities.MessageStatus `json:"status"`
	Timestamp       time.Time              `json:"timestamp"`
}
