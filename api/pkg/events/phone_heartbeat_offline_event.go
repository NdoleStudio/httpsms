package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypePhoneHeartbeatOffline is emitted when the phone is missing a heartbeat
const EventTypePhoneHeartbeatOffline = "phone.heartbeat.offline"

// PhoneHeartbeatOfflinePayload is the payload of the EventTypePhoneHeartbeatOffline event
type PhoneHeartbeatOfflinePayload struct {
	PhoneID                uuid.UUID       `json:"phone_id"`
	HeartbeatID            uuid.UUID       `json:"heartbeat_id,omitempty"`
	UserID                 entities.UserID `json:"user_id"`
	LastHeartbeatTimestamp time.Time       `json:"last_heartbeat_timestamp"`
	Timestamp              time.Time       `json:"timestamp"`
	MonitorID              uuid.UUID       `json:"monitor_id"`
	Owner                  string          `json:"owner"`
}
