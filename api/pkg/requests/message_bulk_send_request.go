package requests

import (
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
}

// Sanitize sets defaults to MessageReceive
func (input *MessageBulkSend) Sanitize() MessageBulkSend {
	var to []string
	for _, address := range input.To {
		to = append(to, input.sanitizeAddress(address))
	}
	input.To = to
	input.From = input.sanitizeAddress(input.From)
	return *input
}

// ToMessageSendParams converts MessageSend to services.MessageSendParams
func (input *MessageBulkSend) ToMessageSendParams(userID entities.UserID, source string) []services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.From, phonenumbers.UNKNOWN_REGION)
	var result []services.MessageSendParams
	for _, to := range input.To {
		toAddress, _ := phonenumbers.Parse(to, phonenumbers.UNKNOWN_REGION)
		result = append(result, services.MessageSendParams{
			Source:            source,
			Owner:             *from,
			UserID:            userID,
			RequestReceivedAt: time.Now().UTC(),
			Contact:           *toAddress,
			Content:           input.Content,
		})
	}

	return result
}
