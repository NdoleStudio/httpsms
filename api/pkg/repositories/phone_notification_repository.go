package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
)

// PhoneNotificationRepository loads and persists an entities.PhoneNotification
type PhoneNotificationRepository interface {
	// Schedule a new entities.PhoneNotification
	Schedule(ctx context.Context, messagesPerMinute uint, notification *entities.PhoneNotification) error

	// UpdateStatus of a notification
	UpdateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) error
}
