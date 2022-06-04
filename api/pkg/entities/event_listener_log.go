package entities

import (
	"time"

	"github.com/google/uuid"
)

// EventListenerLog stores the log of all the events handled
type EventListenerLog struct {
	ID        uuid.UUID     `json:"id" gorm:"primaryKey;type:uuid;"`
	EventID   string        `json:"event_id"`
	EventType string        `json:"event_type"`
	Handler   string        `json:"handler"`
	Duration  time.Duration `json:"duration"`
	HandledAt time.Time     `json:"handled_at"`
	CreatedAt time.Time     `json:"created_at"`
}
