package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypeMessageSendFailed is emitted when the phone could not send
const EventTypeMessageSendFailed = "message.send.failed"

// MessageSendFailedPayload is the payload of the EventTypeMessageSendFailed event
type MessageSendFailedPayload struct {
	ID           uuid.UUID       `json:"id"`
	ErrorMessage string          `json:"error_message"`
	UserID       entities.UserID `json:"user_id"`
	Owner        string          `json:"owner"`
	RequestID    *string         `json:"request_id"`
	Contact      string          `json:"contact"`
	Timestamp    time.Time       `json:"timestamp"`
	Content      string          `json:"content"`
	SIM          entities.SIM    `json:"sim"`
}
