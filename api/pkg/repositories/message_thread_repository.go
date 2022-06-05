package repositories

import (
	"context"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
)

// MessageThreadRepository loads and persists an entities.MessageThread
type MessageThreadRepository interface {
	// Store a new entities.MessageThread
	Store(ctx context.Context, thread *entities.MessageThread) error

	// Update a new entities.MessageThread
	Update(ctx context.Context, thread *entities.MessageThread) error

	// Load a thread between 2 users
	Load(ctx context.Context, owner string, contact string) (*entities.MessageThread, error)

	// Index message threads for an owner
	Index(ctx context.Context, owner string, params IndexParams) (*[]entities.MessageThread, error)
}
