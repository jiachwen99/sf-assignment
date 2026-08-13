package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

type CompleteResult struct {
	Completed domain.Todo
	Spawned   *domain.Todo
}

const markCompleted = `
UPDATE todos
SET status = 'completed', version = version + 1, updated_at = now()
WHERE id = $1 AND version = $2
RETURNING ` + todoColumns

// The schedule moves to the occurrence that is now open. Without this the
// completed row keeps it, and reopening then completing again spawns a second
// occurrence on the same date, forking the series. The version is untouched:
// nobody edited this row, the series moved on.
const handOverSchedule = `
UPDATE todos
SET recur_unit = NULL, recur_interval = NULL, recur_anchor = NULL
WHERE id = $1
RETURNING ` + todoColumns

const insertOccurrence = `
INSERT INTO todos (name, description, due_date, due_sort, status, priority,
                   recur_unit, recur_interval, recur_anchor)
VALUES ($1, $2, $3::timestamptz, COALESCE($3::timestamptz, 'infinity'), 'not_started', $4,
        $5, $6, $7::timestamptz)
RETURNING ` + todoColumns

// Completing and spawning share one transaction, so a series can never be left
// with two live occurrences or none. The version check makes it idempotent
// as well: a double click reads one version, so the second attempt matches no
// rows and cannot spawn a duplicate.
func (s *Store) Complete(ctx context.Context, id int64, version int, now time.Time) (CompleteResult, error) {
	var out CompleteResult

	err := s.tx(ctx, func(tx pgx.Tx) error {
		var was domain.Status
		var unmet int
		err := tx.QueryRow(ctx,
			`SELECT status, unmet_deps_count FROM todos WHERE id = $1`, id).Scan(&was, &unmet)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := domain.CanTransition(was, domain.Completed, unmet, nil, id); err != nil {
			return namedBlockers(ctx, tx, id, err)
		}

		rows, err := tx.Query(ctx, markCompleted, id, version)
		if err != nil {
			return err
		}
		completed, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
		if errors.Is(err, pgx.ErrNoRows) {
			return errStale
		}
		if err != nil {
			return err
		}
		out.Completed = completed

		if err := adjustDependents(ctx, tx, id, was, domain.Completed); err != nil {
			return err
		}

		if completed.RecurUnit == nil || completed.RecurEvery == nil {
			return nil
		}

		// The later of now and this occurrence's own due date. Using now alone
		// would return August again when August's rent is ticked off on the
		// 11th, and the task would never advance.
		after := now
		if completed.DueDate != nil && completed.DueDate.After(after) {
			after = *completed.DueDate
		}

		next, err := domain.NextDue(completed.RecurAnchor, *completed.RecurUnit, *completed.RecurEvery, after)
		if err != nil {
			return err
		}

		rows, err = tx.Query(ctx, insertOccurrence,
			completed.Name, completed.Description, next, completed.Priority,
			completed.RecurUnit, completed.RecurEvery, completed.RecurAnchor)
		if err != nil {
			return err
		}
		spawned, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
		if err != nil {
			return err
		}
		out.Spawned = &spawned

		rows, err = tx.Query(ctx, handOverSchedule, id)
		if err != nil {
			return err
		}
		handedOver, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
		if err != nil {
			return err
		}
		out.Completed = handedOver
		return nil
	})

	if errors.Is(err, errStale) {
		return CompleteResult{}, s.conflictOrNotFound(ctx, id)
	}
	return out, wrap("complete todo", err)
}

var errStale = errors.New("stale version")
