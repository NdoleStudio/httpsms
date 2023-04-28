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
	// sim card to use to send the message
	SIM entities.SIM `json:"sim" example:"DEFAULT"`
}

// Sanitize sets defaults to MessageReceive
func (input *MessageSend) Sanitize() MessageSend {
	input.To = input.sanitizeAddress(input.To)
	input.From = input.sanitizeAddress(input.From)
	if strings.TrimSpace(string(input.SIM)) == "" {
		input.SIM = entities.SIMDefault
	}
	return *input
}

// ToMessageSendParams converts MessageSend to services.MessageSendParams
func (input *MessageSend) ToMessageSendParams(userID entities.UserID, source string) services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.From, phonenumbers.UNKNOWN_REGION)
	return services.MessageSendParams{
		Source:            source,
		Owner:             *from,
		UserID:            userID,
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.To),
		Content:           input.Content,
		SIM:               input.SIM,
	}
}
