package events

// EventTypeMessageAPISent is emitted when a new message is sent
const EventTypeMessageAPISent = "message.api.sent"

// MessageAPISent is the payload of the EventTypeMessageSent event
type MessageAPISent struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Content string `json:"content"`
}
