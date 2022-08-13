package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypePhoneHeartbeatDead is emitted when the phone is missing a heartbeat
const EventTypePhoneHeartbeatDead = "phone.heartbeat.dead"

// PhoneHeartbeatDeadPayload is the payload of the EventTypePhoneHeartbeatDead event
type PhoneHeartbeatDeadPayload struct {
	PhoneID   uuid.UUID       `json:"phone_id"`
	UserID    entities.UserID `json:"user_id"`
	Timestamp time.Time       `json:"timestamp"`
	Owner     string          `json:"owner"`
}
