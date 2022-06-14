package events

import (
	"github.com/google/uuid"
)

// EventTypeMessagePhoneSending is emitted when a message is picked up by the phone and is being sent
const EventTypeMessagePhoneSending = "message.phone.sending"

// MessagePhoneSendingPayload is the payload of the EventTypeMessageSent event
type MessagePhoneSendingPayload struct {
	ID      uuid.UUID `json:"id"`
	Owner   string    `json:"owner"`
	Contact string    `json:"contact"`
	Content string    `json:"content"`
}
