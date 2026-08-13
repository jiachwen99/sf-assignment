package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func at(y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, time.UTC)
}

func TestNextDue(t *testing.T) {
	cases := []struct {
		name     string
		anchor   time.Time
		unit     RecurUnit
		interval int
		now      time.Time
		want     time.Time
	}{
		{
			name:   "daily steps to tomorrow",
			anchor: at(2026, time.March, 2, 9), unit: Daily, interval: 1,
			now:  at(2026, time.March, 2, 9),
			want: at(2026, time.March, 3, 9),
		},
		{
			name:   "an interval above one counts whole periods",
			anchor: at(2026, time.March, 2, 9), unit: Daily, interval: 3,
			now:  at(2026, time.March, 2, 9),
			want: at(2026, time.March, 5, 9),
		},
		{
			// A task left for a quarter spawns one occurrence, not ninety.
			name:   "months of missed dailies collapse to the next one",
			anchor: at(2026, time.January, 5, 8), unit: Daily, interval: 1,
			now:  at(2026, time.April, 7, 15),
			want: at(2026, time.April, 8, 8),
		},
		{
			name:   "weekly lands on the same weekday",
			anchor: at(2026, time.March, 2, 9), unit: Weekly, interval: 1,
			now:  at(2026, time.March, 2, 9),
			want: at(2026, time.March, 9, 9),
		},
		{
			name:   "fortnightly is weekly with an interval of two",
			anchor: at(2026, time.March, 2, 9), unit: Weekly, interval: 2,
			now:  at(2026, time.March, 10, 9),
			want: at(2026, time.March, 16, 9),
		},
		{
			// The case the clamp exists for. AddDate would say 3 March.
			name:   "the 31st clamps to the end of a short month",
			anchor: at(2026, time.January, 31, 9), unit: Monthly, interval: 1,
			now:  at(2026, time.January, 31, 9),
			want: at(2026, time.February, 28, 9),
		},
		{
			// And the reason the anchor is stored rather than the last
			// occurrence: 28 February alone cannot say whether it meant the
			// 28th or a clamped 31st.
			name:   "and returns to the 31st the month after",
			anchor: at(2026, time.January, 31, 9), unit: Monthly, interval: 1,
			now:  at(2026, time.February, 28, 9),
			want: at(2026, time.March, 31, 9),
		},
		{
			name:   "february gets its extra day in a leap year",
			anchor: at(2028, time.January, 31, 9), unit: Monthly, interval: 1,
			now:  at(2028, time.January, 31, 9),
			want: at(2028, time.February, 29, 9),
		},
		{
			name:   "a day that fits is left alone",
			anchor: at(2026, time.January, 30, 9), unit: Monthly, interval: 1,
			now:  at(2026, time.February, 28, 9),
			want: at(2026, time.March, 30, 9),
		},
		{
			name:   "quarterly crosses the year boundary",
			anchor: at(2026, time.November, 15, 9), unit: Monthly, interval: 3,
			now:  at(2026, time.November, 15, 9),
			want: at(2027, time.February, 15, 9),
		},
		{
			// Three months late on a monthly task. The skipped occurrences are
			// not generated, and the answer is still anchored to the 31st.
			name:   "months missed on a monthly task collapse too",
			anchor: at(2026, time.January, 31, 9), unit: Monthly, interval: 1,
			now:  at(2026, time.April, 20, 9),
			want: at(2026, time.April, 30, 9),
		},
		{
			name:   "the time of day is carried",
			anchor: at(2026, time.March, 2, 17), unit: Monthly, interval: 1,
			now:  at(2026, time.March, 2, 17),
			want: at(2026, time.April, 2, 17),
		},
		{
			// Strictly after. An occurrence due at this instant is this one,
			// not the next.
			name:   "a due date on the boundary is not the answer",
			anchor: at(2026, time.March, 2, 9), unit: Daily, interval: 1,
			now:  at(2026, time.March, 3, 9),
			want: at(2026, time.March, 4, 9),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NextDue(&c.anchor, c.unit, c.interval, c.now)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, c.want, got.UTC())
		})
	}
}

// A task can recur without ever having had a date, and that is not an error.
func TestNextDueWithoutAnAnchorIsNothing(t *testing.T) {
	got, err := NextDue(nil, Daily, 1, at(2026, time.March, 2, 9))
	require.NoError(t, err)
	require.Nil(t, got)
}

// Not defensive noise: an interval below one makes every step return the anchor
// and the search for a later date never terminates.
func TestNextDueRejectsWhatWouldNotTerminate(t *testing.T) {
	anchor := at(2026, time.March, 2, 9)
	now := at(2026, time.March, 2, 9)

	_, err := NextDue(&anchor, Daily, 0, now)
	require.Error(t, err)

	_, err = NextDue(&anchor, RecurUnit("fortnight"), 1, now)
	require.Error(t, err)
}

// Kept so the reason for the clamp stays visible. This is what the arithmetic
// would do on its own: 31 January plus one month is 31 February, which
// normalises forward into March.
func TestAddDateOverflowsWhichIsWhyStepClamps(t *testing.T) {
	naive := at(2026, time.January, 31, 9).AddDate(0, 1, 0)
	require.Equal(t, at(2026, time.March, 3, 9), naive)

	anchor := at(2026, time.January, 31, 9)
	clamped, err := NextDue(&anchor, Monthly, 1, anchor)
	require.NoError(t, err)
	require.Equal(t, at(2026, time.February, 28, 9), clamped.UTC())
}
