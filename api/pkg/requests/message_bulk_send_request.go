package requests

import (
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageBulkSend is the payload for sending bulk SMS messages
type MessageBulkSend struct {
	request
	From    string   `json:"from" example:"+18005550199"`
	To      []string `json:"to" example:"+18005550100,+18005550100"`
	Content string   `json:"content" example:"This is a sample text message"`

	// Attachments are optional. When you provide a list of attachments, the message will be sent out as an MMS
	Attachments []string `json:"attachments" validate:"optional"`

	// Encrypted is used to determine if the content is end-to-end encrypted. Make sure to set the encryption key on the httpSMS mobile app
	Encrypted bool `json:"encrypted" example:"false" validate:"optional"`

	// RequestID is an optional parameter used to track a request from the client's perspective
	RequestID string `json:"request_id" example:"153554b5-ae44-44a0-8f4f-7bbac5657ad4" validate:"optional"`
}

// Sanitize sets defaults to MessageReceive
func (input *MessageBulkSend) Sanitize() MessageBulkSend {
	var to []string
	for _, address := range input.To {
		to = append(to, input.sanitizeAddress(address))
	}

	var attachments []string
	for _, attachment := range input.Attachments {
		if strings.TrimSpace(attachment) != "" {
			attachments = append(attachments, strings.TrimSpace(attachment))
		}
	}

	input.Attachments = attachments
	input.To = to
	input.From = input.sanitizeAddress(input.From)
	return *input
}

// ToMessageSendParams converts MessageSend to services.MessageSendParams
func (input *MessageBulkSend) ToMessageSendParams(userID entities.UserID, source string) []services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.From, phonenumbers.UNKNOWN_REGION)

	var result []services.MessageSendParams
	for index, to := range input.To {
		sendAt := time.Now().UTC().Add(time.Duration(index) * time.Second)
		result = append(result, services.MessageSendParams{
			Source:            source,
			Owner:             from,
			Encrypted:         input.Encrypted,
			RequestID:         input.sanitizeStringPointer(input.RequestID),
			UserID:            userID,
			RequestReceivedAt: time.Now().UTC(),
			Contact:           to,
			SendAt:            &sendAt,
			Content:           input.Content,
			Attachments:       input.Attachments,
		})
	}

	return result
}
