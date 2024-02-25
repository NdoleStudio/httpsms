package entities

import (
	"time"

	"github.com/google/uuid"
)

// HeartbeatMonitor is used to monitor heartbeats of a phone
type HeartbeatMonitor struct {
	ID          uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	PhoneID     uuid.UUID `json:"phone_id" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID      UserID    `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	QueueID     string    `json:"queue_id" example:"0360259236613675274"`
	Owner       string    `json:"owner" example:"+18005550199"`
	PhoneOnline bool      `json:"phone_online" example:"true" default:"true"`
	CreatedAt   time.Time `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt   time.Time `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}

// RequiresCheck returns true if the heartbeat monitor requires a check
func (h *HeartbeatMonitor) RequiresCheck() bool {
	return h.UpdatedAt.Add(2 * time.Hour).Before(time.Now())
}

// PhoneIsOffline returns true if the phone is offline
func (h *HeartbeatMonitor) PhoneIsOffline() bool {
	return !h.PhoneOnline
}
