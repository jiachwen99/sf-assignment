package store

import "context"

type Counts struct {
	All        int `json:"all"`
	NotStarted int `json:"notStarted"`
	InProgress int `json:"inProgress"`
	Completed  int `json:"completed"`
	Archived   int `json:"archived"`
	Overdue    int `json:"overdue"`
	Blocked    int `json:"blocked"`
	Recurring  int `json:"recurring"`
	Trash      int `json:"trash"`
}

/*
 * One pass with FILTER rather than nine round trips.
 *
 * This is the only query in the application that scans rather than seeks, and
 * it has to: a count must see every matching row, so the partial indexes that
 * make the list cheap do not help here. That is the argument for asking once
 * and for holding the answer on the client rather than refetching it after
 * every edit.
 */
const selectCounts = `
SELECT
    count(*) FILTER (WHERE deleted_at IS NULL)                            AS all,
    count(*) FILTER (WHERE deleted_at IS NULL AND status = 'not_started') AS not_started,
    count(*) FILTER (WHERE deleted_at IS NULL AND status = 'in_progress') AS in_progress,
    count(*) FILTER (WHERE deleted_at IS NULL AND status = 'completed')   AS completed,
    count(*) FILTER (WHERE deleted_at IS NULL AND status = 'archived')    AS archived,
    count(*) FILTER (WHERE deleted_at IS NULL
                       AND due_date < now()
                       AND status NOT IN ('completed', 'archived'))       AS overdue,
    count(*) FILTER (WHERE deleted_at IS NULL AND unmet_deps_count > 0)   AS blocked,
    count(*) FILTER (WHERE deleted_at IS NULL AND recur_unit IS NOT NULL) AS recurring,
    count(*) FILTER (WHERE deleted_at IS NOT NULL)                        AS trash
FROM todos`

func (s *Store) Counts(ctx context.Context) (Counts, error) {
	var c Counts
	err := s.pool.QueryRow(ctx, selectCounts).Scan(
		&c.All, &c.NotStarted, &c.InProgress, &c.Completed, &c.Archived,
		&c.Overdue, &c.Blocked, &c.Recurring, &c.Trash)
	return c, wrap("counts", err)
}
