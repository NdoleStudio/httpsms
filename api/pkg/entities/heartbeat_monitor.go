package entities

import (
	"github.com/google/uuid"
)

// HeartbeatMonitor is used to monitor heartbeats of a phone
type HeartbeatMonitor struct {
	ID      uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	PhoneID uuid.UUID `json:"phone_id" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID  UserID    `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Owner   string    `json:"owner" example:"+18005550199"`
}
