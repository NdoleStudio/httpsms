package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// HeartbeatMonitorRepository loads and persists an entities.HeartbeatMonitor
type HeartbeatMonitorRepository interface {
	// Store a new entities.HeartbeatMonitor
	Store(ctx context.Context, heartbeat *entities.HeartbeatMonitor) error

	// Load a phone by user and phone number
	Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.HeartbeatMonitor, error)

	// Exists checks if a heartbeat monitor exists for a phone number
	Exists(ctx context.Context, userID entities.UserID, monitorID uuid.UUID) (bool, error)

	// UpdateQueueID updates the queueID of a monitor
	UpdateQueueID(ctx context.Context, monitorID uuid.UUID, queueID string) error

	// Delete an entities.HeartbeatMonitor
	Delete(ctx context.Context, userID entities.UserID, phoneNumber string) error

	// UpdatePhoneOnline updates the phone online status of a monitor
	UpdatePhoneOnline(ctx context.Context, userID entities.UserID, monitorID uuid.UUID, online bool) error

	// DeleteAllForUser deletes all entities.HeartbeatMonitor for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
