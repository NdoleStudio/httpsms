package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/google/uuid"
)

// EventTypeMessageNotificationSent is emitted when a new message notification is scheduled
const EventTypeMessageNotificationSent = "message.notification.sent"

// MessageNotificationSentPayload is the payload of the EventTypeMessageNotificationSent event
type MessageNotificationSentPayload struct {
	MessageID          uuid.UUID       `json:"message_id"`
	UserID             entities.UserID `json:"user_id"`
	PhoneID            uuid.UUID       `json:"phone_id"`
	ScheduledAt        time.Time       `json:"scheduled_at"`
	FcmMessageID       string          `json:"fcm_message_id"`
	NotificationSentAt time.Time       `json:"notification_sent_at"`
	NotificationID     uuid.UUID       `json:"notification_id"`
}
