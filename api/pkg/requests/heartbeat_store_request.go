package requests

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// HeartbeatStore is the payload for fetching entities.Heartbeat of a phone number
type HeartbeatStore struct {
	request
	Owner        string   `json:"owner" swaggerignore:"true"`
	Charging     bool     `json:"charging"`
	PhoneNumbers []string `json:"phone_numbers"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *HeartbeatStore) Sanitize() HeartbeatStore {
	input.Owner = input.sanitizeAddress(input.Owner)

	input.PhoneNumbers = input.sanitizeAddresses(input.PhoneNumbers)
	if len(input.PhoneNumbers) == 0 {
		input.PhoneNumbers = append(input.PhoneNumbers, input.Owner)
	}

	return *input
}

// ToStoreParams converts HeartbeatIndex to repositories.IndexParams
func (input *HeartbeatStore) ToStoreParams(user entities.AuthContext, source string, version string) []services.HeartbeatStoreParams {
	var params []services.HeartbeatStoreParams
	for _, phoneNumber := range input.PhoneNumbers {
		params = append(params, services.HeartbeatStoreParams{
			Owner:     phoneNumber,
			Charging:  input.Charging,
			Source:    source,
			Version:   version,
			UserID:    user.ID,
			Timestamp: time.Now(),
		})
	}
	return params
}
