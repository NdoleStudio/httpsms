package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// WebhookResponse is the payload containing entities.Webhook
type WebhookResponse struct {
	response
	Data entities.Webhook `json:"data"`
}

// WebhooksResponse is the payload containing []entities.Webhook
type WebhooksResponse struct {
	response
	Data []entities.Webhook `json:"data"`
}
