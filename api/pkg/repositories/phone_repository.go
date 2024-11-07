package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// PhoneRepository loads and persists an entities.Phone
type PhoneRepository interface {
	// Save Upsert a new entities.Phone
	Save(ctx context.Context, phone *entities.Phone) error

	// Index entities.Phone of a user
	Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Phone, error)

	// Load a phone by user and phone number
	Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.Phone, error)

	// LoadByID a phone by ID
	LoadByID(ctx context.Context, userID entities.UserID, phoneID uuid.UUID) (*entities.Phone, error)

	// Delete an entities.Phone
	Delete(ctx context.Context, userID entities.UserID, phoneID uuid.UUID) error

	// DeleteAllForUser deletes all entities.Phone for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
