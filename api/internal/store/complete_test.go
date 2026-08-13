package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

func newRecurring(t *testing.T, s *Store, name string, due time.Time, unit domain.RecurUnit, every int) domain.Todo {
	t.Helper()
	todo, err := s.CreateTodo(context.Background(), NewTodo{
		Name: name, Status: domain.NotStarted, Priority: domain.Medium,
		DueDate: &due, RecurUnit: &unit, RecurEvery: &every, RecurAnchor: &due,
	})
	require.NoError(t, err)
	return todo
}

func TestCompletingARecurringTaskOpensTheNextOne(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	due := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	made := newRecurring(t, s, "water the plants", due, domain.Weekly, 1)

	res, err := s.Complete(ctx, made.ID, made.Version, due)
	require.NoError(t, err)
	require.Equal(t, domain.Completed, res.Completed.Status)
	require.NotNil(t, res.Spawned)
	require.Equal(t, domain.NotStarted, res.Spawned.Status)
	require.Equal(t, due.AddDate(0, 0, 7), res.Spawned.DueDate.UTC())
	require.Equal(t, "water the plants", res.Spawned.Name)
}

// One live occurrence per series is the invariant. The completed row gives up
// the schedule in the same transaction that creates its successor.
func TestTheScheduleMovesToTheOccurrenceThatIsNowOpen(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	due := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	made := newRecurring(t, s, "file the returns", due, domain.Monthly, 1)

	res, err := s.Complete(ctx, made.ID, made.Version, due)
	require.NoError(t, err)

	was, err := s.Todo(ctx, made.ID)
	require.NoError(t, err)
	require.Nil(t, was.RecurUnit, "the completed occurrence no longer repeats")
	require.Nil(t, was.RecurEvery)
	require.Nil(t, was.RecurAnchor)

	now, err := s.Todo(ctx, res.Spawned.ID)
	require.NoError(t, err)
	require.Equal(t, domain.Monthly, *now.RecurUnit)
	require.Equal(t, 1, *now.RecurEvery)
	require.Equal(t, due, now.RecurAnchor.UTC(), "and it carries the anchor, not its own date")
}

// The bug the handover exists to prevent: without it, reopening a completed
// occurrence and finishing it again forks the series.
func TestReopeningAndCompletingAgainDoesNotForkTheSeries(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	due := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	made := newRecurring(t, s, "pay the rent", due, domain.Monthly, 1)

	_, err := s.Complete(ctx, made.ID, made.Version, due)
	require.NoError(t, err)

	reopened, err := s.Todo(ctx, made.ID)
	require.NoError(t, err)
	_, err = s.UpdateTodo(ctx, TodoUpdate{
		ID: reopened.ID, Version: reopened.Version,
		Name: reopened.Name, Status: domain.InProgress, Priority: reopened.Priority,
	})
	require.NoError(t, err)

	again, err := s.Todo(ctx, made.ID)
	require.NoError(t, err)
	res, err := s.Complete(ctx, again.ID, again.Version, due)
	require.NoError(t, err)
	require.Nil(t, res.Spawned, "the schedule left with the first successor")

	all, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2, "the original and one successor, not three rows")
}

// Ticking a task off before it is due should still move it on. Stepping from
// now alone would return the same date and the task would never advance.
func TestCompletingEarlyStillAdvancesTheDate(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	due := time.Date(2026, time.March, 31, 9, 0, 0, 0, time.UTC)
	made := newRecurring(t, s, "submit the timesheet", due, domain.Monthly, 1)

	early := time.Date(2026, time.March, 20, 9, 0, 0, 0, time.UTC)
	res, err := s.Complete(ctx, made.ID, made.Version, early)
	require.NoError(t, err)
	require.NotNil(t, res.Spawned)
	require.Equal(t,
		time.Date(2026, time.April, 30, 9, 0, 0, 0, time.UTC),
		res.Spawned.DueDate.UTC(),
		"April after a 31 January style anchor clamps to the 30th")
}

func TestCompletingAOneOffSpawnsNothing(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "cancel the subscription")

	res, err := s.Complete(ctx, made.ID, made.Version, time.Now())
	require.NoError(t, err)
	require.Equal(t, domain.Completed, res.Completed.Status)
	require.Nil(t, res.Spawned)

	all, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
}

// The version guard is what makes a double click safe: the second request
// carries the version the first one consumed.
func TestCompletingTwiceAtTheSameVersionSpawnsOnce(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	due := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	made := newRecurring(t, s, "back up the laptop", due, domain.Weekly, 1)

	_, err := s.Complete(ctx, made.ID, made.Version, due)
	require.NoError(t, err)

	var conflict *domain.ConflictError
	_, err = s.Complete(ctx, made.ID, made.Version, due)
	require.ErrorAs(t, err, &conflict)

	all, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)
}
