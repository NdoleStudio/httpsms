package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// MessageAPIDeleted is emitted when a new message is deleted
const MessageAPIDeleted = "message.api.deleted"

// MessageAPIDeletedPayload is the payload of the MessageAPIDeleted event
type MessageAPIDeletedPayload struct {
	MessageID              uuid.UUID               `json:"message_id"`
	UserID                 entities.UserID         `json:"user_id"`
	Owner                  string                  `json:"owner"`
	RequestID              *string                 `json:"request_id"`
	Contact                string                  `json:"contact"`
	Timestamp              time.Time               `json:"timestamp"`
	Content                string                  `json:"content"`
	Encrypted              bool                    `json:"encrypted"`
	PreviousMessageID      *uuid.UUID              `json:"previous_message_id"`
	PreviousMessageStatus  *entities.MessageStatus `json:"previous_message_status"`
	PreviousMessageContent *string                 `json:"previous_message_content"`
	SIM                    entities.SIM            `json:"sim"`
}
