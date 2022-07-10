package events

import (
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageNotificationScheduled is emitted when a new message notification is scheduled
const EventTypeMessageNotificationScheduled = "message.notification.scheduled"

// MessageNotificationScheduledPayload is the payload of the MessageNotificationScheduledPayload event
type MessageNotificationScheduledPayload struct {
	MessageID      uuid.UUID       `json:"id"`
	UserID         entities.UserID `json:"user_id"`
	PhoneID        uuid.UUID       `json:"phone_id"`
	ScheduledAt    time.Time       `json:"scheduled_at"`
	NotificationID uuid.UUID       `json:"notification_id"`
}
