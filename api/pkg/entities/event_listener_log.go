package entities

import (
	"time"

	"github.com/google/uuid"
)

// EventListenerLog stores the log of all the events handled
type EventListenerLog struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;"`
	EventType string    `json:"event_type"`
	Handler   string    `json:"handler"`
	HandledAt time.Time `json:"handled_at"`
	CreatedAt time.Time `json:"created_at"`
}
