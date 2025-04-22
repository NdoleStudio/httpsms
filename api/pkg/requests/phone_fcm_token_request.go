package requests

import (
	"strings"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// PhoneFCMToken is the payload for updating the FCM token of a phone
type PhoneFCMToken struct {
	request
	PhoneNumber string `json:"phone_number"  example:"[+18005550199]"`
	FcmToken    string `json:"fcm_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzd....."`
	// SIM is the SIM slot of the phone in case the phone has more than 1 SIM slot
	SIM string `json:"sim" example:"SIM1"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *PhoneFCMToken) Sanitize() PhoneFCMToken {
	input.FcmToken = strings.TrimSpace(input.FcmToken)
	input.PhoneNumber = input.sanitizeAddress(input.PhoneNumber)
	input.SIM = input.sanitizeSIM(input.SIM)
	return *input
}

// ToPhoneFCMTokenParams converts PhoneFCMToken to services.PhoneFCMTokenParams
func (input *PhoneFCMToken) ToPhoneFCMTokenParams(user entities.AuthContext, source string) *services.PhoneFCMTokenParams {
	phone, _ := phonenumbers.Parse(input.PhoneNumber, phonenumbers.UNKNOWN_REGION)
	return &services.PhoneFCMTokenParams{
		Source:        source,
		PhoneNumber:   phone,
		PhoneAPIKeyID: user.PhoneAPIKeyID,
		UserID:        user.ID,
		FcmToken:      &input.FcmToken,
		SIM:           entities.SIM(input.SIM),
	}
}
