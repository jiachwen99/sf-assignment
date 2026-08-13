package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

// Everything, in the default order. The tests that predate paging want a whole
// list rather than a page, and saying so once keeps them readable.
func allTodos(ctx context.Context, s *Store) ([]domain.Todo, error) {
	page, err := s.List(ctx, ListFilter{Limit: 200})
	return page.Items, err
}

func dated(t *testing.T, s *Store, name string, due time.Time, status domain.Status, priority domain.Priority) domain.Todo {
	t.Helper()
	todo, err := s.CreateTodo(context.Background(), NewTodo{
		Name: name, DueDate: &due, Status: status, Priority: priority,
	})
	require.NoError(t, err)
	return todo
}

func names(page Page) []string {
	out := make([]string, len(page.Items))
	for i, t := range page.Items {
		out[i] = t.Name
	}
	return out
}

func seedForList(t *testing.T, s *Store) {
	t.Helper()
	day := func(n int) time.Time { return time.Date(2026, time.March, n, 9, 0, 0, 0, time.UTC) }

	dated(t, s, "alpha", day(3), domain.NotStarted, domain.High)
	dated(t, s, "bravo", day(1), domain.InProgress, domain.Low)
	dated(t, s, "charlie", day(2), domain.Completed, domain.Medium)

	// One undated task, which is the row that drops out of every page if the
	// keyset tuple is allowed to contain a NULL.
	_, err := s.CreateTodo(context.Background(), NewTodo{
		Name: "delta", Status: domain.NotStarted, Priority: domain.High,
	})
	require.NoError(t, err)
}

func TestSortingByEachKey(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()
	seedForList(t, s)

	cases := []struct {
		name string
		sort SortField
		dir  SortDir
		want []string
	}{
		{"due ascending puts the undated task last", SortDue, Ascending,
			[]string{"bravo", "charlie", "alpha", "delta"}},
		{"due descending puts it first", SortDue, Descending,
			[]string{"delta", "alpha", "charlie", "bravo"}},
		{"name ascending", SortName, Ascending,
			[]string{"alpha", "bravo", "charlie", "delta"}},
		{"name descending", SortName, Descending,
			[]string{"delta", "charlie", "bravo", "alpha"}},
		// The enum is ordered by declaration, so low sorts before high without
		// a CASE expression.
		{"priority ascending is low first", SortPriority, Ascending,
			[]string{"bravo", "charlie", "alpha", "delta"}},
		{"status ascending follows the enum", SortStatus, Ascending,
			[]string{"alpha", "delta", "bravo", "charlie"}},
		{"created descending is newest first", SortCreated, Descending,
			[]string{"delta", "charlie", "bravo", "alpha"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			page, err := s.List(ctx, ListFilter{Sort: c.sort, Dir: c.dir, Limit: 50})
			require.NoError(t, err)
			require.Equal(t, c.want, names(page))
		})
	}
}

// An undated task must appear on some page. A NULL in the keyset tuple makes
// every comparison NULL, and the row is silently never returned.
func TestAnUndatedTaskIsNeverLostBetweenPages(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()
	seedForList(t, s)

	var seen []string
	filter := ListFilter{Sort: SortDue, Dir: Ascending, Limit: 1}
	for range 10 {
		page, err := s.List(ctx, filter)
		require.NoError(t, err)
		seen = append(seen, names(page)...)
		if page.NextCursor == "" {
			break
		}
		filter.Cursor = page.NextCursor
	}
	require.Equal(t, []string{"bravo", "charlie", "alpha", "delta"}, seen)
}

