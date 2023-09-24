package entities

import (
	"time"

	"github.com/google/uuid"
)

// Integration3CX stores the discord integration of a user
type Integration3CX struct {
	ID         uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID     UserID    `json:"user_id" gorm:"index" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	WebhookURL string    `json:"webhook_url" example:"https://org.3cx.com.au/sms/generic/123"`
	CreatedAt  time.Time `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt  time.Time `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}
