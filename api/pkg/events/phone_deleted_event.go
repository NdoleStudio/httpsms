package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypePhoneDeleted is emitted when the phone os deleted
const EventTypePhoneDeleted = "phone.deleted"

// PhoneDeletedPayload is the payload of the EventTypePhoneDeleted event
type PhoneDeletedPayload struct {
	PhoneID   uuid.UUID       `json:"phone_id"`
	UserID    entities.UserID `json:"user_id"`
	Timestamp time.Time       `json:"timestamp"`
	Owner     string          `json:"owner"`
}
