package requests

import (
	"strings"

	"github.com/google/uuid"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// UserUpdate is the payload for updating a phone
type UserUpdate struct {
	request
	ActivePhoneID string `json:"active_phone_id" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *UserUpdate) Sanitize() UserUpdate {
	input.ActivePhoneID = strings.TrimSpace(input.ActivePhoneID)
	return *input
}

// ToUpdateParams converts UserUpdate to services.UserUpdateParams
func (input *UserUpdate) ToUpdateParams() services.UserUpdateParams {
	return services.UserUpdateParams{
		ActivePhoneID: uuid.MustParse(input.ActivePhoneID),
	}
}
