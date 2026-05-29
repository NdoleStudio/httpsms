package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeBillingCycle(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		anchorDay int
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "anchor day 1 (same as calendar month)",
			now:       time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
			anchorDay: 1,
			wantStart: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "anchor day 15, now is after anchor",
			now:       time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
			anchorDay: 15,
			wantStart: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 6, 14, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "anchor day 15, now is before anchor",
			now:       time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC),
			anchorDay: 15,
			wantStart: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 5, 14, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "anchor day 15, now is exactly on anchor",
			now:       time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			anchorDay: 15,
			wantStart: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 6, 14, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "anchor day 31 in February (clamped to 28)",
			now:       time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
			anchorDay: 31,
			wantStart: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 2, 27, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "anchor day 31 in March (not clamped)",
			now:       time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
			anchorDay: 31,
			wantStart: time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 4, 29, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "anchor day 29 in February leap year",
			now:       time.Date(2024, 2, 29, 10, 0, 0, 0, time.UTC),
			anchorDay: 29,
			wantStart: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 3, 28, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "anchor day 29 in February non-leap year (clamped to 28)",
			now:       time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC),
			anchorDay: 29,
			wantStart: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 3, 28, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "year boundary: anchor day 20, now is Jan 5",
			now:       time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC),
			anchorDay: 20,
			wantStart: time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 1, 19, 23, 59, 59, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := ComputeBillingCycle(tt.now, tt.anchorDay)
			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.wantEnd, end)
		})
	}
}

func TestDaysInMonth(t *testing.T) {
	assert.Equal(t, 31, daysInMonth(2026, time.January))
	assert.Equal(t, 28, daysInMonth(2026, time.February))
	assert.Equal(t, 29, daysInMonth(2024, time.February))
	assert.Equal(t, 30, daysInMonth(2026, time.April))
	assert.Equal(t, 31, daysInMonth(2026, time.December))
}
