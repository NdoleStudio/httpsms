package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypePhoneHeartbeatCheck is emitted when the phone is missing a heartbeat
const EventTypePhoneHeartbeatCheck = "phone.heartbeat.check"

// PhoneHeartbeatCheckPayload is the payload of the EventTypePhoneHeartbeatCheck event
type PhoneHeartbeatCheckPayload struct {
	PhoneID     uuid.UUID       `json:"phone_id"`
	UserID      entities.UserID `json:"user_id"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	Owner       string          `json:"owner"`
	MonitorID   uuid.UUID       `json:"monitor_id"`
}
