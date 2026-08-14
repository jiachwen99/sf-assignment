package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
	"github.com/jiachwen99/sf-assignment/api/internal/events"
	"github.com/jiachwen99/sf-assignment/api/internal/service"
	"github.com/jiachwen99/sf-assignment/api/internal/store"
)

func newService(t *testing.T) (*service.Service, *store.Store) {
	t.Helper()
	st := store.NewTestStore(t)
	return service.New(st, events.NewHub()), st
}

func create(t *testing.T, svc *service.Service, name string) domain.Todo {
	t.Helper()
	made, err := svc.Create(context.Background(), service.TodoInput{Name: name})
	require.NoError(t, err)
	return made
}

func items(todos ...domain.Todo) []service.BulkItem {
	out := make([]service.BulkItem, len(todos))
	for i, t := range todos {
		out[i] = service.BulkItem{ID: t.ID, Version: t.Version}
	}
	return out
}

// The whole reason this is per item rather than one transaction: one refusal
// must not discard the successes around it.
func TestOneStaleItemDoesNotDiscardTheRest(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	first := create(t, svc, "first")
	stale := create(t, svc, "stale")
	last := create(t, svc, "last")

	// Somebody else edits the middle one, so the version held here is old.
	_, err := svc.Update(ctx, stale.ID, stale.Version, service.TodoInput{
		Name: "stale, edited elsewhere", Priority: domain.High,
	})
	require.NoError(t, err)

	results := svc.BulkComplete(ctx, items(first, stale, last))
	require.Len(t, results, 3)

	require.True(t, results[0].OK)
	require.False(t, results[1].OK)
	require.Equal(t, "This task was changed by someone else", results[1].Error)
	require.True(t, results[2].OK, "the refusal did not stop the batch")

	// And the two that succeeded actually did.
	for _, id := range []int64{first.ID, last.ID} {
		got, err := svc.Todo(ctx, id)
		require.NoError(t, err)
		require.Equal(t, domain.Completed, got.Status)
	}
	untouched, err := svc.Todo(ctx, stale.ID)
	require.NoError(t, err)
	require.NotEqual(t, domain.Completed, untouched.Status)
}

/*
 * The trap this ticket carries.
 *
 * A selection holding a blocker and its dependent completes both, because the
 * blocker goes first and releases the dependent on the way past. That is
 * correct behaviour, and it is exactly why it cannot be the partial-failure
 * test: it succeeds, so a test written around it would be asserting success
 * while believing it had proved a refusal.
 */
func TestABlockerAndItsDependentBothComplete(t *testing.T) {
	svc, st := newService(t)
	ctx := context.Background()

	blocker := create(t, svc, "collect the data")
	waiting := create(t, svc, "write the report")
	require.NoError(t, st.AddDependency(ctx, waiting.ID, blocker.ID))

	current, err := svc.Todo(ctx, waiting.ID)
	require.NoError(t, err)

	results := svc.BulkComplete(ctx, items(blocker, current))
	require.True(t, results[0].OK)
	require.True(t, results[1].OK, "the blocker released it on the way past")
}

// And the same pair in the other order is refused, which is what makes the
// order-dependence real rather than incidental.
func TestTheDependentAloneIsRefusedAndSaysWhatBy(t *testing.T) {
	svc, st := newService(t)
	ctx := context.Background()

	blocker := create(t, svc, "collect the data")
	waiting := create(t, svc, "write the report")
	require.NoError(t, st.AddDependency(ctx, waiting.ID, blocker.ID))

	current, err := svc.Todo(ctx, waiting.ID)
	require.NoError(t, err)

	results := svc.BulkComplete(ctx, items(current, blocker))
	require.False(t, results[0].OK)
	require.Equal(t, "This task is blocked by unfinished work", results[0].Error)
	require.Len(t, results[0].Blockers, 1)
	require.Equal(t, "collect the data", results[0].Blockers[0].Name,
		"the batch names what to go and finish, like the single-task path does")
	require.True(t, results[1].OK)
}

func TestArchivingABatch(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	first := create(t, svc, "first")
	second := create(t, svc, "second")

	results := svc.BulkArchive(ctx, items(first, second))
	require.True(t, results[0].OK)
	require.True(t, results[1].OK)

	for _, id := range []int64{first.ID, second.ID} {
		got, err := svc.Todo(ctx, id)
		require.NoError(t, err)
		require.Equal(t, domain.Archived, got.Status)
	}
}

// Archiving is always legal, including for a task nothing can start.
func TestABlockedTaskCanStillBeArchivedInABatch(t *testing.T) {
	svc, st := newService(t)
	ctx := context.Background()

	blocker := create(t, svc, "collect the data")
	waiting := create(t, svc, "write the report")
	require.NoError(t, st.AddDependency(ctx, waiting.ID, blocker.ID))

	current, err := svc.Todo(ctx, waiting.ID)
	require.NoError(t, err)

	results := svc.BulkArchive(ctx, items(current))
	require.True(t, results[0].OK, "shelving a task you cannot start is the point of archiving")
}

func TestAMissingTaskIsReportedRatherThanFailingTheBatch(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	real := create(t, svc, "real")

	results := svc.BulkComplete(ctx, append(items(real),
		service.BulkItem{ID: 999_999, Version: 1}))
	require.True(t, results[0].OK)
	require.False(t, results[1].OK)
	require.Equal(t, "This task no longer exists", results[1].Error)
}

// Results come back in the order asked for, so a caller can line them up
// against its own selection without matching on id.
func TestResultsAreInTheOrderGiven(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	first := create(t, svc, "first")
	second := create(t, svc, "second")
	third := create(t, svc, "third")

	results := svc.BulkComplete(ctx, items(third, first, second))
	require.Equal(t,
		[]int64{third.ID, first.ID, second.ID},
		[]int64{results[0].ID, results[1].ID, results[2].ID})
}
