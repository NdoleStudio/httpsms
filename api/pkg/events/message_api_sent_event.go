package events

import (
	"time"

	"github.com/google/uuid"
)

// EventTypeMessageAPISent is emitted when a new message is sent
const EventTypeMessageAPISent = "message.api.sent"

// MessageAPISentPayload is the payload of the EventTypeMessageSent event
type MessageAPISentPayload struct {
	ID                uuid.UUID `json:"id"`
	Owner             string    `json:"owner"`
	Contact           string    `json:"contact"`
	RequestReceivedAt time.Time `json:"request_received_at"`
	Content           string    `json:"content"`
}
