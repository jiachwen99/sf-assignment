package store

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

/*
 * The flagship test for the denormalisation.
 *
 * unmet_deps_count is derived state, and the risk of derived state is that it
 * drifts from the truth. Four maintenance sites is few enough to reason about
 * and more than enough to miss one. This throws random sequences of dependency
 * and status changes at the store, then checks every counter against a query
 * that recomputes it from scratch.
 *
 * A missed site shows up here rather than as a task that is silently
 * unblockable in production.
 */
func TestCounterInvariant(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	rapid.Check(t, func(rt *rapid.T) {
		reset(rt, s)

		const n = 6
		ids := make([]int64, n)
		for i := range ids {
			todo, err := s.CreateTodo(ctx, NewTodo{
				Name: "task", Status: domain.NotStarted, Priority: domain.Medium,
			})
			if err != nil {
				rt.Fatalf("create: %v", err)
			}
			ids[i] = todo.ID
		}

		steps := rapid.IntRange(1, 25).Draw(rt, "steps")
		for range steps {
			pick := rapid.IntRange(0, n-1)
			a, b := ids[pick.Draw(rt, "a")], ids[pick.Draw(rt, "b")]

			// Cycles, self-dependencies and blocked transitions are refused by
			// design, so a rejection here is correct behaviour rather than a
			// failure. What matters is that the counter is right afterwards
			// either way.
			switch rapid.IntRange(0, 3).Draw(rt, "op") {
			case 0:
				_ = s.AddDependency(ctx, a, b)
			case 1:
				_ = s.RemoveDependency(ctx, a, b)
			case 2:
				setStatus(ctx, s, a, domain.Completed)
			case 3:
				setStatus(ctx, s, a, domain.NotStarted)
			}
		}

		assertCountersMatchTruth(rt, s)
	})
}

func reset(rt *rapid.T, s *Store) {
	if _, err := s.pool.Exec(context.Background(),
		`TRUNCATE todos, todo_dependencies RESTART IDENTITY CASCADE`); err != nil {
		rt.Fatalf("reset: %v", err)
	}
}

func setStatus(ctx context.Context, s *Store, id int64, status domain.Status) {
	current, err := s.Todo(ctx, id)
	if err != nil {
		return
	}
	if status == domain.Completed {
		// Completing goes through the real path, so recurrence spawning and
		// counter maintenance are exercised together.
		_, _ = s.Complete(ctx, id, current.Version, time.Now())
		return
	}
	_, _ = s.UpdateTodo(ctx, TodoUpdate{
		ID: id, Version: current.Version, Name: current.Name,
		Status: status, Priority: current.Priority,
	})
}

// The ground truth: count each task's dependencies that are not completed. Any
// row where the stored count disagrees is returned, so the failure names the
// task and both numbers.
const truthQuery = `
SELECT t.id, t.unmet_deps_count, (
    SELECT count(*)
    FROM todo_dependencies d
    JOIN todos dep ON dep.id = d.depends_on_id
    WHERE d.todo_id = t.id AND dep.status <> 'completed'
) AS actual
FROM todos t
WHERE t.unmet_deps_count <> (
    SELECT count(*)
    FROM todo_dependencies d
    JOIN todos dep ON dep.id = d.depends_on_id
    WHERE d.todo_id = t.id AND dep.status <> 'completed'
)`

func assertCountersMatchTruth(rt *rapid.T, s *Store) {
	rows, err := s.pool.Query(context.Background(), truthQuery)
	if err != nil {
		rt.Fatalf("truth query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var stored, actual int
		if err := rows.Scan(&id, &stored, &actual); err != nil {
			rt.Fatalf("scan: %v", err)
		}
		rt.Errorf("todo %d: unmet_deps_count is %d, recomputed truth is %d", id, stored, actual)
	}
	if err := rows.Err(); err != nil {
		rt.Fatalf("rows: %v", err)
	}
}
