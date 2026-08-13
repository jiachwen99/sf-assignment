// Command seed fills the database with realistic tasks, so the interface and
// the latency numbers are both exercised against something that looks like a
// backlog rather than twenty thousand rows called "task 4213".
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	rows := flag.Int("n", 20_000, "how many tasks to create")
	seed := flag.Uint64("seed", 42, "random seed, so published numbers are reproducible")
	flag.Parse()

	if err := run(*rows, *seed); err != nil {
		log.Fatal(err)
	}
}

func run(rows int, seed uint64) error {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://todo:todo@localhost:5432/todo?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	start := time.Now()
	rng := rand.New(rand.NewPCG(seed, seed))
	now := time.Now().UTC()

	// todo_events references todos, so CASCADE reaches it. Without that key it
	// would survive here, and RESTART IDENTITY replaying the same ids would hand
	// the next task somebody else's history.
	if _, err := pool.Exec(ctx,
		`TRUNCATE todos RESTART IDENTITY CASCADE`); err != nil {
		return err
	}

	// A real list has one internet bill that repeats, not a hundred and seventy
	// separate ones. Repeating work comes from a fixed catalogue of distinct
	// chores, emitted once each; everything else is one-off.
	chores := catalogue()

	// COPY rather than row by row: at two hundred thousand rows the round trips
	// dominate everything else in the run.
	src := pgx.CopyFromSlice(rows, func(i int) ([]any, error) {
		t := generate(rng, now, chores, i)

		var due, sortKey, anchor any = nil, forever, nil
		if t.due != nil {
			due, sortKey = *t.due, *t.due
		}
		var unit, interval any
		if t.unit != "" {
			unit, interval, anchor = t.unit, t.interval, due
		}

		return []any{
			t.name, t.desc, due, sortKey, t.status, t.priority, unit, interval, anchor,
			t.created, t.updated,
		}, nil
	})

	n, err := pool.CopyFrom(ctx,
		pgx.Identifier{"todos"},
		[]string{"name", "description", "due_date", "due_sort", "status", "priority",
			"recur_unit", "recur_interval", "recur_anchor", "created_at", "updated_at"},
		src)
	if err != nil {
		return err
	}

	// Dense enough that the blocked filter matches a small fraction of rows,
	// which is what stresses the partial index. Acyclic by construction: every
	// edge points at a lower id.
	if _, err := pool.Exec(ctx, `
		INSERT INTO todo_dependencies (todo_id, depends_on_id)
		SELECT t.id, t.id - 1 - (random() * 5)::int
		FROM todos t
		WHERE t.id > 10 AND t.id % 10 = 0
		ON CONFLICT DO NOTHING`); err != nil {
		return err
	}

	// The rows were written straight in, so the denormalised counter is
	// backfilled once at the end rather than maintained a row at a time.
	if _, err := pool.Exec(ctx, `
		UPDATE todos t SET unmet_deps_count = (
			SELECT count(*) FROM todo_dependencies d
			JOIN todos dep ON dep.id = d.depends_on_id
			WHERE d.todo_id = t.id AND dep.status <> 'completed'
		)`); err != nil {
		return err
	}

	// A little in the trash, so the view has something in it and the partial
	// indexes are exercised against rows they have to exclude.
	if _, err := pool.Exec(ctx, `
		UPDATE todos SET deleted_at = now() - (random() * 20 || ' days')::interval
		WHERE id % 250 = 0`); err != nil {
		return err
	}

	// Every task has at least the event that created it, so the history panel is
	// never empty on seeded data.
	if _, err := pool.Exec(ctx, `
		INSERT INTO todo_events (todo_id, kind, payload, created_at)
		SELECT id, 'created', '{}'::jsonb, created_at FROM todos`); err != nil {
		return err
	}

	// Without this the planner is working from statistics for an empty table,
	// and the first measurements are of sequential scans.
	if _, err := pool.Exec(ctx, `ANALYZE todos`); err != nil {
		return err
	}

	fmt.Printf("seeded %d tasks in %s\n", n, time.Since(start).Round(time.Millisecond))
	return nil
}

// Literally infinity, not a distant date.
//
// due_sort is NOT NULL so an undated task sorts to one end rather than dropping
// out of every page, and the application writes 'infinity' there. Seeding
// 9999-12-31 instead looks identical until a keyset page compares the two: a
// task edited through the interface would sort ahead of every seeded one that
// is supposed to be its equal.
var forever = pgtype.Timestamptz{InfinityModifier: pgtype.Infinity, Valid: true}

type task struct {
	name     string
	desc     string
	due      *time.Time
	status   string
	priority string
	unit     string
	interval int
	created  time.Time
	updated  time.Time
}

