package entities

import "github.com/google/uuid"

// MessageThreadUnreadItem records an inbound item currently counted as unread.
type MessageThreadUnreadItem struct {
	MessageID       uuid.UUID     `gorm:"primaryKey;type:uuid"`
	MessageThreadID uuid.UUID     `gorm:"not null;type:uuid;index"`
	MessageThread   MessageThread `gorm:"constraint:OnDelete:CASCADE;"`
}
