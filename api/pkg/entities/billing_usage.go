package entities

import (
	"time"

	"github.com/google/uuid"
)

// BillingUsage tracks the billing usage of an account
type BillingUsage struct {
	ID               uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID           UserID    `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	SentMessages     uint      `json:"sent_messages" example:"321"`
	ReceivedMessages uint      `json:"received_messages" example:"465"`
	StartTimestamp   time.Time `json:"start_timestamp" example:"2022-01-01T00:00:00+00:00"`
	EndTimestamp     time.Time `json:"end_timestamp" example:"2022-01-31T23:59:59+00:00"`
	CreatedAt        time.Time `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt        time.Time `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}
