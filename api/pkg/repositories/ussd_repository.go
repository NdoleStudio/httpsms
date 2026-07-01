package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// USSDRepository manages persistence of USSD sessions
type USSDRepository interface {
	// Store saves a new USSD session
	Store(ctx context.Context, ussd *entities.USSD) error

	// Update updates an existing USSD session
	Update(ctx context.Context, ussd *entities.USSD) error

	// Load loads a USSD session by ID and user ID
	Load(ctx context.Context, userID entities.UserID, ussdID uuid.UUID) (*entities.USSD, error)

	// LoadBySessionID loads a USSD session by session ID and user ID
	LoadBySessionID(ctx context.Context, userID entities.UserID, sessionID string) (*entities.USSD, error)

	// Index fetches paginated USSD sessions for a user
	Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.USSD, error)

	// IndexByPhoneID fetches paginated USSD sessions for a phone
	IndexByPhoneID(ctx context.Context, userID entities.UserID, phoneID uuid.UUID, params IndexParams) (*[]entities.USSD, error)

	// Delete deletes a USSD session by ID and user ID
	Delete(ctx context.Context, userID entities.UserID, ussdID uuid.UUID) error

	// DeleteAllForPhone deletes all USSD sessions for a phone
	DeleteAllForPhone(ctx context.Context, phoneID uuid.UUID) error
}