package events

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypeDiscordSendFailed is emitted when we can't send a discord message
const EventTypeDiscordSendFailed = "discord.send.failed"

// DiscordSendFailedPayload is the payload of the EventTypeDiscordSendFailed event
type DiscordSendFailedPayload struct {
	DiscordID              uuid.UUID       `json:"discord_id"`
	UserID                 entities.UserID `json:"user_id"`
	MessageID              uuid.UUID       `json:"message_id"`
	EventType              string          `json:"event_type"`
	Owner                  string          `json:"owner"`
	HTTPResponseStatusCode *int            `json:"http_response_status_code"`
	ErrorMessage           string          `json:"error_message"`
	DiscordChannelID       string          `json:"discord_channel_id"`
}
