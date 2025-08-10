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
