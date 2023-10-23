package entities

import (
	"time"

	"github.com/google/uuid"
)

// Heartbeat represents is a pulse from an active phone
type Heartbeat struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	Owner     string    `json:"owner" gorm:"index:idx_heartbeats_owner_timestamp" example:"+18005550199"`
	Version   string    `json:"version" example:"344c10f"`
	Charging  bool      `json:"charging" example:"true"`
	UserID    UserID    `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Timestamp time.Time `json:"timestamp" gorm:"index:idx_heartbeats_owner_timestamp" example:"2022-06-05T14:26:01.520828+03:00"`
}
