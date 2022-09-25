package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageNotificationSend is emitted when we are to send a phone notification
const EventTypeMessageNotificationSend = "message.notification.send"

// MessageNotificationSendPayload is the payload of the EventTypeMessageNotificationSend event
type MessageNotificationSendPayload struct {
	MessageID      uuid.UUID       `json:"id"`
	UserID         entities.UserID `json:"user_id"`
	PhoneID        uuid.UUID       `json:"phone_id"`
	ScheduledAt    time.Time       `json:"scheduled_at"`
	NotificationID uuid.UUID       `json:"notification_id"`
}
