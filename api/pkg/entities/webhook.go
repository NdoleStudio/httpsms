package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Webhook stores the webhooks of a user
type Webhook struct {
	ID         uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID     UserID         `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	URL        string         `json:"url" example:"https://example.com"`
	SigningKey string         `json:"signing_key" example:"DGW8NwQp7mxKaSZ72Xq9v67SLqSbWQvckzzmK8D6rvd7NywSEkdMJtuxKyEkYnCY"`
	Events     pq.StringArray `json:"events" example:"[message.phone.received]"`
	CreatedAt  time.Time      `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt  time.Time      `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}
