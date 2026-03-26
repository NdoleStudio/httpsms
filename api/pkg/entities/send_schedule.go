package entities

import (
	"time"

	"github.com/google/uuid"
)

// SendSchedule defines weekly sending availability for a user.
type SendSchedule struct {
	ID        uuid.UUID            `json:"id" gorm:"primaryKey;type:uuid;"`
	UserID    UserID               `json:"user_id" gorm:"index:idx_send_schedules_user_id;not null"`
	Name      string               `json:"name"`
	Timezone  string               `json:"timezone"`
	IsDefault bool                 `json:"is_default" gorm:"default:false"`
	IsActive  bool                 `json:"is_active" gorm:"default:true"`
	Windows   []SendScheduleWindow `json:"windows" gorm:"constraint:OnDelete:CASCADE;foreignKey:ScheduleID"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// SendScheduleWindow is a recurring weekly time window.
type SendScheduleWindow struct {
	ID          uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;"`
	ScheduleID  uuid.UUID `json:"schedule_id" gorm:"index:idx_send_schedule_windows_schedule_id;not null"`
	DayOfWeek   int       `json:"day_of_week"`
	StartMinute int       `json:"start_minute"`
	EndMinute   int       `json:"end_minute"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
