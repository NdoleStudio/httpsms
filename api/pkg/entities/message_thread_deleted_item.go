package entities

import "github.com/google/uuid"

// MessageThreadDeletedItem records a permanently deleted message activity.
type MessageThreadDeletedItem struct {
	MessageID uuid.UUID `gorm:"primaryKey;type:uuid"`
}