func TestPagingWalksEveryRowExactlyOnce(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	for i := range 25 {
		newTodo(t, s, "task "+strings.Repeat("x", i%3)+string(rune('a'+i)))
	}

	seen := map[int64]bool{}
	filter := ListFilter{Sort: SortCreated, Dir: Descending, Limit: 7}
	pages := 0
	for {
		page, err := s.List(ctx, filter)
		require.NoError(t, err)
		pages++
		for _, todo := range page.Items {
			require.False(t, seen[todo.ID], "todo %d returned twice", todo.ID)
			seen[todo.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		filter.Cursor = page.NextCursor
		require.Less(t, pages, 10, "paging did not terminate")
	}

	require.Len(t, seen, 25)
	require.Equal(t, 4, pages, "25 rows at 7 a page")
}

// The cursor carries the sort it was issued under, so a client that changes the
// sort without resetting gets an error rather than a wrong page.
func TestACursorFromADifferentSortIsRefused(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()
	seedForList(t, s)

	page, err := s.List(ctx, ListFilter{Sort: SortName, Dir: Ascending, Limit: 2})
	require.NoError(t, err)
	require.NotEmpty(t, page.NextCursor)

	var invalid *domain.ValidationError
	_, err = s.List(ctx, ListFilter{Sort: SortDue, Dir: Ascending, Cursor: page.NextCursor, Limit: 2})
	require.ErrorAs(t, err, &invalid)

	_, err = s.List(ctx, ListFilter{Sort: SortName, Dir: Descending, Cursor: page.NextCursor, Limit: 2})
	require.ErrorAs(t, err, &invalid, "the direction is part of the order too")
}

func TestGarbageCursorIsRejected(t *testing.T) {
	s := NewTestStore(t)

	var invalid *domain.ValidationError
	_, err := s.List(context.Background(), ListFilter{Cursor: "not-a-cursor!!"})
	require.ErrorAs(t, err, &invalid)
}

func TestNoNextCursorOnTheLastPage(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()
	seedForList(t, s)

	page, err := s.List(ctx, ListFilter{Limit: 50})
	require.NoError(t, err)
	require.Len(t, page.Items, 4)
	require.Empty(t, page.NextCursor, "four rows and room for fifty is not a next page")
}

func TestFilters(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()
	seedForList(t, s)

	blocker := newTodo(t, s, "blocker")
	waiting := newTodo(t, s, "waiting")
	require.NoError(t, s.AddDependency(ctx, waiting.ID, blocker.ID))

	yes, no := true, false
	day := func(n int) *time.Time {
		d := time.Date(2026, time.March, n, 9, 0, 0, 0, time.UTC)
		return &d
	}

	cases := []struct {
		name   string
		filter ListFilter
		want   []string
	}{
		{"by status", ListFilter{Statuses: []domain.Status{domain.Completed}, Sort: SortName, Dir: Ascending},
			[]string{"charlie"}},
		{"by several statuses", ListFilter{
			Statuses: []domain.Status{domain.InProgress, domain.Completed}, Sort: SortName, Dir: Ascending},
			[]string{"bravo", "charlie"}},
		{"by priority", ListFilter{Priorities: []domain.Priority{domain.High}, Sort: SortName, Dir: Ascending},
			[]string{"alpha", "delta"}},
		{"by a due range", ListFilter{DueFrom: day(2), DueTo: day(3), Sort: SortName, Dir: Ascending},
			[]string{"alpha", "charlie"}},
		{"blocked", ListFilter{Blocked: &yes, Sort: SortName, Dir: Ascending}, []string{"waiting"}},
		{"unblocked", ListFilter{Blocked: &no, Sort: SortName, Dir: Ascending},
			[]string{"alpha", "blocker", "bravo", "charlie", "delta"}},
		{"by name, matched inside the word", ListFilter{Name: "rav", Sort: SortName, Dir: Ascending},
			[]string{"bravo"}},
		{"filters combine", ListFilter{
			Statuses: []domain.Status{domain.NotStarted}, Priorities: []domain.Priority{domain.High},
			Sort: SortName, Dir: Ascending}, []string{"alpha", "delta"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.filter.Limit = 50
			page, err := s.List(ctx, c.filter)
			require.NoError(t, err)
			require.Equal(t, c.want, names(page))
		})
	}
}

func TestAnUnknownSortIsRejected(t *testing.T) {
	s := NewTestStore(t)

	var invalid *domain.ValidationError
	_, err := s.List(context.Background(), ListFilter{Sort: SortField("colour")})
	require.ErrorAs(t, err, &invalid)
}

/*
 * The test the indexes exist for.
 *
 * Every sort key, in both directions, has to resolve as an index scan with no
 * sort node: the ordering comes out of the index rather than being computed per
 * page. An index that stops being used fails the build here rather than showing
 * up as a slow list at two hundred thousand rows.
 */
func TestListQueryPlan(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()
	seedForList(t, s)

	// The planner picks a sequential scan on a tiny table whatever the indexes
	// say, so the plan is only meaningful once there is enough data to make an
	// index worth using.
	for range 2000 {
		newTodo(t, s, "bulk task")
	}
	_, err := s.pool.Exec(ctx, "ANALYZE todos")
	require.NoError(t, err)

	blocked := true
	cases := []struct {
		name   string
		filter ListFilter
	}{
		{"created descending, the default", ListFilter{Sort: SortCreated, Dir: Descending}},
		{"created ascending", ListFilter{Sort: SortCreated, Dir: Ascending}},
		{"due ascending", ListFilter{Sort: SortDue, Dir: Ascending}},
		{"due descending", ListFilter{Sort: SortDue, Dir: Descending}},
		{"priority", ListFilter{Sort: SortPriority, Dir: Ascending}},
		{"priority descending", ListFilter{Sort: SortPriority, Dir: Descending}},
		{"status", ListFilter{Sort: SortStatus, Dir: Ascending}},
		{"status descending", ListFilter{Sort: SortStatus, Dir: Descending}},
		{"name", ListFilter{Sort: SortName, Dir: Ascending}},
		{"name descending", ListFilter{Sort: SortName, Dir: Descending}},
		{"blocked on the default sort", ListFilter{Blocked: &blocked, Sort: SortCreated, Dir: Descending}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.filter.Limit = 50
			plan := explain(t, s, c.filter)
			require.Contains(t, plan, "Index Scan", "should seek, not scan:\n%s", plan)
			require.NotContains(t, plan, "Sort  ", "ordering should come from the index:\n%s", plan)
		})
	}
}

func explain(t *testing.T, s *Store, f ListFilter) string {
	t.Helper()

	// Explains the real query by asking the store to build it, so the plan
	// cannot drift from what actually runs.
	sql, args := buildListQuery(f)
	rows, err := s.pool.Query(context.Background(), "EXPLAIN "+sql, args...)
	require.NoError(t, err)
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		lines = append(lines, line)
	}
	require.NoError(t, rows.Err())
	return strings.Join(lines, "\n")
}
