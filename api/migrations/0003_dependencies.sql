-- +goose Up
CREATE TABLE todo_dependencies (
    todo_id       bigint NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    depends_on_id bigint NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    PRIMARY KEY (todo_id, depends_on_id),
    CONSTRAINT no_self_dependency CHECK (todo_id <> depends_on_id)
);

-- Finding what depends on a task, for the warning shown before deleting a
-- blocker. The primary key already covers the other direction.
CREATE INDEX todo_dependencies_reverse ON todo_dependencies (depends_on_id);

-- Blocked has to be an indexable predicate rather than a subquery, or filtering
-- to the blocked rows while sorting by due date scans thousands of rows per
-- page. Denormalised, so it is maintained at exactly four sites: dependency
-- added, dependency removed, task entering completed, task leaving completed.
ALTER TABLE todos ADD COLUMN unmet_deps_count integer NOT NULL DEFAULT 0
    CHECK (unmet_deps_count >= 0);

-- +goose Down
ALTER TABLE todos DROP COLUMN unmet_deps_count;
DROP TABLE todo_dependencies;
