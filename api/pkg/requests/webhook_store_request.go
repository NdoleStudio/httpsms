package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// WebhookStore is the payload for creating a new entities.Webhook
type WebhookStore struct {
	request
	SigningKey   string   `json:"signing_key"`
	URL          string   `json:"url"`
	PhoneNumbers []string `json:"phone_numbers" example:"+18005550100,+18005550100"`
	Events       []string `json:"events"`
}

// Sanitize sets defaults to WebhookStore
func (input *WebhookStore) Sanitize() WebhookStore {
	input.URL = input.sanitizeURL(input.URL)
	input.SigningKey = strings.TrimSpace(input.SigningKey)
	input.Events = input.removeStringDuplicates(input.Events)

	var phoneNumbers []string
	for _, address := range input.PhoneNumbers {
		phoneNumbers = append(phoneNumbers, input.sanitizeAddress(address))
	}

	return *input
}

// ToStoreParams converts WebhookStore to services.WebhookStoreParams
func (input *WebhookStore) ToStoreParams(user entities.AuthContext) *services.WebhookStoreParams {
	return &services.WebhookStoreParams{
		UserID:       user.ID,
		SigningKey:   input.SigningKey,
		URL:          input.URL,
		PhoneNumbers: input.PhoneNumbers,
		Events:       input.Events,
	}
}
