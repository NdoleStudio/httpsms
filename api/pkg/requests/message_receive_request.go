package requests

import (
	"time"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// MessageReceive is the payload for sending and SMS message
type MessageReceive struct {
	request
	From    string `json:"from" example:"+18005550199"`
	To      string `json:"to" example:"+18005550100"`
	Content string `json:"content" example:"This is a sample text message received on a phone"`
	// Timestamp is the time when the event was emitted, Please send the timestamp in UTC with as much precision as possible
	Timestamp time.Time `json:"timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
}

// Sanitize sets defaults to MessageReceive
func (input *MessageReceive) Sanitize() MessageReceive {
	input.To = input.sanitizeAddress(input.To)
	input.From = input.sanitizeAddress(input.From)
	return *input
}

// ToMessageReceiveParams converts MessageReceive to services.MessageReceiveParams
func (input MessageReceive) ToMessageReceiveParams(source string) services.MessageReceiveParams {
	phone, _ := phonenumbers.Parse(input.To, phonenumbers.UNKNOWN_REGION)
	return services.MessageReceiveParams{
		Source:    source,
		Contact:   input.From,
		Timestamp: input.Timestamp,
		Owner:     *phone,
		Content:   input.Content,
	}
}
