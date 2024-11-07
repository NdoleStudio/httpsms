package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// PhoneNotificationRepository loads and persists an entities.PhoneNotification
type PhoneNotificationRepository interface {
	// Schedule a new entities.PhoneNotification
	Schedule(ctx context.Context, messagesPerMinute uint, notification *entities.PhoneNotification) error

	// UpdateStatus of a notification
	UpdateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) error

	// DeleteAllForUser deletes all entities.PhoneNotification for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}
