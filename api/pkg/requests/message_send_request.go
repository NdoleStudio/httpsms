package requests

import (
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// MessageSend is the payload for sending and SMS message
type MessageSend struct {
	From    string `json:"from" example:"+18005550199"`
	To      string `json:"to" example:"+18005550100"`
	Content string `json:"content" example:"This is a sample text message"`
}

// ToMessageSendParams converts MessageSend to services.MessageSendParams
func (input MessageSend) ToMessageSendParams(source string) services.MessageSendParams {
	return services.MessageSendParams{
		Source:            source,
		From:              input.From,
		RequestReceivedAt: time.Now().UTC(),
		To:                input.To,
		Content:           input.Content,
	}
}
