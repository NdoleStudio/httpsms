package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// MessageSendScheduleRepository loads and persists entities.MessageSendSchedule.
type MessageSendScheduleRepository interface {
	// Store persists a new message send schedule.
	Store(ctx context.Context, schedule *entities.MessageSendSchedule) error

	// Update persists changes to an existing message send schedule.
	Update(ctx context.Context, schedule *entities.MessageSendSchedule) error

	// Load returns a message send schedule by user ID and schedule ID.
	Load(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) (*entities.MessageSendSchedule, error)

	// Index returns all message send schedules owned by a user.
	Index(ctx context.Context, userID entities.UserID) ([]entities.MessageSendSchedule, error)

	// Delete removes a message send schedule owned by a user.
	Delete(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) error

	// DeleteAllForUser removes all message send schedules owned by a user.
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error

	// CountByUser returns the number of schedules owned by a user.
	CountByUser(ctx context.Context, userID entities.UserID) (int, error)
}
