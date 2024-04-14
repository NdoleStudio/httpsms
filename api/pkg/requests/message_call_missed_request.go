package requests

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageCallMissed is the payload for sending and missed call event
type MessageCallMissed struct {
	request
	From      string    `json:"from" example:"+18005550199"`
	To        string    `json:"to" example:"+18005550100"`
	SIM       string    `json:"sim" example:"SIM1"`
	Timestamp time.Time `json:"timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
}

// Sanitize sets defaults to MessageReceive
func (input *MessageCallMissed) Sanitize() MessageCallMissed {
	input.To = input.sanitizeAddress(input.To)
	input.From = input.sanitizeAddress(input.From)
	input.SIM = input.sanitizeSIM(input.SIM)

	return *input
}

// ToCallMissedParams converts MessageCallMissed to services.MessageSendParams
func (input *MessageCallMissed) ToCallMissedParams(userID entities.UserID, source string) *services.MissedCallParams {
	to, _ := phonenumbers.Parse(input.To, phonenumbers.UNKNOWN_REGION)
	return &services.MissedCallParams{
		Source:    source,
		Owner:     to,
		Timestamp: input.Timestamp,
		SIM:       entities.SIM(input.SIM),
		UserID:    userID,
		Contact:   input.From,
	}
}
