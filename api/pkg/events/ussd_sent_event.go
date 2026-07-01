package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeUSSDResponse is emitted when a USSD response is sent to a mobile phone
const EventTypeUSSDResponse = "ussd.phone.sent"

// USSDResponsePayload is the payload of the EventTypeUSSDResponse event
type USSDResponsePayload struct {
	USSDID    uuid.UUID       `json:"ussd_id"`
	UserID    entities.UserID `json:"user_id"`
	PhoneID   uuid.UUID       `json:"phone_id"`
	Owner     string          `json:"owner"`
	SessionID string          `json:"session_id"`
	Response  string          `json:"response"`
	SIM       entities.SIM    `json:"sim"`
	Timestamp time.Time       `json:"timestamp"`
}