func generate(rng *rand.Rand, now time.Time, chores []task, i int) task {
	var t task
	if i < len(chores) {
		t = chores[i]
	} else {
		t = oneOff(rng)
	}

	// Completing an occurrence hands its schedule to the one it creates, so a
	// finished task never carries one. Seeding otherwise would produce rows the
	// application itself cannot reach, and the first thing anyone would notice
	// is a completed task sitting in the recurring view.
	if t.unit != "" {
		t.status = pick(rng, liveWeights)
	} else {
		t.status = pick(rng, statusWeights)
	}
	t.priority = pick(rng, priorityWeights)
	t.due = dueDate(rng, now)

	// A recurring task with no anchor can never open its next occurrence, so
	// these are always dated.
	if t.unit != "" && t.due == nil {
		d := now.AddDate(0, 0, rng.IntN(30))
		t.due = &d
	}

	t.created = createdAt(rng, now, t.status)
	t.updated = updatedAt(rng, t.created, now, t.status)
	return t
}

// One COPY stamps every row with the same instant, which leaves the created
// column reading "Today" twenty thousand times and gives the default sort
// nothing to order by. Spread over the past year and weighted towards recent,
// because a list grows.
func createdAt(rng *rand.Rand, now time.Time, status string) time.Time {
	days := math.Pow(rng.Float64(), 2) * 365
	if status == "completed" || status == "archived" {
		days = 21 + math.Pow(rng.Float64(), 1.5)*344
	}
	return now.AddDate(0, 0, -int(days)).Add(-time.Duration(rng.IntN(86400)) * time.Second)
}

// Most not-started work has never been touched since it was written down.
func updatedAt(rng *rand.Rand, created, now time.Time, status string) time.Time {
	if status == "not_started" && rng.IntN(100) < 70 {
		return created
	}
	gap := now.Sub(created)
	if gap <= 0 {
		return created
	}
	return created.Add(time.Duration(rng.Int64N(int64(gap))))
}

var (
	liveWeights = []weighted{
		{"not_started", 70}, {"in_progress", 30},
	}
	statusWeights = []weighted{
		{"not_started", 45}, {"in_progress", 20}, {"completed", 30}, {"archived", 5},
	}
	priorityWeights = []weighted{
		{"low", 25}, {"medium", 55}, {"high", 20},
	}
)

type weighted struct {
	value  string
	weight int
}

func pick(rng *rand.Rand, options []weighted) string {
	total := 0
	for _, o := range options {
		total += o.weight
	}
	n := rng.IntN(total)
	for _, o := range options {
		if n < o.weight {
			return o.value
		}
		n -= o.weight
	}
	return options[len(options)-1].value
}

// Most work is due soon, some of it is late, and a long tail sits further out.
// A uniform spread would leave the overdue view either empty or useless.
func dueDate(rng *rand.Rand, now time.Time) *time.Time {
	if rng.IntN(100) < 30 {
		return nil
	}
	var days int
	switch n := rng.IntN(100); {
	case n < 20:
		days = -(rng.IntN(60) + 1)
	case n < 80:
		days = rng.IntN(30)
	default:
		days = 30 + rng.IntN(150)
	}
	d := now.AddDate(0, 0, days).Truncate(time.Hour)
	return &d
}

var systems = []string{
	"billing service", "checkout flow", "search index", "auth gateway",
	"mobile app", "admin console", "data pipeline", "notification worker",
	"reporting API", "image resizer", "webhook dispatcher", "audit log",
	"payments ledger", "onboarding flow", "rate limiter", "session store",
	"invoice renderer", "export service", "feature flag service", "media store",
	"pricing engine", "referral service", "tax calculator", "usage meter",
}

var people = []string{
	"Priya", "Marcus", "Wei", "Sofia", "Daniel", "Amara", "Tomas", "Yuki",
	"Hassan", "Elena", "Noor", "Kofi",
}

var vendors = []string{
	"AWS", "Datadog", "Stripe", "Twilio", "Figma", "Cloudflare", "Sentry", "Postmark",
}

/*
 * Every template carries something that varies beyond the system it names, so
 * the list does not fill with two hundred rows called the same thing.
 *
 * That is not decoration: a screen of identical names makes search, sorting and
 * the dependency picker impossible to judge, which is exactly what a reviewer
 * is trying to judge.
 */
