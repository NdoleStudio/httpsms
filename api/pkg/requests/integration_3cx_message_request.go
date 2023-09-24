package requests

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// Integration3CXMessage is the payload for sending and SMS message via 3CX
type Integration3CXMessage struct {
	request
	From string `json:"from" example:"+18005550199"`
	To   string `json:"to" example:"+18005550100"`
	Text string `json:"text" example:"This is a sample text message"`
}

// Sanitize sets defaults to MessageReceive
func (input *Integration3CXMessage) Sanitize() Integration3CXMessage {
	input.To = input.sanitizeAddress(input.To)
	input.From = input.sanitizeAddress(input.From)
	return *input
}

// ToMessageSendParams converts Integration3CXMessage to services.MessageSendParams
func (input *Integration3CXMessage) ToMessageSendParams(userID entities.UserID, source string) services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.From, phonenumbers.UNKNOWN_REGION)
	return services.MessageSendParams{
		Source:            source,
		Owner:             *from,
		RequestID:         input.sanitizeStringPointer("integration-3cx"),
		UserID:            userID,
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.To),
		Content:           input.Text,
	}
}
