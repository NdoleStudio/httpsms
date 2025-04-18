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

	// LoadAuthContext fetches an entities.AuthContext by apiKey
	LoadAuthContext(ctx context.Context, apiKey string) (entities.AuthContext, error)

	// Load an entities.User by entities.UserID
	Load(ctx context.Context, userID entities.UserID) (*entities.User, error)

	// RotateAPIKey updates the API Key of a user
	RotateAPIKey(ctx context.Context, userID entities.UserID) (*entities.User, error)

	// LoadOrStore an entities.User by entities.AuthContext
	LoadOrStore(ctx context.Context, user entities.AuthContext) (*entities.User, bool, error)

	// LoadBySubscriptionID loads a user based on the lemonsqueezy subscriptionID
	LoadBySubscriptionID(ctx context.Context, subscriptionID string) (*entities.User, error)

	// LoadByEmail loads a user based on the email
	LoadByEmail(ctx context.Context, email string) (*entities.User, error)

	// Delete an entities.User by entities.UserID
	Delete(ctx context.Context, user *entities.User) error
}
