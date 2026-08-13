-- +goose Up

-- The recurring view, in the default order. Only the live occurrence of a
-- series carries a schedule, the completed ones having handed it on, so this
-- index holds one row per series rather than one per occurrence.
CREATE INDEX todos_recurring ON todos (created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND recur_unit IS NOT NULL;

-- +goose Down
DROP INDEX todos_recurring;
