package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

type MessageThreadActivityUpdate struct {
	MessageThreadID uuid.UUID
	UserID          entities.UserID
	Timestamp       time.Time
	MessageID       uuid.UUID
	Content         string
	Status          entities.MessageStatus
	MarksUnread     bool
	EventTimestamp  time.Time
	Unarchive       bool
}

type MessageThreadStatusUpdate struct {
	IsArchived *bool
	IsRead     *bool
	ReadAt     time.Time
}

type MessageThreadDeletedUpdate struct {
	MessageThreadID    uuid.UUID
	UserID             entities.UserID
	LastMessageID      *uuid.UUID
	LastMessageContent *string
	LastMessageStatus  entities.MessageStatus
}

// MessageThreadRepository loads and persists an entities.MessageThread
type MessageThreadRepository interface {
	// Store a new entities.MessageThread
	Store(ctx context.Context, thread *entities.MessageThread) error

	// UpdateActivity persists the last-message activity fields for a thread
	UpdateActivity(ctx context.Context, params MessageThreadActivityUpdate) error

	// UpdateStatus persists archive/read status fields for a thread
	UpdateStatus(ctx context.Context, userID entities.UserID, messageThreadID uuid.UUID, params MessageThreadStatusUpdate) (*entities.MessageThread, error)

	// LoadByOwnerContact fetches a thread between owner and contact
	LoadByOwnerContact(ctx context.Context, userID entities.UserID, owner string, contact string) (*entities.MessageThread, error)

	// Load a thread by ID
	Load(ctx context.Context, userID entities.UserID, ID uuid.UUID) (*entities.MessageThread, error)

	// Index message threads for an owner
	Index(ctx context.Context, userID entities.UserID, owner string, archived bool, params IndexParams) (*[]entities.MessageThread, error)

	// UpdateAfterDeletedMessage updates a thread after the original message has been deleted
	UpdateAfterDeletedMessage(ctx context.Context, params MessageThreadDeletedUpdate) error

	// Delete an entities.MessageThread by ID
	Delete(ctx context.Context, userID entities.UserID, messageThreadID uuid.UUID) error

	// DeleteAllForUser deletes all entities.MessageThread for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
