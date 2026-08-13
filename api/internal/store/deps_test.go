package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

func TestADependencyBlocksTheTaskThatWaitsForIt(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")
	data := newTodo(t, s, "collect the data")

	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))

	got, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Equal(t, 1, got.UnmetDeps)

	blockers, err := s.Blockers(ctx, report.ID)
	require.NoError(t, err)
	require.Len(t, blockers, 1)
	require.Equal(t, "collect the data", blockers[0].Name)
}

// Only completing releases what waits. That is the whole rule, and it is what
// makes the counter maintainable at four sites rather than everywhere.
func TestCompletingABlockerReleasesItsDependents(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")
	data := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))

	_, err := s.Complete(ctx, data.ID, data.Version, time.Now())
	require.NoError(t, err)

	got, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Zero(t, got.UnmetDeps)
}

func TestReopeningABlockerBlocksItsDependentsAgain(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")
	data := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))

	_, err := s.Complete(ctx, data.ID, data.Version, time.Now())
	require.NoError(t, err)

	done, err := s.Todo(ctx, data.ID)
	require.NoError(t, err)
	_, err = s.UpdateTodo(ctx, TodoUpdate{
		ID: done.ID, Version: done.Version, Name: done.Name,
		Status: domain.InProgress, Priority: done.Priority,
	})
	require.NoError(t, err)

	got, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Equal(t, 1, got.UnmetDeps)
}

// Completing a task is what releases its dependents, so archiving one must not.
// The comment in the store says so; this is the test that keeps it true.
func TestArchivingABlockerLeavesItsDependentsBlocked(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")
	data := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))

	_, err := s.UpdateTodo(ctx, TodoUpdate{
		ID: data.ID, Version: data.Version, Name: data.Name,
		Status: domain.Archived, Priority: data.Priority,
	})
	require.NoError(t, err)

	got, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Equal(t, 1, got.UnmetDeps, "shelving work is not the same as doing it")
}

func TestABlockedTaskCannotStartOrFinish(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")
	data := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))

	current, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)

	var blocked *domain.BlockedError
	_, err = s.UpdateTodo(ctx, TodoUpdate{
		ID: current.ID, Version: current.Version, Name: current.Name,
		Status: domain.InProgress, Priority: current.Priority,
	})
	require.ErrorAs(t, err, &blocked)
	require.Equal(t, "blocked by collect the data", blocked.Error(),
		"the refusal names what to go and finish")

	_, err = s.Complete(ctx, current.ID, current.Version, time.Now())
	require.ErrorAs(t, err, &blocked, "and the further destination is guarded too")
}

func TestAddingADependencyTwiceCountsOnce(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")
	data := newTodo(t, s, "collect the data")

	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))
	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))

	got, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Equal(t, 1, got.UnmetDeps)
}

func TestRemovingADependencyUnblocks(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")
	data := newTodo(t, s, "collect the data")
	require.NoError(t, s.AddDependency(ctx, report.ID, data.ID))
	require.NoError(t, s.RemoveDependency(ctx, report.ID, data.ID))

	got, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Zero(t, got.UnmetDeps)

	deps, err := s.Dependencies(ctx, report.ID)
	require.NoError(t, err)
	require.Empty(t, deps)
}

// The case the recursive walk exists for, and the one it is easiest to get
// wrong: seeding the target into the path makes the don't-revisit guard exclude
// ever reaching it, and every cycle is quietly accepted.
func TestACycleIsRefusedAndTheLoopIsNamed(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	a := newTodo(t, s, "a")
	b := newTodo(t, s, "b")
	require.NoError(t, s.AddDependency(ctx, a.ID, b.ID))

	err := s.AddDependency(ctx, b.ID, a.ID)
	var invalid *domain.ValidationError
	require.ErrorAs(t, err, &invalid)
	require.Contains(t, invalid.Fields["dependsOnId"], "loop")
	require.Contains(t, invalid.Fields["dependsOnId"], "b → a → b",
		"the refusal names the loop and reads in the direction of waiting")
}

func TestALongerCycleIsRefusedToo(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	a := newTodo(t, s, "a")
	b := newTodo(t, s, "b")
	c := newTodo(t, s, "c")
	require.NoError(t, s.AddDependency(ctx, a.ID, b.ID))
	require.NoError(t, s.AddDependency(ctx, b.ID, c.ID))

	var invalid *domain.ValidationError
	require.ErrorAs(t, s.AddDependency(ctx, c.ID, a.ID), &invalid)
}

// A diamond is not a cycle. Refusing it would be the over-eager version of the
// same check.
func TestADiamondIsAllowed(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	top := newTodo(t, s, "top")
	left := newTodo(t, s, "left")
	right := newTodo(t, s, "right")
	base := newTodo(t, s, "base")

	require.NoError(t, s.AddDependency(ctx, top.ID, left.ID))
	require.NoError(t, s.AddDependency(ctx, top.ID, right.ID))
	require.NoError(t, s.AddDependency(ctx, left.ID, base.ID))
	require.NoError(t, s.AddDependency(ctx, right.ID, base.ID))

	got, err := s.Todo(ctx, top.ID)
	require.NoError(t, err)
	require.Equal(t, 2, got.UnmetDeps)
}

func TestATaskCannotDependOnItself(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	only := newTodo(t, s, "only")
	require.Error(t, s.AddDependency(ctx, only.ID, only.ID))
}

func TestDependenciesAndDependentsAreBothVisible(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	middle := newTodo(t, s, "middle")
	above := newTodo(t, s, "above")
	below := newTodo(t, s, "below")

	require.NoError(t, s.AddDependency(ctx, middle.ID, below.ID))
	require.NoError(t, s.AddDependency(ctx, above.ID, middle.ID))

	deps, err := s.Dependencies(ctx, middle.ID)
	require.NoError(t, err)
	require.Len(t, deps, 1)
	require.Equal(t, "below", deps[0].Name)

	dependents, err := s.Dependents(ctx, middle.ID)
	require.NoError(t, err)
	require.Len(t, dependents, 1)
	require.Equal(t, "above", dependents[0].Name)
}

func TestSearchMatchesInsideTheName(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	newTodo(t, s, "renew the office insurance")
	newTodo(t, s, "call the plumber")
	excluded := newTodo(t, s, "insurance paperwork")

	found, err := s.SearchTodos(ctx, "insur", excluded.ID, 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, "renew the office insurance", found[0].Name,
		"substring, not prefix, and the task doing the searching is left out")
}
