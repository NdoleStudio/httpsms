package requests

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/google/uuid"
)

// DiscordUpdate is the payload for updating an entities.Webhook
type DiscordUpdate struct {
	DiscordStore
	DiscordID string `json:"discordID" swaggerignore:"true"` // used internally for validation
}

// Sanitize sets defaults to WebhookUpdate
func (input *DiscordUpdate) Sanitize() DiscordUpdate {
	input.DiscordStore.Sanitize()
	return *input
}

// ToUpdateParams converts DiscordUpdate to services.DiscordUpdateParams
func (input *DiscordUpdate) ToUpdateParams(user entities.AuthUser) *services.DiscordUpdateParams {
	return &services.DiscordUpdateParams{
		UserID:            user.ID,
		Name:              input.Name,
		ServerID:          input.ServerID,
		IncomingChannelID: input.IncomingChannelID,
		DiscordID:         uuid.MustParse(input.DiscordID),
	}
}
