package store

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

/*
 * Walks up from the proposed dependency looking for the dependent.
 *
 * Finding it means the edge would close a cycle, and the path is returned so
 * the error can name the loop rather than just refusing.
 *
 * The walk starts at the dependency and the target is deliberately not seeded
 * into the path. Putting it there would make the don't-revisit guard exclude
 * ever reaching it, and every cycle would be accepted.
 */
const findCycle = `
WITH RECURSIVE walk(id, path) AS (
    SELECT $2::bigint, ARRAY[$2::bigint]
  UNION ALL
    SELECT d.depends_on_id, w.path || d.depends_on_id
    FROM walk w
    JOIN todo_dependencies d ON d.todo_id = w.id
    WHERE NOT d.depends_on_id = ANY(w.path)
)
SELECT $1::bigint || path FROM walk WHERE id = $1 LIMIT 1`

const insertEdge = `
INSERT INTO todo_dependencies (todo_id, depends_on_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`

// Site 1 of 4. A new edge to an unfinished task adds one to the count.
const bumpForNewEdge = `
UPDATE todos SET unmet_deps_count = unmet_deps_count + 1
WHERE id = $1 AND EXISTS (
    SELECT 1 FROM todos dep WHERE dep.id = $2 AND dep.status <> 'completed'
)`

func (s *Store) AddDependency(ctx context.Context, todoID, dependsOnID int64) error {
	if todoID == dependsOnID {
		return domain.Invalid("dependsOnId", "A task cannot depend on itself")
	}

	err := s.tx(ctx, func(tx pgx.Tx) error {
		for _, id := range []int64{todoID, dependsOnID} {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT true FROM todos WHERE id = $1`, id).Scan(&exists); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return domain.ErrNotFound
				}
				return err
			}
		}

		var path []int64
		err := tx.QueryRow(ctx, findCycle, todoID, dependsOnID).Scan(&path)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if len(path) > 0 {
			named, err := namePath(ctx, tx, path)
			if err != nil {
				return err
			}
			return domain.Invalid("dependsOnId", "This would create a loop: "+named)
		}

		tag, err := tx.Exec(ctx, insertEdge, todoID, dependsOnID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil // Already there, so the count must not move.
		}

		_, err = tx.Exec(ctx, bumpForNewEdge, todoID, dependsOnID)
		return err
	})

	return wrap("add dependency", err)
}

const deleteEdge = `DELETE FROM todo_dependencies WHERE todo_id = $1 AND depends_on_id = $2`

// Site 2 of 4.
const dropForRemovedEdge = `
UPDATE todos SET unmet_deps_count = unmet_deps_count - 1
WHERE id = $1 AND unmet_deps_count > 0 AND EXISTS (
    SELECT 1 FROM todos dep WHERE dep.id = $2 AND dep.status <> 'completed'
)`

func (s *Store) RemoveDependency(ctx context.Context, todoID, dependsOnID int64) error {
	err := s.tx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, deleteEdge, todoID, dependsOnID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		_, err = tx.Exec(ctx, dropForRemovedEdge, todoID, dependsOnID)
		return err
	})
	return wrap("remove dependency", err)
}

// Sites 3 and 4. Only completed satisfies a dependency, so archiving or
// deleting a blocker deliberately leaves its dependents blocked: shelving work
// is not the same as doing it.
const dropForCompleted = `
UPDATE todos SET unmet_deps_count = unmet_deps_count - 1
WHERE unmet_deps_count > 0 AND id IN (
    SELECT todo_id FROM todo_dependencies WHERE depends_on_id = $1
)`

const bumpForReopened = `
UPDATE todos SET unmet_deps_count = unmet_deps_count + 1
WHERE id IN (SELECT todo_id FROM todo_dependencies WHERE depends_on_id = $1)`

// Note what this does not do: it never bumps the dependent's version. The
// dependent was not edited, and raising its version would hand a conflict to
// everyone with that task open downstream.
func adjustDependents(ctx context.Context, tx pgx.Tx, id int64, was, now domain.Status) error {
	switch {
	case was != domain.Completed && now == domain.Completed:
		_, err := tx.Exec(ctx, dropForCompleted, id)
		return err
	case was == domain.Completed && now != domain.Completed:
		_, err := tx.Exec(ctx, bumpForReopened, id)
		return err
	}
	return nil
}

// What this task waits for.
const selectDependencies = `
SELECT t.id, t.name, t.status, t.deleted_at IS NOT NULL
FROM todo_dependencies d
JOIN todos t ON t.id = d.depends_on_id
WHERE d.todo_id = $1
ORDER BY t.name`

func (s *Store) Dependencies(ctx context.Context, todoID int64) ([]domain.Blocker, error) {
	return s.blockerQuery(ctx, "read dependencies", selectDependencies, todoID)
}

// What waits on this task.
const selectDependents = `
SELECT t.id, t.name, t.status, t.deleted_at IS NOT NULL
FROM todo_dependencies d
JOIN todos t ON t.id = d.todo_id
WHERE d.depends_on_id = $1
ORDER BY t.name`

func (s *Store) Dependents(ctx context.Context, todoID int64) ([]domain.Blocker, error) {
	return s.blockerQuery(ctx, "read dependents", selectDependents, todoID)
}

// The subset of dependencies that are actually holding this task up.
const selectBlockers = `
SELECT t.id, t.name, t.status, t.deleted_at IS NOT NULL
FROM todo_dependencies d
JOIN todos t ON t.id = d.depends_on_id
WHERE d.todo_id = $1 AND t.status <> 'completed'
ORDER BY t.name`

func (s *Store) Blockers(ctx context.Context, todoID int64) ([]domain.Blocker, error) {
	return s.blockerQuery(ctx, "read blockers", selectBlockers, todoID)
}

func (s *Store) blockerQuery(ctx context.Context, op, sql string, todoID int64) ([]domain.Blocker, error) {
	rows, err := s.pool.Query(ctx, sql, todoID)
	if err != nil {
		return nil, wrap(op, err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[domain.Blocker])
	return out, wrap(op, err)
}

// Inside a transaction, for the refusal that has to name what is blocking.
func blockersTx(ctx context.Context, tx pgx.Tx, todoID int64) ([]domain.Blocker, error) {
	rows, err := tx.Query(ctx, selectBlockers, todoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[domain.Blocker])
}

// The refusal is read by a person, so the loop is named rather than numbered.
// One extra query, on a path that only runs when something is being refused.
func namePath(ctx context.Context, tx pgx.Tx, path []int64) (string, error) {
	rows, err := tx.Query(ctx, `SELECT id, name FROM todos WHERE id = ANY($1)`, path)
	if err != nil {
		return "", err
	}
	names := map[int64]string{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return "", err
		}
		names[id] = name
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	parts := make([]string, len(path))
	for i, id := range path {
		if name, ok := names[id]; ok {
			parts[i] = name
		} else {
			parts[i] = "task " + strconv.FormatInt(id, 10)
		}
	}
	return strings.Join(parts, " → "), nil
}
