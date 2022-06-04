package entities

import (
	"time"

	"github.com/google/uuid"
)

// MessageThread represents a message thread between 2 phone numbers
type MessageThread struct {
	ID             uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;"`
	From           string    `json:"from"`
	To             string    `json:"to"`
	LastMessage    string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	OrderTimestamp time.Time `json:"order_timestamp"`
}
