package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

func kinds(events []Event) []EventKind {
	out := make([]EventKind, len(events))
	for i, e := range events {
		out[i] = e.Kind
	}
	return out
}

func payload(t *testing.T, e Event) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(e.Payload, &out))
	return out
}

func TestEveryChangeIsRecordedInOrder(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "draft the note")
	_, err := s.UpdateTodo(ctx, TodoUpdate{
		ID: made.ID, Version: made.Version,
		Name: "draft the memo", Status: domain.NotStarted, Priority: domain.High,
	})
	require.NoError(t, err)

	current, err := s.Todo(ctx, made.ID)
	require.NoError(t, err)
	_, err = s.UpdateTodo(ctx, TodoUpdate{
		ID: current.ID, Version: current.Version,
		Name: current.Name, Status: domain.InProgress, Priority: current.Priority,
	})
	require.NoError(t, err)

	events, err := s.Events(ctx, made.ID)
	require.NoError(t, err)
	require.Equal(t, []EventKind{EventCreated, EventUpdated, EventStatus}, kinds(events),
		"a status change is its own kind, because that is what people look for")

	moved := payload(t, events[2])
	require.Equal(t, "not_started", moved["from"])
	require.Equal(t, "in_progress", moved["to"])
}

// An event must not survive a rollback of the change it describes, which is why
// record takes the caller's transaction rather than the pool.
func TestARefusedChangeLeavesNoEvent(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	waiting := newTodo(t, s, "write the report")
	blocker := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, waiting.ID, blocker.ID))

	before, err := s.Events(ctx, waiting.ID)
	require.NoError(t, err)

	current, err := s.Todo(ctx, waiting.ID)
	require.NoError(t, err)
	_, err = s.UpdateTodo(ctx, TodoUpdate{
		ID: current.ID, Version: current.Version,
		Name: current.Name, Status: domain.InProgress, Priority: current.Priority,
	})
	require.Error(t, err, "blocked, so refused")

	after, err := s.Events(ctx, waiting.ID)
	require.NoError(t, err)
	require.Len(t, after, len(before), "the refusal wrote nothing")
}

// The whole reason the name is snapshotted: a log that rewrites itself when a
// task is renamed is not an audit trail.
func TestRenamingABlockerDoesNotRewriteWhatTheLogSaid(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	waiting := newTodo(t, s, "write the report")
	blocker := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, waiting.ID, blocker.ID))

	_, err := s.UpdateTodo(ctx, TodoUpdate{
		ID: blocker.ID, Version: blocker.Version,
		Name: "collect the numbers", Status: domain.NotStarted, Priority: domain.Medium,
	})
	require.NoError(t, err)

	events, err := s.Events(ctx, waiting.ID)
	require.NoError(t, err)
	require.Equal(t, EventDepAdded, events[len(events)-1].Kind)

	linked := payload(t, events[len(events)-1])
	require.Equal(t, "collect the data", linked["dependsOnName"],
		"the log says what the task was called at the time")
	require.Equal(t, float64(blocker.ID), linked["dependsOn"])
}

// Either end of a recurrence opens the other, so the history is navigable in
// both directions rather than only forwards.
func TestBothEndsOfARecurrenceAreLinked(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	due := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	made := newRecurring(t, s, "pay the rent", due, domain.Monthly, 1)

	res, err := s.Complete(ctx, made.ID, made.Version, due)
	require.NoError(t, err)
	require.NotNil(t, res.Spawned)

	completed, err := s.Events(ctx, made.ID)
	require.NoError(t, err)
	last := completed[len(completed)-1]
	require.Equal(t, EventCompleted, last.Kind)
	require.Equal(t, float64(res.Spawned.ID), payload(t, last)["spawned"],
		"the completed occurrence names what it created")

	// Spawned and not also created: "spawned from the March occurrence" already
	// says the row is new and says something a bare "created" does not.
	spawned, err := s.Events(ctx, res.Spawned.ID)
	require.NoError(t, err)
	require.Equal(t, []EventKind{EventSpawned}, kinds(spawned))
	from := payload(t, spawned[0])
	require.Equal(t, float64(made.ID), from["from"], "and the new one names where it came from")
	require.Equal(t, "pay the rent", from["fromName"])
}

func TestDeletingAndRestoringAreBothRecorded(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "cancel the subscription")
	require.NoError(t, s.DeleteTodo(ctx, made.ID, made.Version))

	binned, err := s.Trash(ctx)
	require.NoError(t, err)
	_, err = s.RestoreTodo(ctx, binned[0].ID)
	require.NoError(t, err)

	events, err := s.Events(ctx, made.ID)
	require.NoError(t, err)
	require.Equal(t, []EventKind{EventCreated, EventDeleted, EventRestored}, kinds(events),
		"the history survives the trash, which is what makes restore trustworthy")
}

func TestUnlinkingIsRecordedSeparately(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	waiting := newTodo(t, s, "write the report")
	blocker := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, waiting.ID, blocker.ID))
	require.NoError(t, s.RemoveDependency(ctx, waiting.ID, blocker.ID))

	events, err := s.Events(ctx, waiting.ID)
	require.NoError(t, err)
	require.Equal(t, []EventKind{EventCreated, EventDepAdded, EventDepRemove}, kinds(events))
}

// A no-op write should not fabricate history.
func TestAddingTheSameDependencyTwiceRecordsOnce(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	waiting := newTodo(t, s, "write the report")
	blocker := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, waiting.ID, blocker.ID))
	require.NoError(t, s.AddDependency(ctx, waiting.ID, blocker.ID))

	events, err := s.Events(ctx, waiting.ID)
	require.NoError(t, err)
	require.Equal(t, []EventKind{EventCreated, EventDepAdded}, kinds(events))
}

// The foreign key exists for this: without it a truncate leaves the log behind
// and the next task issued an id inherits somebody else's history.
func TestHistoryDoesNotOutliveItsTask(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made := newTodo(t, s, "first")
	require.NotEmpty(t, mustEvents(t, s, made.ID))

	_, err := s.pool.Exec(ctx, `TRUNCATE todos RESTART IDENTITY CASCADE`)
	require.NoError(t, err)

	reissued := newTodo(t, s, "second")
	require.Equal(t, made.ID, reissued.ID, "the same id comes round again")

	events := mustEvents(t, s, reissued.ID)
	require.Equal(t, []EventKind{EventCreated}, kinds(events),
		"and it starts with its own history, not the previous task's")
}

func mustEvents(t *testing.T, s *Store, id int64) []Event {
	t.Helper()
	events, err := s.Events(context.Background(), id)
	require.NoError(t, err)
	return events
}
