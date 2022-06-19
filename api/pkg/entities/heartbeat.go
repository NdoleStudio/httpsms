package entities

import (
	"time"

	"github.com/google/uuid"
)

// Heartbeat represents a message sent between 2 phone numbers
type Heartbeat struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	Owner     string    `json:"owner" gorm:"index:idx_heartbeats_owner_timestamp" example:"+18005550199"`
	Timestamp time.Time `json:"timestamp" gorm:"index:idx_heartbeats_owner_timestamp" example:"2022-06-05T14:26:01.520828+03:00"`
	Quantity  int       `json:"quantity" example:"2"`
}
