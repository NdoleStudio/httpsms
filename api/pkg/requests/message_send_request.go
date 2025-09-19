package requests

import (
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageSend is the payload for sending and SMS message
type MessageSend struct {
	request
	From    string `json:"from" example:"+18005550199"`
	To      string `json:"to" example:"+18005550100"`
	Content string `json:"content" example:"This is a sample text message"`

	// Encrypted is an optional parameter used to determine if the content is end-to-end encrypted. Make sure to set the encryption key on the httpSMS mobile app
	Encrypted bool `json:"encrypted" example:"false" validate:"optional"`
	// RequestID is an optional parameter used to track a request from the client's perspective
	RequestID string `json:"request_id" example:"153554b5-ae44-44a0-8f4f-7bbac5657ad4" validate:"optional"`
	// SendAt is an optional parameter used to schedule a message to be sent in the future. The time is considered to be in your profile's local timezone and you can queue messages for up to 20 days (480 hours) in the future.
	SendAt *time.Time `json:"send_at" example:"2022-06-05T14:26:09.527976+03:00" validate:"optional"`
}

// Sanitize sets defaults to MessageReceive
func (input *MessageSend) Sanitize() MessageSend {
	input.To = input.sanitizeAddress(input.To)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.From = input.sanitizeAddress(input.From)
	return *input
}

// ToMessageSendParams converts MessageSend to services.MessageSendParams
func (input *MessageSend) ToMessageSendParams(userID entities.UserID, source string) services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.From, phonenumbers.UNKNOWN_REGION)
	return services.MessageSendParams{
		Source:            source,
		Owner:             from,
		Encrypted:         input.Encrypted,
		RequestID:         input.sanitizeStringPointer(input.RequestID),
		UserID:            userID,
		SendAt:            input.SendAt,
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.To),
		Content:           input.Content,
	}
}
