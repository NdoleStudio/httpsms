package repositories

import (
	"context"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/google/uuid"
)

// MessageRepository loads and persists an entities.Message
type MessageRepository interface {
	// Store a new entities.Message
	Store(ctx context.Context, message *entities.Message) error

	// Update a new entities.Message
	Update(ctx context.Context, message *entities.Message) error

	// Load an entities.Message by ID
	Load(ctx context.Context, messageID uuid.UUID) (*entities.Message, error)

	// Index entities.Message between 2 phone numbers
	Index(ctx context.Context, owner string, contact string, params IndexParams) (*[]entities.Message, error)

	// GetOutstanding fetches list of outstanding []entities.Message
	GetOutstanding(ctx context.Context, limit int) (*[]entities.Message, error)
}
