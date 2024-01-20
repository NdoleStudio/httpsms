package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageSendRetry is emitted when the phone a message expires and is being retried
const EventTypeMessageSendRetry = "message.send.retry"

// MessageSendRetryPayload is the payload of the EventTypeMessageSendRetry event
type MessageSendRetryPayload struct {
	MessageID uuid.UUID       `json:"message_id"`
	Owner     string          `json:"owner"`
	Contact   string          `json:"contact"`
	Encrypted bool            `json:"encrypted"`
	UserID    entities.UserID `json:"user_id"`
	Timestamp time.Time       `json:"timestamp"`
	Content   string          `json:"content"`
	SIM       entities.SIM    `json:"sim"`
}
