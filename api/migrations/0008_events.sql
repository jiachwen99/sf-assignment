-- +goose Up

-- Append-only, written inside the transaction of the change it describes, so an
-- event can never exist for work that rolled back.
--
-- The foreign key is deliberate. Deletion here is soft, so nothing routinely
-- removes a task's history, but without the reference a truncate of todos
-- leaves this table behind and the next task to be issued an id inherits
-- somebody else's log. Orphaned history is worse than none.
CREATE TABLE todo_events (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    todo_id    bigint NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    kind       text NOT NULL,
    payload    jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

-- The panel reads one task's history in order, which is the only access pattern.
CREATE INDEX todo_events_by_todo ON todo_events (todo_id, id);

-- +goose Down
DROP TABLE todo_events;
