package utils

import "time"

func GetCurrentWeek(now time.Time) int {
	startDate := time.Date(2025, 7, 21, 0, 0, 0, 0, time.UTC)

	if now.Before(startDate) {
		return 1
	}

	days := int(now.Sub(startDate).Hours() / 24)
	week := (days / 7) + 1

	if week > 24 {
		week = 24
	}

	return week
}

func GetDayBounds(now time.Time) (time.Time, time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	return today, tomorrow
}

// GetMonToFriBounds returns [Monday 00:00, Saturday 00:00) for the current week in the
// provided time's location (callers should pass Central time for consistency).
func GetMonToFriBounds(now time.Time) (time.Time, time.Time) {
	// Normalize to start of day
	base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Go's Weekday: Sunday==0 ... Saturday==6
	wd := int(base.Weekday())
	// Compute offset to Monday (1). If Sunday (0), go back 6 days.
	var toMonday int
	if wd == 0 {
		toMonday = -6
	} else {
		toMonday = 1 - wd
	}
	monday := base.AddDate(0, 0, toMonday)
	saturday := monday.AddDate(0, 0, 5) // Monday + 5 days = Saturday 00:00
	return monday, saturday
}
