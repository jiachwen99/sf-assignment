package domain

import "time"

type RecurUnit string

const (
	Daily   RecurUnit = "day"
	Weekly  RecurUnit = "week"
	Monthly RecurUnit = "month"
)

func (u RecurUnit) Valid() bool {
	switch u {
	case Daily, Weekly, Monthly:
		return true
	}
	return false
}

// NextDue returns the first occurrence after now, counted from anchor.
//
// Stepping runs from the anchor rather than from the last occurrence: 28
// February cannot tell you whether it meant the 28th or a clamped 31st. Missed
// periods are skipped rather than generated, so a task left for a month spawns
// one occurrence and not thirty.
func NextDue(anchor *time.Time, unit RecurUnit, interval int, now time.Time) (*time.Time, error) {
	if !unit.Valid() {
		return nil, Invalid("recurUnit", "Repeat must be by day, week or month")
	}
	// Guards the walk below, which would never terminate on an interval of
	// zero because every step would return the anchor.
	if interval < 1 {
		return nil, Invalid("recurInterval", "Repeat interval must be at least 1")
	}
	if anchor == nil {
		return nil, nil
	}

	// Jump close, then walk. The jump keeps a task years overdue from looping
	// thousands of times; the walk handles months the jump cannot count exactly.
	k := max(periodsBetween(*anchor, now, unit, interval), 0)
	next := step(*anchor, unit, interval, k)
	for !next.After(now) {
		k++
		next = step(*anchor, unit, interval, k)
	}
	return &next, nil
}

func periodsBetween(anchor, now time.Time, unit RecurUnit, interval int) int {
	switch unit {
	case Daily:
		return int(now.Sub(anchor).Hours()) / (24 * interval)
	case Weekly:
		return int(now.Sub(anchor).Hours()) / (24 * 7 * interval)
	case Monthly:
		months := (now.Year()-anchor.Year())*12 + int(now.Month()-anchor.Month())
		return months / interval
	}
	return 0
}

// step advances the anchor by n whole periods. Months compute the target year
// and month and then clamp the day, because time.AddDate normalises an overflow
// instead: 31 January plus one month gives 3 March. Clamping from the anchor
// each time is what lets a later month return to the 31st.
func step(anchor time.Time, unit RecurUnit, interval, n int) time.Time {
	switch unit {
	case Daily:
		return anchor.AddDate(0, 0, interval*n)
	case Weekly:
		return anchor.AddDate(0, 0, 7*interval*n)
	case Monthly:
		total := int(anchor.Month()) - 1 + interval*n
		year := anchor.Year() + total/12
		month := time.Month(total%12 + 1)

		day := anchor.Day()
		if last := daysIn(year, month); day > last {
			day = last
		}
		return time.Date(year, month, day,
			anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), anchor.Location())
	}
	return anchor
}

// Day zero of the following month is the last day of this one.
func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
