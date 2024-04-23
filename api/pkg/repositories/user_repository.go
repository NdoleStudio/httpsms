package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserRepository loads and persists an entities.User
type UserRepository interface {
	// Store a new entities.User
	Store(ctx context.Context, user *entities.User) error

	// Update a new entities.User
	Update(ctx context.Context, user *entities.User) error

	// LoadAuthUser fetches an entities.AuthUser by apiKey
	LoadAuthUser(ctx context.Context, apiKey string) (entities.AuthUser, error)

	// Load an entities.User by entities.UserID
	Load(ctx context.Context, userID entities.UserID) (*entities.User, error)

	// RotateAPIKey updates the API Key of a user
	RotateAPIKey(ctx context.Context, userID entities.UserID) (*entities.User, error)

	// LoadOrStore an entities.User by entities.AuthUser
	LoadOrStore(ctx context.Context, user entities.AuthUser) (*entities.User, bool, error)

	// LoadBySubscriptionID loads a user based on the lemonsqueezy subscriptionID
	LoadBySubscriptionID(ctx context.Context, subscriptionID string) (*entities.User, error)
}
