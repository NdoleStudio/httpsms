package entities

import (
	"time"

	"github.com/google/uuid"
)

const (
	// PhoneNotificationStatusPending is the status when a notification is scheduled to be sent
	PhoneNotificationStatusPending = "pending"
	// PhoneNotificationStatusSent is the status when a notification has been sent
	PhoneNotificationStatusSent = "sent"
	// PhoneNotificationStatusFailed is the status when a notification could not be sent.
	PhoneNotificationStatusFailed = "failed"
)

// PhoneNotificationStatus is the status of a phone notification
type PhoneNotificationStatus string

// PhoneNotification represents an FCM notification to a mobile phone
type PhoneNotification struct {
	ID          uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;"`
	MessageID   uuid.UUID `json:"message_id"`
	UserID      UserID    `json:"user_id"`
	PhoneID     uuid.UUID `json:"phone_id"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
