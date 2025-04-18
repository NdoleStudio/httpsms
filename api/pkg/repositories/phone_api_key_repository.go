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

	// Load an entities.PhoneAPIKey by userID and phoneAPIKeyID
	Load(ctx context.Context, userID entities.UserID, phoneAPIKeyID uuid.UUID) (*entities.PhoneAPIKey, error)

	// LoadAuthContext fetches an entities.AuthContext by apiKey
	LoadAuthContext(ctx context.Context, apiKey string) (entities.AuthContext, error)

	// Index entities.PhoneAPIKey of a user
	Index(ctx context.Context, userID entities.UserID, params IndexParams) ([]*entities.PhoneAPIKey, error)

	// Delete an entities.PhoneAPIKey
	Delete(ctx context.Context, phoneAPIKey *entities.PhoneAPIKey) error

	// AddPhone adds an entities.Phone to an entities.PhoneAPIKey
	AddPhone(ctx context.Context, authContext entities.AuthContext, phoneID uuid.UUID, phoneNumber string) error

	// RemovePhone removes an entities.Phone to an entities.PhoneAPIKey
	RemovePhone(ctx context.Context, phoneAPIKey *entities.PhoneAPIKey, phone *entities.Phone) error

	// DeleteAllForUser deletes all entities.PhoneAPIKey for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
