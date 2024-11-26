package requests

import (
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// UserNotificationUpdate is the payload for updating a phone
type UserNotificationUpdate struct {
	request
	MessageStatusEnabled bool `json:"message_status_enabled" example:"true"`
	WebhookEnabled       bool `json:"webhook_enabled"  example:"true"`
	HeartbeatEnabled     bool `json:"heartbeat_enabled" example:"true"`
	NewsletterEnabled    bool `json:"newsletter_enabled" example:"true"`
}

// ToUserNotificationUpdateParams converts UserNotificationUpdate to services.UserNotificationUpdateParams
func (input *UserNotificationUpdate) ToUserNotificationUpdateParams() *services.UserNotificationUpdateParams {
	return &services.UserNotificationUpdateParams{
		MessageStatusEnabled: input.MessageStatusEnabled,
		WebhookEnabled:       input.WebhookEnabled,
		HeartbeatEnabled:     input.HeartbeatEnabled,
		NewsletterEnabled:    input.NewsletterEnabled,
	}
}
