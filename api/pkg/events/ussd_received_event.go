package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeUSSDReceived is emitted when a USSD request is received from a mobile phone
const EventTypeUSSDReceived = "ussd.phone.received"

// USSDReceivedPayload is the payload of the EventTypeUSSDReceived event
type USSDReceivedPayload struct {
	USSDID    uuid.UUID       `json:"ussd_id"`
	UserID    entities.UserID `json:"user_id"`
	PhoneID   uuid.UUID       `json:"phone_id"`
	Owner     string          `json:"owner"`
	SessionID string          `json:"session_id"`
	Content   string          `json:"content"`
	SIM       entities.SIM    `json:"sim"`
	Timestamp time.Time       `json:"timestamp"`
}