package entities

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// HeartbeatCheckClaimLease bounds recovery of interrupted heartbeat check claims.
const HeartbeatCheckClaimLease = time.Minute

// HeartbeatMonitor is used to monitor heartbeats of a phone
type HeartbeatMonitor struct {
	ID      uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;" bson:"_id" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	PhoneID uuid.UUID `json:"phone_id" bson:"phone_id" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID  UserID    `json:"user_id" bson:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	// QueueID is the current check ownership marker, not the external task queue's ID.
	QueueID string `json:"queue_id" bson:"queue_id" example:"0360259236613675274"`
	// LastHeartbeatID fences delayed offline transitions independently of check leases.
	LastHeartbeatID uuid.UUID `json:"-" bson:"last_heartbeat_id,omitempty" gorm:"type:uuid"`
	Owner           string    `json:"owner" bson:"owner" example:"+18005550199"`
	PhoneOnline     bool      `json:"phone_online" bson:"phone_online" example:"true" default:"true"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt       time.Time `json:"updated_at" bson:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}

// RequiresCheck returns true if the heartbeat monitor requires a check
func (h *HeartbeatMonitor) RequiresCheck() bool {
	return h.QueueID == "" || h.UpdatedAt.Add(2*time.Hour).Before(time.Now()) ||
		(strings.HasPrefix(h.QueueID, "pending::") && h.UpdatedAt.Add(HeartbeatCheckClaimLease).Before(time.Now()))
}

// PhoneIsOffline returns true if the phone is offline
func (h *HeartbeatMonitor) PhoneIsOffline() bool {
	return !h.PhoneOnline
}
