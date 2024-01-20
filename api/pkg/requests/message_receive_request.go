package requests

import (
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageReceive is the payload for sending and SMS message
type MessageReceive struct {
	request
	From    string `json:"from" example:"+18005550199"`
	To      string `json:"to" example:"+18005550100"`
	Content string `json:"content" example:"This is a sample text message received on a phone"`
	// Encrypted is used to determine if the content is end-to-end encrypted. Make sure to set the encryption key on the httpSMS mobile app
	Encrypted bool `json:"encrypted" example:"false"`
	// SIM card that received the message
	SIM entities.SIM `json:"sim" example:"SIM1"`
	// Timestamp is the time when the event was emitted, Please send the timestamp in UTC with as much precision as possible
	Timestamp time.Time `json:"timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
}

// Sanitize sets defaults to MessageReceive
func (input *MessageReceive) Sanitize() MessageReceive {
	input.To = input.sanitizeAddress(input.To)
	input.From = input.sanitizeAddress(input.From)
	if strings.TrimSpace(string(input.SIM)) == "" || input.SIM == ("DEFAULT") {
		input.SIM = entities.SIM1
	}
	return *input
}

// ToMessageReceiveParams converts MessageReceive to services.MessageReceiveParams
func (input *MessageReceive) ToMessageReceiveParams(userID entities.UserID, source string) *services.MessageReceiveParams {
	phone, _ := phonenumbers.Parse(input.To, phonenumbers.UNKNOWN_REGION)
	return &services.MessageReceiveParams{
		Source:    source,
		Contact:   input.From,
		UserID:    userID,
		Timestamp: input.Timestamp,
		Encrypted: input.Encrypted,
		Owner:     *phone,
		Content:   input.Content,
		SIM:       input.SIM,
	}
}
