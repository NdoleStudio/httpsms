package services

import (
	"time"

	"github.com/google/uuid"
)

// MessageSendParams parameters for sending a new message
type MessageSendParams struct {
	From              string
	To                string
	Content           string
	Source            string
	RequestReceivedAt time.Time
}

// MessageStoreParams are parameters for creating a new message
type MessageStoreParams struct {
	From              string
	To                string
	Content           string
	ID                uuid.UUID
	Source            string
	RequestReceivedAt time.Time
}
