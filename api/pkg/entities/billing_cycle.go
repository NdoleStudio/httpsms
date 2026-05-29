package entities

import "time"

// ComputeBillingCycle returns the start and end timestamps of the billing cycle
// that contains `now`, given the user's anchor day (1–31). The anchor day is
// dynamically clamped to the number of days in the relevant month.
func ComputeBillingCycle(now time.Time, anchorDay int) (start, end time.Time) {
	clampedDay := min(anchorDay, daysInMonth(now.Year(), now.Month()))

	if now.Day() >= clampedDay {
		// Cycle started this month
		start = time.Date(now.Year(), now.Month(), clampedDay, 0, 0, 0, 0, time.UTC)
	} else {
		// Cycle started last month
		prev := now.AddDate(0, -1, 0)
		prevClamped := min(anchorDay, daysInMonth(prev.Year(), prev.Month()))
		start = time.Date(prev.Year(), prev.Month(), prevClamped, 0, 0, 0, 0, time.UTC)
	}

	// Compute next cycle start by moving to next month and clamping the day
	nextMonth := start.Month() + 1
	nextYear := start.Year()
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	nextClamped := min(anchorDay, daysInMonth(nextYear, nextMonth))
	nextCycleStart := time.Date(nextYear, nextMonth, nextClamped, 0, 0, 0, 0, time.UTC)

	// End = one second before the next cycle start
	end = nextCycleStart.Add(-time.Second)

	return start, end
}

// daysInMonth returns the number of days in the given month/year.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
