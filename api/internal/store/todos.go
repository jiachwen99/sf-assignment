package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

const todoColumns = `id, name, description, due_date, status, priority,
	version, created_at, updated_at`

type NewTodo struct {
	Name        string
	Description string
	DueDate     *time.Time
	Status      domain.Status
	Priority    domain.Priority
}

type TodoUpdate struct {
	ID          int64
	Version     int
	Name        string
	Description string
	DueDate     *time.Time
	Status      domain.Status
	Priority    domain.Priority
}

// The casts are not optional. Without them Postgres has nothing to deduce a
// type from inside COALESCE, and every insert with no due date fails.
const insertTodo = `
INSERT INTO todos (name, description, due_date, due_sort, status, priority)
VALUES ($1, $2, $3::timestamptz, COALESCE($3::timestamptz, 'infinity'), $4, $5)
RETURNING ` + todoColumns

func (s *Store) CreateTodo(ctx context.Context, n NewTodo) (domain.Todo, error) {
	rows, err := s.pool.Query(ctx, insertTodo, n.Name, n.Description, n.DueDate, n.Status, n.Priority)
	if err != nil {
		return domain.Todo{}, wrap("create todo", err)
	}
	out, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
	return out, wrap("create todo", err)
}

const selectTodo = `SELECT ` + todoColumns + ` FROM todos WHERE id = $1`

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

const listTodos = `SELECT ` + todoColumns + ` FROM todos ORDER BY created_at DESC, id DESC`

// Unbounded on purpose. Paging, filtering and sorting arrive together in
// SF-007, where the indexes that make them cheap are designed alongside them.
func (s *Store) Todos(ctx context.Context) ([]domain.Todo, error) {
	rows, err := s.pool.Query(ctx, listTodos)
	if err != nil {
		return nil, wrap("list todos", err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Todo])
	return out, wrap("list todos", err)
}

// Matched on the version the client read, so a write built from a stale copy
// updates nothing rather than overwriting whatever landed in between.
const updateTodo = `
UPDATE todos
SET name = $3, description = $4,
    due_date = $5::timestamptz, due_sort = COALESCE($5::timestamptz, 'infinity'),
    status = $6, priority = $7,
    version = version + 1, updated_at = now()
WHERE id = $1 AND version = $2
RETURNING ` + todoColumns

func (s *Store) UpdateTodo(ctx context.Context, u TodoUpdate) (domain.Todo, error) {
	rows, err := s.pool.Query(ctx, updateTodo,
		u.ID, u.Version, u.Name, u.Description, u.DueDate, u.Status, u.Priority)
	if err != nil {
		return domain.Todo{}, wrap("update todo", err)
	}
	out, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Todo])
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Todo{}, s.conflictOrNotFound(ctx, u.ID)
	}
	return out, wrap("update todo", err)
}

// Delete is a write too, so it carries a version. Otherwise the one operation
// you cannot undo is the one that ignores what you were looking at.
func (s *Store) DeleteTodo(ctx context.Context, id int64, version int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM todos WHERE id = $1 AND version = $2`, id, version)
	if err != nil {
		return wrap("delete todo", err)
	}
	if tag.RowsAffected() == 0 {
		return s.conflictOrNotFound(ctx, id)
	}
	return nil
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
