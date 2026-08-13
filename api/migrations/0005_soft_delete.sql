-- +goose Up

-- Delete is recoverable. The row stays, its dependency edges stay, and
-- restoring is one column change, so a restored task blocks its dependents
-- exactly as it did before rather than approximately.
ALTER TABLE todos ADD COLUMN deleted_at timestamptz;

-- Every list reads live rows only, so the predicate belongs in the index rather
-- than being applied to the result of one. The columns match the default list
-- as it stands today; the indexes for the other sort keys arrive with the sorts
-- themselves in SF-007, where they can be checked against a query plan.
CREATE INDEX todos_live ON todos (created_at DESC, id DESC) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX todos_live;
ALTER TABLE todos DROP COLUMN deleted_at;
