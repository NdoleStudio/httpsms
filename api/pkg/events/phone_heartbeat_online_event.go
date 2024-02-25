package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypePhoneHeartbeatOnline is emitted when the phone is missing a heartbeat
const EventTypePhoneHeartbeatOnline = "phone.heartbeat.online"

// PhoneHeartbeatOnlinePayload is the payload of the EventTypePhoneHeartbeatOnline event
type PhoneHeartbeatOnlinePayload struct {
	PhoneID                uuid.UUID       `json:"phone_id"`
	UserID                 entities.UserID `json:"user_id"`
	LastHeartbeatTimestamp time.Time       `json:"last_heartbeat_timestamp"`
	Timestamp              time.Time       `json:"timestamp"`
	MonitorID              uuid.UUID       `json:"monitor_id"`
	Owner                  string          `json:"owner"`
}
