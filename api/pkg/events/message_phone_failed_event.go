package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypeMessagePhoneFailed is emitted when the phone could not send
const EventTypeMessagePhoneFailed = "message.phone.failed"

// MessagePhoneFailedPayload is the payload of the EventTypeMessagePhoneFailed event
type MessagePhoneFailedPayload struct {
	ID        uuid.UUID       `json:"id"`
	UserID    entities.UserID `json:"user_id"`
	Owner     string          `json:"owner"`
	Contact   string          `json:"contact"`
	Timestamp time.Time       `json:"timestamp"`
	Content   string          `json:"content"`
}
