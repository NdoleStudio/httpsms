package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// WebhookStore is the payload for creating a new entities.Webhook
type WebhookStore struct {
	request
	SigningKey string   `json:"signing_key"`
	URL        string   `json:"url"`
	Events     []string `json:"events"`
}

// Sanitize sets defaults to WebhookStore
func (input *WebhookStore) Sanitize() WebhookStore {
	input.URL = strings.TrimSpace(input.URL)
	input.Events = input.removeStringDuplicates(input.Events)
	return *input
}

// ToStoreParams converts WebhookStore to services.WebhookStoreParams
func (input *WebhookStore) ToStoreParams(user entities.AuthUser) *services.WebhookStoreParams {
	return &services.WebhookStoreParams{
		UserID:     user.ID,
		SigningKey: input.SigningKey,
		URL:        input.URL,
		Events:     input.Events,
	}
}
