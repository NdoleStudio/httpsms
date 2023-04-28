package entities

import (
	"time"

	"github.com/google/uuid"
)

// Discord stores the discord integration of a user
type Discord struct {
	ID                uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID            UserID    `json:"user_id" gorm:"index" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Name              string    `json:"name" example:"Game Server"`
	ServerID          string    `json:"server_id" gorm:"uniqueIndex" example:"1095780203256627291"`
	IncomingChannelID string    `json:"incoming_channel_id" example:"1095780203256627291"`
	CreatedAt         time.Time `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt         time.Time `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}
