package requests

import (
	"strings"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// PhoneUpsert is the payload for updating a phone
type PhoneUpsert struct {
	request
	PhoneNumber string `json:"phone_number" example:"+18005550199"`
	FcmToken    string `json:"fcm_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzd....."`
}

// Sanitize sets defaults to MessageOutstanding
func (input *PhoneUpsert) Sanitize() PhoneUpsert {
	input.FcmToken = strings.TrimSpace(input.FcmToken)
	input.PhoneNumber = strings.TrimSpace(input.PhoneNumber)
	return *input
}

// ToUpsertParams converts PhoneUpsert to services.PhoneUpsertParams
func (input *PhoneUpsert) ToUpsertParams(user entities.AuthUser) services.PhoneUpsertParams {
	return services.PhoneUpsertParams{
		PhoneNumber: input.PhoneNumber,
		FcmToken:    input.FcmToken,
		UserID:      user.ID,
	}
}
