package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// PhoneNotificationRepository loads and persists an entities.PhoneNotification
type PhoneNotificationRepository interface {
	// Schedule a new entities.PhoneNotification
	Schedule(ctx context.Context, messagesPerMinute uint, schedule *entities.MessageSendSchedule, notification *entities.PhoneNotification) error

	// ScheduleExact stores a phone notification with a fixed ScheduledAt time,
	// bypassing rate-limit and schedule window logic.
	ScheduleExact(ctx context.Context, notification *entities.PhoneNotification) error

	// UpdateStatus of a notification
	UpdateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) error

	// DeleteAllForUser deletes all entities.PhoneNotification for a user
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error

	// DeleteByMessageID deletes entities.PhoneNotification for a message and user
	DeleteByMessageID(ctx context.Context, userID entities.UserID, messageID uuid.UUID) error
}
