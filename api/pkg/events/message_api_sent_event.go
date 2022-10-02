package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageAPISent is emitted when a new message is sent
const EventTypeMessageAPISent = "message.api.sent"

// MessageAPISentPayload is the payload of the EventTypeMessageSent event
type MessageAPISentPayload struct {
	MessageID         uuid.UUID       `json:"message_id"`
	UserID            entities.UserID `json:"user_id"`
	Owner             string          `json:"owner"`
	MaxSendAttempts   uint            `json:"max_send_attempts"`
	Contact           string          `json:"contact"`
	RequestReceivedAt time.Time       `json:"request_received_at"`
	Content           string          `json:"content"`
}

// MessageAPISentPayloadV1 is the old event
type MessageAPISentPayloadV1 struct {
	MessageID         uuid.UUID       `json:"id"`
	UserID            entities.UserID `json:"userID"`
	Owner             string          `json:"owner"`
	MaxSendAttempts   uint            `json:"max_send_attempts"`
	Contact           string          `json:"contact"`
	RequestReceivedAt time.Time       `json:"request_received_at"`
	Content           string          `json:"content"`
}
