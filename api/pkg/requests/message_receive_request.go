package requests

import (
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// MessageReceive is the payload for sending and SMS message
type MessageReceive struct {
	From    string `json:"from" example:"+18005550199"`
	To      string `json:"to" example:"+18005550100"`
	Content string `json:"content" example:"This is a sample text message received on a phone"`
	// Timestamp is the time when the event was emitted, Please send the timestamp in UTC with as much precision as possible
	Timestamp time.Time `json:"timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
}

// ToMessageReceiveParams converts MessageReceive to services.MessageReceiveParams
func (input MessageReceive) ToMessageReceiveParams(source string) services.MessageReceiveParams {
	return services.MessageReceiveParams{
		Source:    source,
		From:      input.From,
		Timestamp: input.Timestamp,
		To:        input.To,
		Content:   input.Content,
	}
}
