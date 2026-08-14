package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

type EventKind string

const (
	EventCreated   EventKind = "created"
	EventUpdated   EventKind = "updated"
	EventStatus    EventKind = "status_changed"
	EventCompleted EventKind = "completed"
	EventSpawned   EventKind = "spawned"
	EventDeleted   EventKind = "deleted"
	EventRestored  EventKind = "restored"
	EventDepAdded  EventKind = "dependency_added"
	EventDepRemove EventKind = "dependency_removed"
)

type Event struct {
	ID      int64           `db:"id" json:"id"`
	TodoID  int64           `db:"todo_id" json:"todoId"`
	Kind    EventKind       `db:"kind" json:"kind"`
	Payload json.RawMessage `db:"payload" json:"payload"`
	// Null for a change made while nobody was signed in, which is every change
	// the application made before authentication existed.
	ActorID   *int64    `db:"actor_id" json:"actorId"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

/*
 * Who is making the change, carried on the context.
 *
 * The alternative is an actor argument on every store method, which would put
 * an authentication concern in the signature of every write whether or not that
 * write records anything. A context value is the narrower change.
 */
type actorKey struct{}

func WithActor(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, actorKey{}, id)
}

func actorFrom(ctx context.Context) *int64 {
	if id, ok := ctx.Value(actorKey{}).(int64); ok {
		return &id
	}
	return nil
}

const insertEvent = `
INSERT INTO todo_events (todo_id, kind, payload, actor_id)
VALUES ($1, $2, $3, $4)`

// Takes the caller's transaction rather than the pool, so an event cannot
// survive a rollback of the change it describes.
func record(ctx context.Context, tx pgx.Tx, todoID int64, kind EventKind, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, insertEvent, todoID, kind, body, actorFrom(ctx))
	return err
}

const selectEvents = `
SELECT id, todo_id, kind, payload, actor_id, created_at
FROM todo_events WHERE todo_id = $1 ORDER BY id`

func (s *Store) Events(ctx context.Context, todoID int64) ([]Event, error) {
	rows, err := s.pool.Query(ctx, selectEvents, todoID)
	if err != nil {
		return nil, wrap("read events", err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByName[Event])
	return out, wrap("read events", err)
}

func statusPayload(from, to domain.Status) map[string]any {
	return map[string]any{"from": from, "to": to}
}

// The other task's name is snapshotted rather than resolved when the log is
// read. A log that renames itself when a task is renamed is not an audit trail,
// and a deleted task would leave the entry with nothing to say at all.
func recordEdge(ctx context.Context, tx pgx.Tx, todoID, dependsOnID int64, kind EventKind) error {
	var name string
	if err := tx.QueryRow(ctx,
		`SELECT name FROM todos WHERE id = $1`, dependsOnID).Scan(&name); err != nil {
		return err
	}
	return record(ctx, tx, todoID, kind, map[string]any{
		"dependsOn":     dependsOnID,
		"dependsOnName": name,
	})
}
