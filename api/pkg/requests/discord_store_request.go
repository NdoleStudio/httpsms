package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// DiscordStore is the payload for creating a new entities.Discord
type DiscordStore struct {
	request
	Name              string `json:"name"`
	ServerID          string `json:"server_id"`
	IncomingChannelID string `json:"incoming_channel_id"`
}

// Sanitize sets defaults to DiscordStore
func (input *DiscordStore) Sanitize() DiscordStore {
	input.Name = strings.TrimSpace(input.Name)
	input.ServerID = strings.TrimSpace(input.ServerID)
	input.IncomingChannelID = strings.TrimSpace(input.IncomingChannelID)
	return *input
}

// ToStoreParams converts DiscordStore to services.WebhookStoreParams
func (input *DiscordStore) ToStoreParams(user entities.AuthUser) *services.DiscordStoreParams {
	return &services.DiscordStoreParams{
		UserID:            user.ID,
		Name:              input.Name,
		ServerID:          input.ServerID,
		IncomingChannelID: input.IncomingChannelID,
	}
}
