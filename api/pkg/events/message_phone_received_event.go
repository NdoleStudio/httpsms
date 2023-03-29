package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessagePhoneReceived is emitted when a new message is received by a mobile phone
const EventTypeMessagePhoneReceived = "message.phone.received"

// MessagePhoneReceivedPayload is the payload of the EventTypeMessagePhoneReceived event
type MessagePhoneReceivedPayload struct {
	MessageID uuid.UUID       `json:"message_id"`
	UserID    entities.UserID `json:"user_id"`
	Owner     string          `json:"owner"`
	Contact   string          `json:"contact"`
	Timestamp time.Time       `json:"timestamp"`
	Content   string          `json:"content"`
	SIM       entities.SIM    `json:"sim"`
}
