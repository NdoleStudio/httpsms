package entities

import (
	"time"

	"github.com/google/uuid"
)

// SendScheduleWindow represents a single availability window for a day of the week.
type SendScheduleWindow struct {
	DayOfWeek   int `json:"day_of_week" example:"1"`
	StartMinute int `json:"start_minute" example:"540"`
	EndMinute   int `json:"end_minute" example:"1020"`
}

// SendSchedule controls when a phone is allowed to send outgoing SMS messages.
type SendSchedule struct {
	ID        uuid.UUID            `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID    UserID               `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Name      string               `json:"name" example:"Business Hours"`
	Timezone  string               `json:"timezone" example:"Europe/Tallinn"`
	IsActive  bool                 `json:"is_active" gorm:"default:true" example:"true"`
	Windows   []SendScheduleWindow `json:"windows" gorm:"type:jsonb;serializer:json"`
	CreatedAt time.Time            `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt time.Time            `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}
