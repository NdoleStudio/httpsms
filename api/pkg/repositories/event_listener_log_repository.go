package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// EventListenerLogRepository loads and persists an entities.EventListenerLog
type EventListenerLogRepository interface {
	// Store a new entities.EventListenerLog
	Store(ctx context.Context, log *entities.EventListenerLog) error

	// Has verifies that the listener has not already been called
	Has(ctx context.Context, eventID string, handler string) (bool, error)
}
