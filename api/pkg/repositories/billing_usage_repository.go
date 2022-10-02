package repositories

import (
	"context"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// BillingUsageRepository loads and persists an entities.BillingUsage
type BillingUsageRepository interface {
	// RegisterSentMessage registers a message as sent
	RegisterSentMessage(ctx context.Context, timestamp time.Time, user entities.UserID) error

	// RegisterReceivedMessage registers a message as received
	RegisterReceivedMessage(ctx context.Context, timestamp time.Time, user entities.UserID) error

	// GetCurrent returns the current billing usage by entities.UserID
	GetCurrent(ctx context.Context, userID entities.UserID) (*entities.BillingUsage, error)

	// GetHistory returns past billing usage by entities.UserID
	GetHistory(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.BillingUsage, error)
}
