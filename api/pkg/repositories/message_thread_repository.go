package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
)

// MessageThreadRepository loads and persists an entities.MessageThread
type MessageThreadRepository interface {
	// Store a new entities.MessageThread
	Store(ctx context.Context, thread *entities.MessageThread) error

	// Update a new entities.MessageThread
	Update(ctx context.Context, thread *entities.MessageThread) error

	// LoadByOwnerContact fetches a thread between owner and contact
	LoadByOwnerContact(ctx context.Context, owner string, contact string) (*entities.MessageThread, error)

	// Load a thread by ID
	Load(ctx context.Context, ID uuid.UUID) (*entities.MessageThread, error)

	// Index message threads for an owner
	Index(ctx context.Context, owner string, archived bool, params IndexParams) (*[]entities.MessageThread, error)
}
