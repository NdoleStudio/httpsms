package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// DiscordRepository loads and persists an entities.Discord
type DiscordRepository interface {
	// Save Upsert a new entities.Discord
	Save(ctx context.Context, phone *entities.Discord) error

	// Index entities.Discord by entities.UserID
	Index(ctx context.Context, userID entities.UserID, params IndexParams) ([]*entities.Discord, error)

	// FetchHavingIncomingChannel loads Discords for a user that has an incoming channel ID set.
	FetchHavingIncomingChannel(ctx context.Context, userID entities.UserID) ([]*entities.Discord, error)

	// Load loads a Discord by ID.
	Load(ctx context.Context, userID entities.UserID, DiscordID uuid.UUID) (*entities.Discord, error)

	// Delete an entities.Discord
	Delete(ctx context.Context, userID entities.UserID, DiscordID uuid.UUID) error
}
