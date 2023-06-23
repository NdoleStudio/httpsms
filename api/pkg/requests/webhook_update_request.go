package requests

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/google/uuid"
)

// WebhookUpdate is the payload for updating an entities.Webhook
type WebhookUpdate struct {
	WebhookStore
	WebhookID string `json:"webhookID" swaggerignore:"true"` // used internally for validation
}

// Sanitize sets defaults to WebhookUpdate
func (input *WebhookUpdate) Sanitize() WebhookUpdate {
	input.WebhookStore.Sanitize()
	return *input
}

// ToUpdateParams converts WebhookUpdate to services.WebhookUpdateParams
func (input *WebhookUpdate) ToUpdateParams(user entities.AuthUser) *services.WebhookUpdateParams {
	return &services.WebhookUpdateParams{
		UserID:       user.ID,
		WebhookID:    uuid.MustParse(input.WebhookID),
		SigningKey:   input.SigningKey,
		URL:          input.URL,
		PhoneNumbers: input.PhoneNumbers,
		Events:       input.Events,
	}
}
