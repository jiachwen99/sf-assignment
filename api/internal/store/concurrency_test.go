package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

/*
 * The brief asks the API to support several people using the same list at once.
 * The counter is maintained by application code inside a transaction, so the
 * question the SF-016 audit has to answer is whether two people completing two
 * different blockers of the same task at the same moment leave the count right.
 *
 * Ten blockers, ten goroutines, one dependent. If the read-modify-write raced,
 * the count would end above zero and the dependent would be permanently
 * unstartable with nothing left to finish.
 */
func TestConcurrentCompletionsLeaveTheCounterCorrect(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	report := newTodo(t, s, "write the report")

	const n = 10
	blockers := make([]int64, n)
	for i := range blockers {
		b := newTodo(t, s, "blocker")
		blockers[i] = b.ID
		require.NoError(t, s.AddDependency(ctx, report.ID, b.ID))
	}

	before, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Equal(t, n, before.UnmetDeps)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, id := range blockers {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			<-start // release them together
			_, err := s.Complete(context.Background(), id, 1, time.Now())
			require.NoError(t, err)
		}(id)
	}
	close(start)
	wg.Wait()

	after, err := s.Todo(ctx, report.ID)
	require.NoError(t, err)
	require.Zero(t, after.UnmetDeps, "every blocker finished, so nothing is still waiting")
}
