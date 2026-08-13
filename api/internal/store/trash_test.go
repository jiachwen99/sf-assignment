package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

func TestDeletingHidesATaskWithoutDestroyingIt(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "cancel the subscription")
	require.NoError(t, s.DeleteTodo(ctx, made.ID, made.Version))

	live, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Empty(t, live)

	_, err = s.Todo(ctx, made.ID)
	require.ErrorIs(t, err, domain.ErrNotFound, "gone from every live read")

	trash, err := s.Trash(ctx)
	require.NoError(t, err)
	require.Len(t, trash, 1)
	require.Equal(t, "cancel the subscription", trash[0].Name)
}

func TestRestoringPutsItBackInTheList(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "renew the domain")
	require.NoError(t, s.DeleteTodo(ctx, made.ID, made.Version))

	back, err := s.RestoreTodo(ctx, made.ID)
	require.NoError(t, err)
	require.Equal(t, made.ID, back.ID)

	live, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Len(t, live, 1)

	trash, err := s.Trash(ctx)
	require.NoError(t, err)
	require.Empty(t, trash)
}

// The reason soft delete is worth the column: edges survive, so restoring is
// exact rather than approximate.
func TestDeletingAndRestoringABlockerLeavesItsDependentsExactlyAsBlocked(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")
	data := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))

	require.NoError(t, s.DeleteTodo(ctx, data.ID, data.Version))

	// Deleting work is not doing it, so the dependent is still waiting.
	blocked, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Equal(t, 1, blocked.UnmetDeps)

	deps, err := s.Dependencies(ctx, report.ID)
	require.NoError(t, err)
	require.Len(t, deps, 1, "the edge survives the delete")
	require.True(t, deps[0].Deleted, "and the chain says the blocker is in the trash")

	_, err = s.RestoreTodo(ctx, data.ID)
	require.NoError(t, err)

	after, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Equal(t, 1, after.UnmetDeps, "exactly as blocked as before, not approximately")

	restored, err := s.Dependencies(ctx, report.ID)
	require.NoError(t, err)
	require.False(t, restored[0].Deleted)
}

func TestDeletingTwiceIsNotFound(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "only once")
	require.NoError(t, s.DeleteTodo(ctx, made.ID, made.Version))

	current, err := s.Trash(ctx)
	require.NoError(t, err)
	require.ErrorIs(t, s.DeleteTodo(ctx, made.ID, current[0].Version), domain.ErrNotFound)
}

func TestRestoringSomethingThatWasNeverDeletedIsNotFound(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "still here")
	_, err := s.RestoreTodo(ctx, made.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// A deleted task should not turn up when picking something to depend on.
func TestSearchDoesNotOfferDeletedTasks(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	keep := newTodo(t, s, "insurance renewal")
	gone := newTodo(t, s, "insurance paperwork")
	require.NoError(t, s.DeleteTodo(ctx, gone.ID, gone.Version))

	found, err := s.SearchTodos(ctx, "insur", 0, 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, keep.ID, found[0].ID)
}
