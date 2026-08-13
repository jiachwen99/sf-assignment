-- +goose Up

-- Declaration order is the sort order. ORDER BY priority DESC returns
-- high, medium, low without a CASE expression, which would stop the
-- index serving the sort.
CREATE TYPE todo_status AS ENUM ('not_started', 'in_progress', 'completed', 'archived');
CREATE TYPE todo_priority AS ENUM ('low', 'medium', 'high');

CREATE TABLE todos (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        text NOT NULL CHECK (length(btrim(name)) > 0),
    description text NOT NULL DEFAULT '',
    due_date    timestamptz,

    -- Keyset pagination compares (sort_key, id) against the cursor. A NULL
    -- due_date makes that comparison NULL, so WHERE drops the row and undated
    -- tasks vanish from every page. due_sort is never NULL, so the comparison
    -- is total in both directions. Written alongside due_date on every write.
    due_sort    timestamptz NOT NULL DEFAULT 'infinity',

    status      todo_status NOT NULL DEFAULT 'not_started',
    priority    todo_priority NOT NULL DEFAULT 'medium',

    -- Every row carries one from the start, so the concurrency work in SF-003
    -- needs no backfill.
    version     integer NOT NULL DEFAULT 1,

    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE todos;
DROP TYPE todo_priority;
DROP TYPE todo_status;