func oneOff(rng *rand.Rand) task {
	sys := func() string { return systems[rng.IntN(len(systems))] }
	who := func() string { return people[rng.IntN(len(people))] }
	from := func(xs []string) string { return xs[rng.IntN(len(xs))] }

	switch rng.IntN(14) {
	case 0, 1:
		return task{
			name: fmt.Sprintf("Review pull request %d in the %s", 1000+rng.IntN(9000), sys()),
			desc: "Second reviewer needed before it can merge.",
		}
	case 2, 3:
		return task{
			name: fmt.Sprintf("Investigate %d %s in the %s", 10+rng.IntN(900), from(symptoms), sys()),
			desc: "Started around the same time as the traffic increase.",
		}
	case 4:
		return task{
			name: fmt.Sprintf("Fix the flaky %s in the %s", from(suites), sys()),
			desc: "Fails about one run in twenty on CI. Suspect a timing assumption.",
		}
	case 5:
		return task{
			name: fmt.Sprintf("Write the %s migration for the %s", from(migrations), sys()),
			desc: "The backfill has to run before the column becomes NOT NULL.",
		}
	case 6:
		return task{
			name: fmt.Sprintf("Update the %s runbook for the %s", from(procedures), sys()),
			desc: "Out of date since the last deploy.",
		}
	case 7:
		return task{
			name: fmt.Sprintf("Upgrade the %s SDK to %d.%d", from(vendors), 2+rng.IntN(8), rng.IntN(20)),
			desc: "The current version stops receiving security fixes this quarter.",
		}
	case 8:
		return task{
			name: fmt.Sprintf("Add tracing to the %s in the %s", from(paths), sys()),
			desc: "No visibility into where the time goes once a request leaves the edge.",
		}
	case 9:
		return task{
			name: fmt.Sprintf("Draft the design doc for %s in the %s", from(topics), sys()),
			desc: "Two approaches to compare before committing to either.",
		}
	case 10:
		return task{
			name: fmt.Sprintf("Remove the %s flag from the %s", from(flags), sys()),
			desc: "Fully rolled out for six weeks.",
		}
	case 11:
		return task{
			name: fmt.Sprintf("Cut release %d.%d.%d", 1+rng.IntN(4), rng.IntN(12), rng.IntN(20)),
			desc: "Tag, notes, and a smoke test against staging first.",
		}
	case 12:
		return task{
			name: fmt.Sprintf("Follow up with %s about %s in the %s", who(), from(topics), sys()),
			desc: "Waiting on an answer before the next step can start.",
		}
	default:
		return task{
			name: fmt.Sprintf("Benchmark the %s under %s", sys(), from(loads)),
			desc: "Numbers before and after, with the command that reproduces them.",
		}
	}
}

var (
	symptoms   = []string{"timeouts", "duplicate deliveries", "5xx responses", "slow queries", "retries", "dropped jobs"}
	suites     = []string{"integration test", "contract test", "smoke test", "end-to-end test", "unit test"}
	migrations = []string{"audit column", "tenant id", "soft delete", "index rebuild", "column split", "backfill"}
	procedures = []string{"failover", "deploy", "rollback", "on-call", "restore", "incident"}
	paths      = []string{"write path", "read path", "background worker", "webhook handler", "batch job"}
	topics     = []string{"partitioning", "rate limits", "idempotency keys", "schema versioning", "retry policy", "cache invalidation", "quotas"}
	flags      = []string{"legacy checkout", "new pricing", "beta search", "async export", "v2 onboarding", "dark launch"}
	loads      = []string{"peak load", "a cold cache", "a single connection", "concurrent writes"}
)

// The unit and interval match what the task actually is, so the controls read
// sensibly rather than offering a daily quarterly review. Each entry appears
// once, which is what makes the recurring view worth opening.
func catalogue() []task {
	var out []task

	for _, v := range vendors {
		out = append(out, task{
			name: "Rotate the " + v + " credentials", unit: "month", interval: 3,
			desc: "Issue the new key, deploy, then revoke the old one.",
		})
	}
	for _, sys := range systems {
		out = append(out,
			task{
				name: "Check certificate expiry on the " + sys, unit: "month", interval: 1,
				desc: "Anything inside thirty days gets renewed now, not later.",
			},
			task{
				name: "Review the " + sys + " error budget", unit: "month", interval: 1,
				desc: "If it is spent, the next sprint is reliability work.",
			})
	}
	for _, who := range people {
		out = append(out, task{
			name: "One to one with " + who, unit: "week", interval: 2,
			desc: "Their agenda first.",
		})
	}

	return append(out,
		task{name: "Weekly team sync notes", unit: "week", interval: 1,
			desc: "Decisions and owners, circulated the same day."},
		task{name: "Monthly invoice run", unit: "month", interval: 1,
			desc: "Reconcile against the ledger before sending."},
		task{name: "Quarterly access review", unit: "month", interval: 3,
			desc: "Everyone who left, and everyone whose role changed."},
		task{name: "Pay the office internet bill", unit: "month", interval: 1,
			desc: "The invoice arrives on the second and is due on the fifteenth."},
		task{name: "Back up the analytics warehouse", unit: "day", interval: 1,
			desc: "Check the restore, not just that the job exited zero."},
		task{name: "Sprint planning", unit: "week", interval: 2,
			desc: "Carry over what did not land and say why."},
		task{name: "Review the on-call handover", unit: "week", interval: 1,
			desc: "Anything that paged twice becomes a ticket."},
		task{name: "Reconcile the payments ledger", unit: "month", interval: 1,
			desc: "Anything that does not net to zero gets written up."},
	)
}
