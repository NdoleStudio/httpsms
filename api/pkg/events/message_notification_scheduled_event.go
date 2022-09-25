package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageNotificationScheduled is emitted when a new message notification is scheduled
const EventTypeMessageNotificationScheduled = "message.notification.scheduled"

// MessageNotificationScheduledPayload is the payload of the EventTypeMessageNotificationScheduled event
type MessageNotificationScheduledPayload struct {
	MessageID      uuid.UUID       `json:"id"`
	Owner          string          `json:"owner"`
	Contact        string          `json:"contact"`
	Content        string          `json:"content"`
	UserID         entities.UserID `json:"user_id"`
	PhoneID        uuid.UUID       `json:"phone_id"`
	ScheduledAt    time.Time       `json:"scheduled_at"`
	NotificationID uuid.UUID       `json:"notification_id"`
}
