package requests

import (
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/google/uuid"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// MessageEvent is the payload for sending and SMS message
type MessageEvent struct {
	// Timestamp is the time when the event was emitted, Please send the timestamp in UTC with as much precision as possible
	Timestamp time.Time `json:"sent_at" example:"2022-06-05T14:26:09.527976+03:00"`

	// EventName is the type of event
	// * SENT: is emitted when a message is sent by the mobile phone (only SENT is implemented)
	// * FAILED: is event is emitted when the message could not be sent by the mobile phone
	// * DELIVERED: is event is emitted when a delivery report has been received by the mobile phone
	EventName string `json:"event_name" example:"SENT"`

	MessageID string `json:"messageID" swaggerignore:"true"` // used internally for validation
}

// ToMessageStoreEventParams converts MessageEvent to services.MessageStorePhoneEventParams
func (input MessageEvent) ToMessageStoreEventParams(source string) services.MessageStorePhoneEventParams {
	return services.MessageStorePhoneEventParams{
		MessageID: uuid.MustParse(input.MessageID),
		Source:    source,
		EventName: entities.MessageEventName(input.EventName),
		Timestamp: input.Timestamp,
	}
}
