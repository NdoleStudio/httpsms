package requests

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageEvent is the payload for sending and SMS message
type MessageEvent struct {
	request

	// Timestamp is the time when the event was emitted, Please send the timestamp in UTC with as much precision as possible
	Timestamp time.Time `json:"timestamp" example:"2022-06-05T14:26:09.527976+03:00"`

	// EventName is the type of event
	// * SENT: is emitted when a message is sent by the mobile phone
	// * FAILED: is event is emitted when the message could not be sent by the mobile phone
	// * DELIVERED: is event is emitted when a delivery report has been received by the mobile phone
	EventName string `json:"event_name" example:"SENT"`

	// Reason is the exact error message in case the event is an error
	Reason *string `json:"reason"`

	MessageID string `json:"messageID" swaggerignore:"true"` // used internally for validation
}

// Sanitize the message event
func (input *MessageEvent) Sanitize() *MessageEvent {
	input.MessageID = input.sanitizeMessageID(input.MessageID)
	return input
}

// ToMessageStoreEventParams converts MessageEvent to services.MessageStoreEventParams
func (input *MessageEvent) ToMessageStoreEventParams(source string) services.MessageStoreEventParams {
	return services.MessageStoreEventParams{
		MessageID:    uuid.MustParse(input.MessageID),
		Source:       source,
		ErrorMessage: input.Reason,
		EventName:    entities.MessageEventName(input.EventName),
		Timestamp:    input.Timestamp,
	}
}
