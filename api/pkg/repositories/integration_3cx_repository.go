package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// Integration3CxRepository loads and persists an entities.Integration3CX
type Integration3CxRepository interface {
	// Save an entities.Integration3CX
	Save(ctx context.Context, heartbeat *entities.Integration3CX) error
}
