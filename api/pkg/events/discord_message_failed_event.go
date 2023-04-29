package events

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypeDiscordMessageFailed is emitted when we can't send a discord message
const EventTypeDiscordMessageFailed = "discord.message.failed"

// DiscordMessageFailedPayload is the payload of the EventTypeDiscordMessageFailed event
type DiscordMessageFailedPayload struct {
	DiscordID        uuid.UUID       `json:"discord_id"`
	UserID           entities.UserID `json:"user_id"`
	MessageID        uuid.UUID       `json:"message_id"`
	EventType        string          `json:"event_type"`
	HTTPStatusCode   int             `json:"http_status_code"`
	ErrorMessage     string          `json:"error_message"`
	DiscordChannelID string          `json:"discord_channel_id"`
}
