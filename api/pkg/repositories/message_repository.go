package repositories

import (
	"context"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/google/uuid"
)

// MessageRepository loads and persists an entities.Message
type MessageRepository interface {
	// Save a new entities.Message
	Save(ctx context.Context, message *entities.Message) error

	// Load an entities.Message by ID
	Load(ctx context.Context, messageID uuid.UUID) (*entities.Message, error)
}
