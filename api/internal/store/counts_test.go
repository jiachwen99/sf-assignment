package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

func TestCountsCoverEveryView(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	past := time.Now().Add(-48 * time.Hour)
	future := time.Now().Add(48 * time.Hour)

	dated(t, s, "overdue and not started", past, domain.NotStarted, domain.Medium)
	dated(t, s, "overdue and in progress", past, domain.InProgress, domain.Medium)
	dated(t, s, "not due yet", future, domain.NotStarted, domain.Medium)

	// Past due but finished with, so not overdue: there is nothing left to do.
	dated(t, s, "late but done", past, domain.Completed, domain.Medium)
	dated(t, s, "late but shelved", past, domain.Archived, domain.Medium)

	weekly := domain.Weekly
	every := 1
	_, err := s.CreateTodo(ctx, NewTodo{
		Name: "repeats", Status: domain.NotStarted, Priority: domain.Medium,
		DueDate: &future, RecurUnit: &weekly, RecurEvery: &every, RecurAnchor: &future,
	})
	require.NoError(t, err)

	blocker := newTodo(t, s, "blocker")
	waiting := newTodo(t, s, "waiting")
	require.NoError(t, s.AddDependency(ctx, waiting.ID, blocker.ID))

	binned := newTodo(t, s, "deleted")
	require.NoError(t, s.DeleteTodo(ctx, binned.ID, binned.Version))

	counts, err := s.Counts(ctx)
	require.NoError(t, err)

	require.Equal(t, 8, counts.All, "nine created, one deleted")
	require.Equal(t, 5, counts.NotStarted)
	require.Equal(t, 1, counts.InProgress)
	require.Equal(t, 1, counts.Completed)
	require.Equal(t, 1, counts.Archived)
	require.Equal(t, 2, counts.Overdue, "finished and shelved tasks are not overdue")
	require.Equal(t, 1, counts.Blocked)
	require.Equal(t, 1, counts.Recurring)
	require.Equal(t, 1, counts.Trash, "the trash is not part of all")
}

// The rail claims the four statuses add up to the total, and that is the claim
// a reader checks by eye. If it stops being true the rail is lying, so it is
// held to it here rather than trusted.
func TestTheFourStatusesSumToAllTasks(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()
	seedForList(t, s)

	blocker := newTodo(t, s, "blocker")
	waiting := newTodo(t, s, "waiting")
	require.NoError(t, s.AddDependency(ctx, waiting.ID, blocker.ID))

	binned := newTodo(t, s, "deleted")
	require.NoError(t, s.DeleteTodo(ctx, binned.ID, binned.Version))

	counts, err := s.Counts(ctx)
	require.NoError(t, err)
	require.Equal(t,
		counts.All,
		counts.NotStarted+counts.InProgress+counts.Completed+counts.Archived,
		"every live task has exactly one status")
}

func TestCountsOfAnEmptyListAreAllZero(t *testing.T) {
	s := NewTestStore(t)

	counts, err := s.Counts(context.Background())
	require.NoError(t, err)
	require.Equal(t, Counts{}, counts)
}
