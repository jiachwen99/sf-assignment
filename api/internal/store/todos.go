package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

const todoColumns = `id, name, description, due_date, status, priority,
	recur_unit, recur_interval, recur_anchor, unmet_deps_count,
	version, created_at, updated_at`

type NewTodo struct {
	Name        string
	Description string
	DueDate     *time.Time
	Status      domain.Status
	Priority    domain.Priority
	RecurUnit   *domain.RecurUnit
	RecurEvery  *int
	RecurAnchor *time.Time
}

type TodoUpdate struct {
	ID          int64
	Version     int
	Name        string
	Description string
	DueDate     *time.Time
	Status      domain.Status
	Priority    domain.Priority
	RecurUnit   *domain.RecurUnit
	RecurEvery  *int
	RecurAnchor *time.Time
}

// The casts are not optional. Without them Postgres has nothing to deduce a
// type from inside COALESCE, and every insert with no due date fails.
const insertTodo = `
INSERT INTO todos (name, description, due_date, due_sort, status, priority,
                   recur_unit, recur_interval, recur_anchor)
VALUES ($1, $2, $3::timestamptz, COALESCE($3::timestamptz, 'infinity'), $4, $5,
        $6, $7, $8::timestamptz)
RETURNING ` + todoColumns

func (s *Store) CreateTodo(ctx context.Context, n NewTodo) (domain.Todo, error) {
	var out domain.Todo

	err := s.tx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, insertTodo,
			n.Name, n.Description, n.DueDate, n.Status, n.Priority,
			n.RecurUnit, n.RecurEvery, n.RecurAnchor)
		if err != nil {
			return err
		}
		out, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
		if err != nil {
			return err
		}
		return record(ctx, tx, out.ID, EventCreated, map[string]any{})
	})

	return out, wrap("create todo", err)
}

const selectTodo = `SELECT ` + todoColumns + ` FROM todos WHERE id = $1 AND deleted_at IS NULL`

func (s *Store) Todo(ctx context.Context, id int64) (domain.Todo, error) {
	rows, err := s.pool.Query(ctx, selectTodo, id)
	if err != nil {
		return domain.Todo{}, wrap("read todo", err)
	}
	t, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Todo{}, domain.ErrNotFound
	}
	return t, wrap("read todo", err)
}

// Matched on the version the client read, so a write built from a stale copy
// updates nothing rather than overwriting whatever landed in between.
const updateTodo = `
UPDATE todos
SET name = $3, description = $4,
    due_date = $5::timestamptz, due_sort = COALESCE($5::timestamptz, 'infinity'),
    status = $6, priority = $7,
    recur_unit = $8, recur_interval = $9, recur_anchor = $10::timestamptz,
    version = version + 1, updated_at = now()
WHERE id = $1 AND version = $2 AND deleted_at IS NULL
RETURNING ` + todoColumns

// The blocking rule and the dependents' counters are settled in the same
// transaction as the write, so a counter can never disagree with the status it
// was derived from.
func (s *Store) UpdateTodo(ctx context.Context, u TodoUpdate) (domain.Todo, error) {
	var out domain.Todo

	err := s.tx(ctx, func(tx pgx.Tx) error {
		var was domain.Status
		var unmet int
		err := tx.QueryRow(ctx,
			`SELECT status, unmet_deps_count FROM todos WHERE id = $1`, u.ID).Scan(&was, &unmet)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}

		if err := domain.CanTransition(was, u.Status, unmet, nil, u.ID); err != nil {
			return namedBlockers(ctx, tx, u.ID, err)
		}

		rows, err := tx.Query(ctx, updateTodo,
			u.ID, u.Version, u.Name, u.Description, u.DueDate, u.Status, u.Priority,
			u.RecurUnit, u.RecurEvery, u.RecurAnchor)
		if err != nil {
			return err
		}
		out, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
		if errors.Is(err, pgx.ErrNoRows) {
			return errStale
		}
		if err != nil {
			return err
		}

		if err := adjustDependents(ctx, tx, u.ID, was, u.Status); err != nil {
			return err
		}

		// A status change is the thing people look for in a history, so it is
		// its own kind rather than one "updated" among many.
		if was != u.Status {
			return record(ctx, tx, u.ID, EventStatus, statusPayload(was, u.Status))
		}
		return record(ctx, tx, u.ID, EventUpdated, map[string]any{})
	})

	if errors.Is(err, errStale) {
		return domain.Todo{}, s.conflictOrNotFound(ctx, u.ID)
	}
	return out, wrap("update todo", err)
}

