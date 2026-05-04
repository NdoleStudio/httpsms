package entities

import (
	"time"

	"github.com/google/uuid"
)

// MessageSendScheduleWindow represents a single availability window for a day of the week.
type MessageSendScheduleWindow struct {
	DayOfWeek   int `json:"day_of_week" example:"1"`
	StartMinute int `json:"start_minute" example:"540"`
	EndMinute   int `json:"end_minute" example:"1020"`
}

// MessageSendSchedule controls when a phone is allowed to send outgoing SMS messages.
type MessageSendSchedule struct {
	ID        uuid.UUID                   `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID    UserID                      `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Name      string                      `json:"name" example:"Business Hours"`
	Timezone  string                      `json:"timezone" example:"Europe/Tallinn"`
	Windows   []MessageSendScheduleWindow `json:"windows" gorm:"type:jsonb;serializer:json"`
	CreatedAt time.Time                   `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt time.Time                   `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}

// ResolveScheduledAt returns the next allowed send time based on the schedule.
// If the schedule is inactive, has no windows, or has an invalid timezone,
// the current time is returned in UTC. An active schedule with no windows
// is treated as inactive (messages are sent immediately).
func (schedule *MessageSendSchedule) ResolveScheduledAt(current time.Time) time.Time {
	if schedule == nil || len(schedule.Windows) == 0 {
		return current.UTC()
	}

	location, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return current.UTC()
	}

	base := current.In(location)
	var best time.Time

	for dayOffset := 0; dayOffset <= 7; dayOffset++ {
		day := base.AddDate(0, 0, dayOffset)
		weekday := int(day.Weekday())

		for _, window := range schedule.Windows {
			if window.DayOfWeek != weekday {
				continue
			}

			start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, location).
				Add(time.Duration(window.StartMinute) * time.Minute)

			end := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, location).
				Add(time.Duration(window.EndMinute) * time.Minute)

			var candidate time.Time

			switch {
			case dayOffset == 0 && base.Before(start):
				candidate = start
			case dayOffset == 0 && (base.Equal(start) || (base.After(start) && base.Before(end))):
				candidate = base
			case dayOffset > 0:
				candidate = start
			default:
				continue
			}

			if best.IsZero() || candidate.Before(best) {
				best = candidate
			}
		}

		if !best.IsZero() {
			break
		}
	}

	if best.IsZero() {
		return current.UTC()
	}

	return best.UTC()
}
