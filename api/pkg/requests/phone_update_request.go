package requests

import (
	"strings"
	"time"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// PhoneUpsert is the payload for updating a phone
type PhoneUpsert struct {
	request
	MessagesPerMinute uint   `json:"messages_per_minute" example:"1"`
	PhoneNumber       string `json:"phone_number" example:"+18005550199"`

	// MessageExpirationSeconds is the duration in seconds after sending a message when it is considered to be expired.
	MessageExpirationSeconds uint `json:"message_expiration_seconds" example:"12345"`

	// MaxSendAttempts is the number of attempts when sending an SMS message to handle the case where the phone is offline.
	MaxSendAttempts uint `json:"max_send_attempts" example:"2"`

	FcmToken string `json:"fcm_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzd....."`
}

// Sanitize sets defaults to MessageOutstanding
func (input *PhoneUpsert) Sanitize() PhoneUpsert {
	input.FcmToken = strings.TrimSpace(input.FcmToken)
	input.PhoneNumber = input.sanitizeAddress(input.PhoneNumber)
	return *input
}

// ToUpsertParams converts PhoneUpsert to services.PhoneUpsertParams
func (input *PhoneUpsert) ToUpsertParams(user entities.AuthUser, source string) services.PhoneUpsertParams {
	phone, _ := phonenumbers.Parse(input.PhoneNumber, phonenumbers.UNKNOWN_REGION)

	// ignore value if it's default
	var messagesPerMinute *uint
	if input.MessagesPerMinute != 0 {
		messagesPerMinute = &input.MessagesPerMinute
	}

	// ignore default
	var fcmToken *string
	if input.FcmToken != "" {
		fcmToken = &input.FcmToken
	}

	// ignore default
	var timeout *time.Duration
	if input.MessageExpirationSeconds != 0 {
		duration := time.Duration(input.MessageExpirationSeconds) * time.Second
		timeout = &duration
	}

	var maxSendAttempts *uint
	if input.MaxSendAttempts != 0 {
		maxSendAttempts = &input.MaxSendAttempts
	}

	return services.PhoneUpsertParams{
		Source:                    source,
		PhoneNumber:               *phone,
		MessagesPerMinute:         messagesPerMinute,
		MessageExpirationDuration: timeout,
		MaxSendAttempts:           maxSendAttempts,
		FcmToken:                  fcmToken,
		UserID:                    user.ID,
	}
}