// CanTransition is given the count rather than the names, because the count is
// already on the row and the names cost a query. They are only worth fetching
// once something is actually refused.
func namedBlockers(ctx context.Context, tx pgx.Tx, id int64, err error) error {
	var blocked *domain.BlockedError
	if !errors.As(err, &blocked) {
		return err
	}
	named, lookupErr := blockersTx(ctx, tx, id)
	if lookupErr != nil {
		return lookupErr
	}
	blocked.Blockers = named
	return err
}

// Substring rather than prefix, because nobody types the first word of a task
// name to find it. The caller requires three characters before asking, which
// keeps the least selective searches off the database entirely.
const searchTodos = `
SELECT ` + todoColumns + `
FROM todos
WHERE name ILIKE '%' || $1 || '%' AND id <> $2 AND deleted_at IS NULL
ORDER BY name, id
LIMIT $3`

func (s *Store) SearchTodos(ctx context.Context, term string, excludeID int64, limit int) ([]domain.Todo, error) {
	rows, err := s.pool.Query(ctx, searchTodos, term, excludeID, limit)
	if err != nil {
		return nil, wrap("search todos", err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Todo])
	return out, wrap("search todos", err)
}

// Soft delete: the row stays and so do its dependency edges, which is the whole
// point. A dependent of a deleted task stays blocked, because deleting work is
// not doing it, and restoring puts the chain back exactly rather than roughly.
//
// It carries a version like any other write. Otherwise the one operation you
// cannot undo is the one that ignores what you were looking at.
const softDeleteTodo = `
UPDATE todos SET deleted_at = now(), version = version + 1
WHERE id = $1 AND version = $2 AND deleted_at IS NULL`

func (s *Store) DeleteTodo(ctx context.Context, id int64, version int) error {
	var affected int64

	err := s.tx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, softDeleteTodo, id, version)
		if err != nil {
			return err
		}
		affected = tag.RowsAffected()
		if affected == 0 {
			return nil
		}
		return record(ctx, tx, id, EventDeleted, map[string]any{})
	})
	if err != nil {
		return wrap("delete todo", err)
	}
	if affected == 0 {
		// conflictOrNotFound reads through Todo, which is live-only, so an
		// already-deleted row reports as gone rather than as a conflict.
		return s.conflictOrNotFound(ctx, id)
	}
	return nil
}

const restoreTodo = `
UPDATE todos SET deleted_at = NULL, version = version + 1
WHERE id = $1 AND deleted_at IS NOT NULL
RETURNING ` + todoColumns

// No version guard. Nothing can edit a task while it is in the trash, so the
// copy you are restoring is the only copy there has been.
func (s *Store) RestoreTodo(ctx context.Context, id int64) (domain.Todo, error) {
	var out domain.Todo

	err := s.tx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, restoreTodo, id)
		if err != nil {
			return err
		}
		out, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		return record(ctx, tx, id, EventRestored, map[string]any{})
	})

	return out, wrap("restore todo", err)
}

/*
 * Most recently deleted first: the thing you want back is almost always the
 * thing you just lost.
 *
 * Bounded, like every other list here, and for the same reason: nothing purges
 * the trash, so it is the one table that only ever grows. Capped rather than
 * paged, because the useful end is the near end, and the number that says how
 * much is really in there comes from the counts query.
 *
 * Found unbounded by the SF-016 audit, where it was the only list that would
 * have handed back two hundred thousand rows in a single response.
 */
const trashLimit = 100

const listTrash = `
SELECT ` + todoColumns + `
FROM todos WHERE deleted_at IS NOT NULL
ORDER BY deleted_at DESC, id DESC
LIMIT $1`

func (s *Store) Trash(ctx context.Context) ([]domain.Todo, error) {
	rows, err := s.pool.Query(ctx, listTrash, trashLimit)
	if err != nil {
		return nil, wrap("list trash", err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Todo])
	return out, wrap("list trash", err)
}

// A version-guarded write matching nothing means either the row is gone or
// somebody changed it first, and only a second read tells those apart.
func (s *Store) conflictOrNotFound(ctx context.Context, id int64) error {
	current, err := s.Todo(ctx, id)
	if err != nil {
		return err
	}
	return &domain.ConflictError{Current: current}
}
