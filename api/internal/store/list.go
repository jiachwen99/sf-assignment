package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

type SortField string

const (
	SortCreated  SortField = "created"
	SortDue      SortField = "due"
	SortPriority SortField = "priority"
	SortStatus   SortField = "status"
	SortName     SortField = "name"
)

// Due sorts on due_sort rather than due_date, so the keyset tuple is never NULL
// and undated tasks sort to one end instead of dropping out of every page.
var sortColumns = map[SortField]struct{ column, cast string }{
	SortCreated:  {"created_at", "timestamptz"},
	SortDue:      {"due_sort", "timestamptz"},
	SortPriority: {"priority", "todo_priority"},
	SortStatus:   {"status", "todo_status"},
	SortName:     {"name", "text"},
}

func (f SortField) Valid() bool {
	_, ok := sortColumns[f]
	return ok
}

// A direction rather than a bool, so "unset" is a value the default can fill
// in. A bool would make ascending and unspecified the same thing, and the
// default direction would have to be decided somewhere else.
type SortDir string

const (
	Ascending  SortDir = "asc"
	Descending SortDir = "desc"
)

type ListFilter struct {
	Statuses   []domain.Status
	Priorities []domain.Priority
	DueFrom    *time.Time
	DueTo      *time.Time
	Blocked    *bool
	Recurring  *bool
	// A one-sided filter, because "not overdue" is not a question anyone asks
	// of a task list.
	Overdue bool
	Name    string

	Sort   SortField
	Dir    SortDir
	Cursor string
	Limit  int
}

type Page struct {
	Items      []domain.Todo `json:"items"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

// The cursor carries the sort it was issued under and is rejected against a
// different one, rather than silently paging through the wrong order.
type cursor struct {
	Sort  SortField `json:"s"`
	Value string    `json:"v"`
	ID    int64     `json:"i"`
	Dir   SortDir   `json:"d"`
}

func encodeCursor(c cursor) string {
	raw, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(s string) (cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, domain.Invalid("cursor", "This page link is not valid")
	}
	var c cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return cursor{}, domain.Invalid("cursor", "This page link is not valid")
	}
	return c, nil
}

// Conditions and their arguments together, so a filter cannot be added without
// its placeholder and the numbering cannot drift.
type builder struct {
	conds []string
	args  []any
}

func (b *builder) add(cond string, args ...any) {
	for _, a := range args {
		b.args = append(b.args, a)
		cond = strings.Replace(cond, "?", "$"+strconv.Itoa(len(b.args)), 1)
	}
	b.conds = append(b.conds, cond)
}

const defaultLimit = 50

// Newest first, because a task list is read from the top. The alternative,
// oldest due first, opens on whatever has been rotting longest.
func (f *ListFilter) normalise() error {
	if f.Sort == "" {
		f.Sort = SortCreated
	}
	if !f.Sort.Valid() {
		return domain.Invalid("sort", "Sort must be created, due, priority, status or name")
	}
	if f.Dir == "" {
		f.Dir = Descending
	}
	if f.Dir != Ascending && f.Dir != Descending {
		return domain.Invalid("dir", "Direction must be asc or desc")
	}
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = defaultLimit
	}
	return nil
}

// Built separately from being run, so the query-plan test can EXPLAIN the real
// query rather than an approximation of it that could drift.
func buildListQuery(f ListFilter) (string, []any) {
	_ = f.normalise()
	col := sortColumns[f.Sort]

	b := &builder{}
	b.add("deleted_at IS NULL")

	if len(f.Statuses) > 0 {
		b.add("status = ANY(?::todo_status[])", asStrings(f.Statuses))
	}
	if len(f.Priorities) > 0 {
		b.add("priority = ANY(?::todo_priority[])", asStrings(f.Priorities))
	}
	if f.DueFrom != nil {
		b.add("due_date >= ?", *f.DueFrom)
	}
	if f.DueTo != nil {
		b.add("due_date <= ?", *f.DueTo)
	}
	if f.Blocked != nil {
		if *f.Blocked {
			b.add("unmet_deps_count > 0")
		} else {
			b.add("unmet_deps_count = 0")
		}
	}
	if f.Recurring != nil {
		if *f.Recurring {
			b.add("recur_unit IS NOT NULL")
		} else {
			b.add("recur_unit IS NULL")
		}
	}
	// Finished and shelved tasks are not overdue: there is nothing left to do
	// about them, and counting them would make the number permanent.
	if f.Overdue {
		b.add("due_date < now() AND status NOT IN ('completed', 'archived')")
	}
	if f.Name != "" {
		// The one filter that scans rather than seeks; see 0004_name_search.sql.
		b.add("name ILIKE '%' || ? || '%'", f.Name)
	}

	// Keyset, not OFFSET. Offset makes the database walk and discard every row
	// it skips, so page 400 costs four hundred times page one.
	if f.Cursor != "" {
		c, _ := decodeCursor(f.Cursor)
		op := ">"
		if f.Dir == Descending {
			op = "<"
		}
		b.add("("+col.column+", id) "+op+" (?::"+col.cast+", ?)", c.Value, c.ID)
	}

	direction := "ASC"
	if f.Dir == Descending {
		direction = "DESC"
	}

	return `SELECT ` + todoColumns + `
FROM todos
WHERE ` + strings.Join(b.conds, " AND ") + `
ORDER BY ` + col.column + ` ` + direction + `, id ` + direction + `
LIMIT ` + strconv.Itoa(f.Limit+1), b.args
}

func (s *Store) List(ctx context.Context, f ListFilter) (Page, error) {
	if err := f.normalise(); err != nil {
		return Page{}, err
	}
	if f.Cursor != "" {
		c, err := decodeCursor(f.Cursor)
		if err != nil {
			return Page{}, err
		}
		if c.Sort != f.Sort || c.Dir != f.Dir {
			return Page{}, domain.Invalid("cursor", "This page link belongs to a different sort order")
		}
	}

	query, args := buildListQuery(f)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return Page{}, wrap("list", err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Todo])
	if err != nil {
		return Page{}, wrap("list", err)
	}

	// One extra row was asked for, so a next page can be detected without a
	// second query and without counting the whole table.
	page := Page{Items: items}
	if len(items) > f.Limit {
		page.Items = items[:f.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(cursor{
			Sort: f.Sort, Dir: f.Dir, ID: last.ID, Value: sortValue(f.Sort, last),
		})
	}
	return page, nil
}

// pgx cannot encode a slice of a named string type into a Postgres enum array,
// so these are flattened and cast in the query.
func asStrings[T ~string](in []T) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return out
}

func sortValue(field SortField, t domain.Todo) string {
	switch field {
	case SortDue:
		if t.DueDate == nil {
			return "infinity"
		}
		return t.DueDate.UTC().Format(time.RFC3339Nano)
	case SortPriority:
		return string(t.Priority)
	case SortStatus:
		return string(t.Status)
	case SortName:
		return t.Name
	default:
		return t.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
}
