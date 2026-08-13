-- +goose Up

-- Substring search over names, needed here because you cannot add a dependency
-- without first finding the task by name.
--
-- Prefix matching would use the plain btree index and keep the single-seek
-- promise intact. It is also not what people mean by search: nobody types the
-- first word of a task name to find it.
--
-- A leading wildcard cannot use a btree index, so this adds a trigram index
-- instead. The honest cost is that name search is the one filter that does not
-- resolve as a single index seek: it is a bitmap scan followed by a sort. The
-- client requires three characters before it queries, which keeps the shortest
-- and least selective searches from ever reaching the database.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX todos_name_trgm ON todos USING gin (name gin_trgm_ops);

-- +goose Down
DROP INDEX todos_name_trgm;
