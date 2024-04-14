package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// MessageCallMissed is emitted when a new message is sent
const MessageCallMissed = "message.call.missed"

// MessageCallMissedPayload is the payload of the MessageCallMissed event
type MessageCallMissedPayload struct {
	MessageID uuid.UUID       `json:"message_id"`
	UserID    entities.UserID `json:"user_id"`
	Owner     string          `json:"owner"`
	Contact   string          `json:"contact"`
	Timestamp time.Time       `json:"timestamp"`
	SIM       entities.SIM    `json:"sim"`
}
