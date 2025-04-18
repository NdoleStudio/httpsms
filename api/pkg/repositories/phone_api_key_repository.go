package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// PhoneAPIKeyRepository loads and persists an entities.PhoneAPIKey
type PhoneAPIKeyRepository interface {
	// Create a new entities.PhoneAPIKey
	Create(ctx context.Context, phone *entities.PhoneAPIKey) error

	// LoadAuthContext fetches an entities.AuthContext by apiKey
	LoadAuthContext(ctx context.Context, apiKey string) (entities.AuthContext, error)

	// Index entities.PhoneAPIKey of a user
	Index(ctx context.Context, userID entities.UserID, params IndexParams) ([]*entities.PhoneAPIKey, error)

	// Delete an entities.PhoneAPIKey
	Delete(ctx context.Context, userID entities.UserID, phoneAPIKeyID uuid.UUID) error

	// AddPhone an entities.Phone to an entities.PhoneAPIKey
	AddPhone(ctx context.Context, authContext entities.AuthContext, phone *entities.Phone) error

	// DeleteAllForUser deletes all entities.PhoneAPIKey for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
