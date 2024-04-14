package entities

import (
	"time"

	"github.com/google/uuid"
)

// Phone represents an android phone which has installed the http sms app
type Phone struct {
	ID                uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID            UserID    `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	FcmToken          *string   `json:"fcm_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzd....."`
	PhoneNumber       string    `json:"phone_number" example:"+18005550199"`
	MessagesPerMinute uint      `json:"messages_per_minute" example:"1"`
	SIM               SIM       `json:"sim" gorm:"default:SIM1"`
	// MaxSendAttempts determines how many times to retry sending an SMS message
	MaxSendAttempts uint `json:"max_send_attempts" example:"2"`

	// MessageExpirationSeconds is the duration in seconds after sending a message when it is considered to be expired.
	MessageExpirationSeconds uint `json:"message_expiration_seconds"`

	MissedCallAutoReply *string `json:"missed_call_auto_reply" example:"This phone cannot receive calls. Please send an SMS instead."`

	CreatedAt time.Time `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt time.Time `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}

// MessageExpirationDuration returns the message expiration as time.Duration
func (phone *Phone) MessageExpirationDuration() time.Duration {
	return time.Duration(int(phone.MessageExpirationSecondsSanitized())) * time.Second
}

// MessageExpirationSecondsSanitized returns the message expiration seconds with default of 1 hour
func (phone *Phone) MessageExpirationSecondsSanitized() uint {
	if phone.MessageExpirationSeconds == 0 {
		return 10 * 60 // 10 minutes
	}
	return phone.MessageExpirationSeconds
}

// MaxSendAttemptsSanitized returns the max send attempts replacing 0 with 2
func (phone *Phone) MaxSendAttemptsSanitized() uint {
	if phone.MaxSendAttempts == 0 {
		return 2
	}
	return phone.MaxSendAttempts
}
