package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageSendExpiredCheck is emitted to trigger checking if a message is expired
const EventTypeMessageSendExpiredCheck = "message.send.expired.check"

// MessageSendExpiredCheckPayload is the payload of the EventTypeMessageSendExpiredCheck event
type MessageSendExpiredCheckPayload struct {
	MessageID   uuid.UUID       `json:"message_id"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	UserID      entities.UserID `json:"user_id"`
}
