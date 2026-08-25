package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// ContactRepository loads and persists an entities.Contact.
type ContactRepository interface {
	// Store one or many new entities.Contact.
	Store(ctx context.Context, contacts []*entities.Contact) error

	// Update an existing entities.Contact.
	Update(ctx context.Context, contact *entities.Contact) error

	// Load a contact by ID for a user.
	Load(ctx context.Context, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error)

	// Index contacts for a user with optional search.
	Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Contact, error)

	// Count returns the number of contacts for a user matching the same
	// name/emails/phone_numbers filter as Index, ignoring pagination.
	Count(ctx context.Context, userID entities.UserID, params IndexParams) (int64, error)

	// FetchByPhoneNumbers returns contacts containing at least one requested
	// phone number, ordered by updated_at ascending.
	FetchByPhoneNumbers(ctx context.Context, userID entities.UserID, phoneNumbers []string) (*[]entities.Contact, error)

	// Delete a contact by ID for a user.
	Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error

	// DeleteAllForUser deletes all contacts for a user.
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
