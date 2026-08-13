package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

func newTodo(t *testing.T, s *Store, name string) domain.Todo {
	t.Helper()
	todo, err := s.CreateTodo(context.Background(), NewTodo{
		Name: name, Status: domain.NotStarted, Priority: domain.Medium,
	})
	require.NoError(t, err)
	return todo
}

func TestCreateRoundTripsEveryField(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	due := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)
	made, err := s.CreateTodo(ctx, NewTodo{
		Name:        "pay rent",
		Description: "standing order",
		DueDate:     &due,
		Status:      domain.InProgress,
		Priority:    domain.High,
	})
	require.NoError(t, err)

	// Read it back rather than trusting what the insert returned.
	got, err := s.Todo(ctx, made.ID)
	require.NoError(t, err)
	require.Equal(t, "pay rent", got.Name)
	require.Equal(t, "standing order", got.Description)
	require.Equal(t, domain.InProgress, got.Status)
	require.Equal(t, domain.High, got.Priority)
	require.Equal(t, due, got.DueDate.UTC())
	require.Equal(t, 1, got.Version)
}

// An undated task is the case that breaks if the COALESCE placeholder is not
// cast, because Postgres has nothing to deduce the type from.
func TestUndatedTodoIsAccepted(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made, err := s.CreateTodo(ctx, NewTodo{
		Name: "someday", Status: domain.NotStarted, Priority: domain.Low,
	})
	require.NoError(t, err)
	require.Nil(t, made.DueDate)

	// Asserted in SQL rather than scanned: infinity has no time.Time to scan
	// into, which is the point of using it as the sort key.
	var infinite bool
	require.NoError(t, s.pool.QueryRow(ctx,
		`SELECT due_sort = 'infinity' FROM todos WHERE id = $1`, made.ID).Scan(&infinite))
	require.True(t, infinite,
		"an undated task sorts to one end rather than dropping out of the ordering")
}

func TestEmptyNameIsRejectedAndNothingPersists(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	_, err := s.CreateTodo(ctx, NewTodo{
		Name: "   ", Status: domain.NotStarted, Priority: domain.Medium,
	})
	require.Error(t, err)

	all, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Empty(t, all)
}

func TestUpdateBumpsVersion(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "draft the note")
	updated, err := s.UpdateTodo(ctx, TodoUpdate{
		ID: made.ID, Version: made.Version,
		Name: "draft the memo", Status: domain.InProgress, Priority: domain.High,
	})
	require.NoError(t, err)
	require.Equal(t, made.Version+1, updated.Version)

	got, err := s.Todo(ctx, made.ID)
	require.NoError(t, err)
	require.Equal(t, "draft the memo", got.Name)
	require.Equal(t, domain.InProgress, got.Status)
}

func TestUnknownIDIsNotFound(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	_, err := s.Todo(ctx, 4242)
	require.ErrorIs(t, err, domain.ErrNotFound)

	require.ErrorIs(t, s.DeleteTodo(ctx, 4242, 1), domain.ErrNotFound)
}

func TestDeleteRemovesItFromTheList(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	keep := newTodo(t, s, "keep")
	gone := newTodo(t, s, "gone")

	require.NoError(t, s.DeleteTodo(ctx, gone.ID, gone.Version))

	all, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, keep.ID, all[0].ID)
}

// Newest first, because a task list is read from the top.
func TestListIsNewestFirst(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	first := newTodo(t, s, "first")
	second := newTodo(t, s, "second")

	all, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, second.ID, all[0].ID)
	require.Equal(t, first.ID, all[1].ID)
}

// The whole point of the version guard: the second writer loses, and the row
// still holds what the first writer put there.
func TestStaleUpdateIsRejectedAndChangesNothing(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "book the room")
	stale := made.Version

	_, err := s.UpdateTodo(ctx, TodoUpdate{
		ID: made.ID, Version: stale,
		Name: "book the big room", Status: domain.InProgress, Priority: domain.Medium,
	})
	require.NoError(t, err)

	_, err = s.UpdateTodo(ctx, TodoUpdate{
		ID: made.ID, Version: stale,
		Name: "cancel the room", Status: domain.Archived, Priority: domain.Low,
	})

	var conflict *domain.ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, "book the big room", conflict.Current.Name,
		"the conflict carries the row as it now stands, not as the loser sent it")

	// Read it back rather than trusting the error: the assertion that matters
	// is that nothing of the second write reached the table.
	got, err := s.Todo(ctx, made.ID)
	require.NoError(t, err)
	require.Equal(t, "book the big room", got.Name)
	require.Equal(t, domain.InProgress, got.Status)
	require.Equal(t, stale+1, got.Version)
}

func TestStaleDeleteIsRejectedAndTheRowSurvives(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "pay the invoice")
	_, err := s.UpdateTodo(ctx, TodoUpdate{
		ID: made.ID, Version: made.Version,
		Name: "pay the invoice", Status: domain.InProgress, Priority: domain.High,
	})
	require.NoError(t, err)

	var conflict *domain.ConflictError
	require.ErrorAs(t, s.DeleteTodo(ctx, made.ID, made.Version), &conflict)

	all, err := s.Todos(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
}

// A missing row and a stale version both match nothing, and the difference
// decides whether the client reloads or is told the task is gone.
func TestDeletedRowIsNotFoundRatherThanAConflict(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "chase the courier")
	require.NoError(t, s.DeleteTodo(ctx, made.ID, made.Version))

	_, err := s.UpdateTodo(ctx, TodoUpdate{
		ID: made.ID, Version: made.Version,
		Name: "chase the courier again", Status: domain.NotStarted, Priority: domain.Medium,
	})
	require.ErrorIs(t, err, domain.ErrNotFound)
}
