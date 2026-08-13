-- +goose Up

-- One index per sort key, each with the id as the trailing tiebreak so the
-- keyset seek and the ordering come from the same index and no sort node is
-- needed. The brief asks for four sort keys and for the list not to degrade at
-- scale; those two together are what pays for these.
--
-- Created already has one: todos_live, from 0005.
CREATE INDEX todos_by_due      ON todos (due_sort, id) WHERE deleted_at IS NULL;
CREATE INDEX todos_by_priority ON todos (priority, id) WHERE deleted_at IS NULL;
CREATE INDEX todos_by_status   ON todos (status, id)   WHERE deleted_at IS NULL;
CREATE INDEX todos_by_name     ON todos (name, id)     WHERE deleted_at IS NULL;

-- Blocked state gets two partial indexes rather than one composite index
-- leading with unmet_deps_count.
--
-- A composite (unmet_deps_count, created_at, id) does serve the filter, but
-- "blocked" is the range predicate unmet_deps_count > 0, and a range on the
-- leading column does not preserve ordering on the columns after it. The
-- planner uses the index for the filter and then adds a sort, which is exactly
-- what keyset pagination exists to avoid. Moving the predicate into a partial
-- index leaves (created_at, id) as the whole key, so the ordering comes free
-- and each index holds only the rows it serves.
--
-- These follow the default sort rather than every sort, because that is the
-- combination the blocked view actually opens on. A blocked list sorted by name
-- falls back to a scan and a sort, and is far rarer.
CREATE INDEX todos_blocked ON todos (created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND unmet_deps_count > 0;
CREATE INDEX todos_unblocked ON todos (created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND unmet_deps_count = 0;

-- +goose Down
DROP INDEX todos_unblocked;
DROP INDEX todos_blocked;
DROP INDEX todos_by_name;
DROP INDEX todos_by_status;
DROP INDEX todos_by_priority;
DROP INDEX todos_by_due;
