package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// EventTypeMessageSendScheduleDeleted is emitted when a message send schedule is deleted
const EventTypeMessageSendScheduleDeleted = "message-send-schedule.deleted"

// MessageSendScheduleDeletedPayload is the payload of the EventTypeMessageSendScheduleDeleted event
type MessageSendScheduleDeletedPayload struct {
	ScheduleID uuid.UUID       `json:"schedule_id"`
	UserID     entities.UserID `json:"user_id"`
	Timestamp  time.Time       `json:"timestamp"`
}
