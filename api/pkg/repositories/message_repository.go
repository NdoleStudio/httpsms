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

	// GetOutstanding fetches list of outstanding []entities.Message
	GetOutstanding(ctx context.Context, take int) (*[]entities.Message, error)
}
