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

	// RequestID is an optional parameter used to track a request from the client's perspective
	RequestID string `json:"request_id" example:"153554b5-ae44-44a0-8f4f-7bbac5657ad4" validate:"optional"`
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
		Owner:             *from,
		RequestID:         input.sanitizeStringPointer(input.RequestID),
		UserID:            userID,
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.To),
		Content:           input.Content,
	}
}
