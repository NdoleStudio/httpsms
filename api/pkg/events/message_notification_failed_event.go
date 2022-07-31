package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageNotificationFailed is emitted when a new message notification is failed
const EventTypeMessageNotificationFailed = "message.notification.failed"

// MessageNotificationFailedPayload is the payload of the EventTypeMessageNotificationFailed event
type MessageNotificationFailedPayload struct {
	MessageID            uuid.UUID       `json:"message_id"`
	UserID               entities.UserID `json:"user_id"`
	NotificationID       uuid.UUID       `json:"notification_id"`
	PhoneID              uuid.UUID       `json:"phone_id"`
	ErrorMessage         string          `json:"error_message"`
	NotificationFailedAt time.Time       `json:"notification_failed_at"`
}
