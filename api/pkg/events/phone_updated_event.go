package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypePhoneUpdated is emitted when the phone is updated
const EventTypePhoneUpdated = "phone.updated"

// PhoneUpdatedPayload is the payload of the EventTypePhoneUpdated event
type PhoneUpdatedPayload struct {
	PhoneID   uuid.UUID       `json:"phone_id"`
	UserID    entities.UserID `json:"user_id"`
	Timestamp time.Time       `json:"timestamp"`
	Owner     string          `json:"owner"`
	SIM       entities.SIM    `json:"sim"`
}
