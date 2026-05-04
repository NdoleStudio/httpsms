package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveScheduledAt_NilSchedule_ReturnsCurrentUTC(t *testing.T) {
	now := time.Now()
	var schedule *MessageSendSchedule
	result := schedule.ResolveScheduledAt(now)
	assert.Equal(t, now.UTC(), result)
}

func TestResolveScheduledAt_InactiveSchedule_ReturnsCurrentUTC(t *testing.T) {
	now := time.Now()
	schedule := &MessageSendSchedule{}
	result := schedule.ResolveScheduledAt(now)
	assert.Equal(t, now.UTC(), result)
}

func TestResolveScheduledAt_NoWindows_ReturnsCurrentUTC(t *testing.T) {
	now := time.Now()
	schedule := &MessageSendSchedule{
		Timezone: "UTC",
		Windows:  []MessageSendScheduleWindow{},
	}
	result := schedule.ResolveScheduledAt(now)
	assert.Equal(t, now.UTC(), result)
}

func TestResolveScheduledAt_WithinWindow_ReturnsCurrentUTC(t *testing.T) {
	// Wednesday at 10:00 UTC, window is Wed 9:00-17:00 (540-1020 minutes)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC) // Wednesday
	schedule := &MessageSendSchedule{
		Timezone: "UTC",
		Windows: []MessageSendScheduleWindow{
			{DayOfWeek: int(now.Weekday()), StartMinute: 540, EndMinute: 1020},
		},
	}
	result := schedule.ResolveScheduledAt(now)
	assert.Equal(t, now.UTC(), result)
}

func TestResolveScheduledAt_BeforeWindow_ReturnsWindowStart(t *testing.T) {
	// Wednesday at 7:00 UTC, window is Wed 9:00-17:00
	now := time.Date(2025, 1, 1, 7, 0, 0, 0, time.UTC) // Wednesday
	schedule := &MessageSendSchedule{
		Timezone: "UTC",
		Windows: []MessageSendScheduleWindow{
			{DayOfWeek: int(now.Weekday()), StartMinute: 540, EndMinute: 1020},
		},
	}
	result := schedule.ResolveScheduledAt(now)
	expected := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}
