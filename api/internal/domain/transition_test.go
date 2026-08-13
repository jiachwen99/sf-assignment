package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		name      string
		from, to  Status
		unmetDeps int
		allowed   bool
	}{
		{name: "an unblocked task can start", from: NotStarted, to: InProgress, allowed: true},
		{name: "an unblocked task can finish", from: InProgress, to: Completed, allowed: true},

		{
			name: "a blocked task cannot start",
			from: NotStarted, to: InProgress, unmetDeps: 1, allowed: false,
		},
		{
			// The gate the brief does not name. Without it, choosing the
			// further destination walks around the one it does.
			name: "and cannot skip to finished either",
			from: NotStarted, to: Completed, unmetDeps: 1, allowed: false,
		},
		{
			name: "but can be shelved, which is what you do with a task you cannot start",
			from: NotStarted, to: Archived, unmetDeps: 2, allowed: true,
		},
		{
			name: "and can be put back to not started",
			from: InProgress, to: NotStarted, unmetDeps: 1, allowed: true,
		},
		{
			name: "staying where it is is not a transition",
			from: InProgress, to: InProgress, unmetDeps: 3, allowed: true,
		},

		{
			name: "archived comes back to not started",
			from: Archived, to: NotStarted, allowed: true,
		},
		{
			name: "not straight to in progress",
			from: Archived, to: InProgress, allowed: false,
		},
		{
			name: "and not straight to completed",
			from: Archived, to: Completed, allowed: false,
		},
		{
			name: "an archived task can be left archived",
			from: Archived, to: Archived, allowed: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := CanTransition(c.from, c.to, c.unmetDeps, nil, 1)
			if c.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// The refusal has to say what to go and finish. A task that is simply "blocked"
// leaves the reader to work out by what.
func TestBlockingNamesTheBlockers(t *testing.T) {
	blockers := []Blocker{
		{ID: 2, Name: "book the room", Status: NotStarted},
		{ID: 3, Name: "confirm the agenda", Status: InProgress},
	}

	err := CanTransition(NotStarted, InProgress, 2, blockers, 1)

	var blocked *BlockedError
	require.ErrorAs(t, err, &blocked)
	require.Equal(t, int64(1), blocked.Target)
	require.Equal(t, "blocked by book the room, confirm the agenda", blocked.Error())
}

func TestAnUnknownStatusIsRejected(t *testing.T) {
	err := CanTransition(NotStarted, Status("nearly"), 0, nil, 1)

	var invalid *ValidationError
	require.ErrorAs(t, err, &invalid)
	require.Contains(t, invalid.Fields["status"], "must be one of")
}
