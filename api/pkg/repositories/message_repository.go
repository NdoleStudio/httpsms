package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// MessageRepository loads and persists an entities.Message
type MessageRepository interface {
	// Store a new entities.Message
	Store(ctx context.Context, message *entities.Message) error

	// Update a new entities.Message
	Update(ctx context.Context, message *entities.Message) error

	// Load an entities.Message by ID
	Load(ctx context.Context, userID entities.UserID, messageID uuid.UUID) (*entities.Message, error)

	// Index entities.Message between 2 phone numbers
	Index(ctx context.Context, userID entities.UserID, owner string, contact string, params IndexParams) (*[]entities.Message, error)

	// LastMessage fetches the last message between an owner and a contact
	LastMessage(ctx context.Context, userID entities.UserID, owner string, contact string) (*entities.Message, error)

	// Search entities.Message for a user
	Search(ctx context.Context, userID entities.UserID, owners []string, types []entities.MessageType, statuses []entities.MessageStatus, params IndexParams) ([]*entities.Message, error)

	// GetOutstanding fetches an entities.Message which is outstanding
	GetOutstanding(ctx context.Context, userID entities.UserID, messageID uuid.UUID) (*entities.Message, error)

	// Delete an entities.Message by ID
	Delete(ctx context.Context, userID entities.UserID, messageID uuid.UUID) error

	// DeleteByOwnerAndContact deletes messages between an owner and a contact
	DeleteByOwnerAndContact(ctx context.Context, userID entities.UserID, owner string, contact string) error
}
