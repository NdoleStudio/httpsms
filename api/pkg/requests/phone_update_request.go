package requests

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

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

	MissedCallAutoReply *string `json:"missed_call_auto_reply" example:"e.g. This phone cannot receive calls. Please send an SMS instead."`

	// SIM is the SIM slot of the phone in case the phone has more than 1 SIM slot
	SIM string `json:"sim" example:"SIM1"`

	MessageSendScheduleID string `json:"message_send_schedule_id,omitempty" example:"32343a19-da5e-4b1b-a767-3298a73703cb" validate:"optional"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *PhoneUpsert) Sanitize() PhoneUpsert {
	input.FcmToken = strings.TrimSpace(input.FcmToken)
	input.PhoneNumber = input.sanitizeAddress(input.PhoneNumber)
	input.SIM = input.sanitizeSIM(input.SIM)
	if input.MissedCallAutoReply != nil {
		input.MissedCallAutoReply = input.sanitizeStringPointer(*input.MissedCallAutoReply)
	}
	return *input
}

// ToUpsertParams converts PhoneUpsert to services.PhoneUpsertParams.
// The body parameter is the raw JSON request body used to detect which fields were explicitly sent.
func (input *PhoneUpsert) ToUpsertParams(user entities.AuthContext, source string, body []byte) *services.PhoneUpsertParams {
	phone, _ := phonenumbers.Parse(input.PhoneNumber, phonenumbers.UNKNOWN_REGION)

	fields := make(map[string]json.RawMessage)
	_ = json.Unmarshal(body, &fields)

	var messagesPerMinute *uint
	if _, exists := fields["messages_per_minute"]; exists {
		messagesPerMinute = &input.MessagesPerMinute
	}

	var fcmToken *string
	if _, exists := fields["fcm_token"]; exists {
		fcmToken = &input.FcmToken
	}

	var timeout *time.Duration
	if _, exists := fields["message_expiration_seconds"]; exists {
		duration := time.Duration(input.MessageExpirationSeconds) * time.Second
		timeout = &duration
	}

	var maxSendAttempts *uint
	if _, exists := fields["max_send_attempts"]; exists {
		maxSendAttempts = &input.MaxSendAttempts
	}

	var scheduleID *uuid.UUID
	if _, exists := fields["message_send_schedule_id"]; exists {
		if parsed, err := uuid.Parse(strings.TrimSpace(input.MessageSendScheduleID)); err == nil {
			scheduleID = &parsed
		}
	}

	return &services.PhoneUpsertParams{
		Source:                    source,
		PhoneNumber:               phone,
		MessagesPerMinute:         messagesPerMinute,
		MissedCallAutoReply:       input.MissedCallAutoReply,
		MessageExpirationDuration: timeout,
		MaxSendAttempts:           maxSendAttempts,
		FcmToken:                  fcmToken,
		UserID:                    user.ID,
		SIM:                       entities.SIM(input.SIM),
		MessageSendScheduleID:     scheduleID,
	}
}
