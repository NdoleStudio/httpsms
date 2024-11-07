package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// WebhookRepository loads and persists an entities.User
type WebhookRepository interface {
	// Save Upsert a new entities.Webhook
	Save(ctx context.Context, phone *entities.Webhook) error

	// Index entities.Webhook by entities.UserID
	Index(ctx context.Context, userID entities.UserID, params IndexParams) ([]*entities.Webhook, error)

	// LoadByEvent loads webhooks for a user and event.
	LoadByEvent(ctx context.Context, userID entities.UserID, event string, phoneNumber string) ([]*entities.Webhook, error)

	// Load loads a webhook by ID.
	Load(ctx context.Context, userID entities.UserID, webhookID uuid.UUID) (*entities.Webhook, error)

	// Delete an entities.Webhook
	Delete(ctx context.Context, userID entities.UserID, webhookID uuid.UUID) error

	// DeleteAllForUser deletes all entities.Webhook for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
