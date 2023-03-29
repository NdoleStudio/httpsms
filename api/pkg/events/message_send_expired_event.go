package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageSendExpired is emitted when the phone a message expires
const EventTypeMessageSendExpired = "message.send.expired"

// MessageSendExpiredPayload is the payload of the EventTypeMessageSendExpired event
type MessageSendExpiredPayload struct {
	MessageID uuid.UUID       `json:"message_id"`
	Owner     string          `json:"owner"`
	Contact   string          `json:"contact"`
	UserID    entities.UserID `json:"user_id"`
	Timestamp time.Time       `json:"timestamp"`
	Content   string          `json:"content"`
	SIM       entities.SIM    `json:"sim"`
}
