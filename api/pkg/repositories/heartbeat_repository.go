package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// HeartbeatRepository loads and persists an entities.Heartbeat
type HeartbeatRepository interface {
	// Store a new entities.Heartbeat
	Store(ctx context.Context, heartbeat *entities.Heartbeat) error

	// Index entities.Heartbeat of an owner
	Index(ctx context.Context, userID entities.UserID, owner string, params IndexParams) (*[]entities.Heartbeat, error)

	// Last entities.Heartbeat returns the last heartbeat
	Last(ctx context.Context, userID entities.UserID, owner string) (*entities.Heartbeat, error)
}
