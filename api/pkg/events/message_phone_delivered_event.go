package events

import (
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessagePhoneDelivered is emitted when the phone delivers a message
const EventTypeMessagePhoneDelivered = "message.phone.delivered"

// MessagePhoneDeliveredPayload is the payload of the EventTypeMessagePhoneDelivered event
type MessagePhoneDeliveredPayload struct {
	ID        uuid.UUID       `json:"id"`
	Owner     string          `json:"owner"`
	Contact   string          `json:"contact"`
	UserID    entities.UserID `json:"user_id"`
	Timestamp time.Time       `json:"timestamp"`
	Content   string          `json:"content"`
}
