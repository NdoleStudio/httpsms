package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// PhoneHeartbeatMissed is emitted when the phone is missing a heartbeat
const PhoneHeartbeatMissed = "phone.heartbeat.missed"

// PhoneHeartbeatMissedPayload is the payload of the PhoneHeartbeatMissed event
type PhoneHeartbeatMissedPayload struct {
	PhoneID                uuid.UUID       `json:"phone_id"`
	UserID                 entities.UserID `json:"user_id"`
	LastHeartbeatTimestamp time.Time       `json:"last_heartbeat_timestamp"`
	Timestamp              time.Time       `json:"timestamp"`
	MonitorID              uuid.UUID       `json:"monitor_id"`
	Owner                  string          `json:"owner"`
}
