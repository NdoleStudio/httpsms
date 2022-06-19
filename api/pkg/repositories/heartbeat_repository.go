package repositories

import (
	"context"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
)

// HeartbeatRepository loads and persists an entities.Heartbeat
type HeartbeatRepository interface {
	// Store a new entities.Heartbeat
	Store(ctx context.Context, heartbeat *entities.Heartbeat) error

	// Index entities.Heartbeat of an owner
	Index(ctx context.Context, owner string, params IndexParams) (*[]entities.Heartbeat, error)
}
