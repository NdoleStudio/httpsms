package repositories

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
)

// SendScheduleRepository loads and persists send schedules.
type SendScheduleRepository interface {
	Store(ctx context.Context, schedule *entities.SendSchedule) error
	Update(ctx context.Context, schedule *entities.SendSchedule) error
	Load(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) (*entities.SendSchedule, error)
	Index(ctx context.Context, userID entities.UserID) ([]*entities.SendSchedule, error)
	Delete(ctx context.Context, userID entities.UserID, scheduleID uuid.UUID) error
	Default(ctx context.Context, userID entities.UserID) (*entities.SendSchedule, error)
	ClearDefault(ctx context.Context, userID entities.UserID) error
}
