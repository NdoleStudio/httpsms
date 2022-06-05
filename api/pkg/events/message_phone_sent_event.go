package events

import "github.com/google/uuid"

// EventTypeMessagePhoneSent is emitted when the phone sends a message
const EventTypeMessagePhoneSent = "message.phone.sent"

// MessagePhoneSentPayload is the payload of the EventTypeMessagePhoneSent event
type MessagePhoneSentPayload struct {
	ID      uuid.UUID `json:"id"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	Content string    `json:"content"`
}
