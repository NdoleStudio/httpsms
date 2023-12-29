package events

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypeWebhookSendFailed is emitted when we can't send a webhook event
const EventTypeWebhookSendFailed = "webhook.send.failed"

// WebhookSendFailedPayload is the payload of the EventTypeWebhookSendFailed event
type WebhookSendFailedPayload struct {
	WebhookID              uuid.UUID       `json:"webhook_id"`
	WebhookURL             string          `json:"webhook_url"`
	Owner                  string          `json:"owner"`
	UserID                 entities.UserID `json:"user_id"`
	EventID                string          `json:"event_id"`
	EventType              string          `json:"event_type"`
	EventPayload           string          `json:"event_payload"`
	HTTPResponseStatusCode *int            `json:"http_response_status_code"`
	ErrorMessage           string          `json:"error_message"`
}
