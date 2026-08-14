# TODO list

A shared task list with recurring tasks, dependencies between tasks, filtering
and sorting that hold up at scale, and real-time updates. Go API, React front
end, PostgreSQL.

Built for the SleekFlow software engineer project. The reasoning behind the
choices is in [`DECISIONS.md`](DECISIONS.md); this file is how to run it.

![The list](docs/images/01-list.png)

## Running it

Requires Docker and [Bun](https://bun.sh).

```bash
bun install --cwd web      # once, before the first start
docker compose up --build
```

| | |
|---|---|
| Interface | <http://localhost:5173> |
| API | <http://localhost:8080> |
| API reference | <http://localhost:8080/docs> |
| Database | `postgres://todo:todo@localhost:5432/todo` |

The install comes first on a clean clone, and the order matters. The web
container mounts a named volume at `/app/node_modules`, which sits inside the
`./web` bind mount, so its mount point is the host's `web/node_modules`. If
Docker creates that directory it does so as root, and the next install fails
with `EACCES`.

Migrations run at startup, so the database is ready when the API answers. The
application works on an empty database; the seed below only matters for
questions about scale.

**If the interface stops loading data after a long session**, restart the web
container: `docker compose restart web`. Compose runs the Vite dev server, and
its proxy does not pass a closed browser tab through to the API, so each page
load leaves an event-stream connection open. After a few hundred of them the dev
server runs out of upstream sockets and stops proxying, while the API behind it
stays healthy. It is a dev-server limitation rather than an application one, and
`docs/06` has the measurements.

### Seeding

```bash
cd api
DATABASE_URL=postgres://todo:todo@localhost:5432/todo?sslmode=disable \
  go run ./cmd/seed -n 20000
```

Fixed random seed, so the same data comes back every time. `-n 200000` takes
about 26 seconds and is what the latency numbers below were measured against.

## Testing it

```bash
cd api && go test ./...        # 92 tests; needs Docker for the database containers
cd web && bun run test:e2e     # 16 tests; needs the stack running
```

The Go suite runs against a real PostgreSQL in a container rather than a mock,
because the blocking rule, the counter, the keyset cursor and the soft delete are
all enforced in SQL. CI runs both on every push, plus `go vet`, `gofmt`, a
TypeScript build, and the Go suite under `-race`.

To check it by hand instead, [`docs/07`](docs/07-manual-test-plan.md) is a
browser-only checklist that names the requirement each step covers. What the
suites do **not** cover is in [`docs/08`](docs/08-e2e-coverage.md), which is the
more useful half.

## What it does

**Tasks** carry a name, description, due date, status (not started, in progress,
completed, archived) and priority. Deleting is soft: the row stays, and Trash
restores it whole.

**Recurring tasks** repeat daily, weekly, monthly or on a custom interval.
Completing one creates the next occurrence and hands over the schedule, stepping
from the previous due date rather than from when you ticked it — so a bill due on
the 1st stays due on the 1st. A monthly task due on the 31st clamps to the end of
February rather than overflowing into March.

**Dependencies** can be many per task. A task cannot start until *all* of its
dependencies are completed, and the refusal names which one to go and finish.
Cycles are refused with the loop spelled out.

**Filtering and sorting** by everything the brief names — status, priority, due
date range, blocked or unblocked; due date, priority, status, name — and the
whole list state lives in the URL, so a filtered view can be shared or reloaded.

![A blocked task, open](docs/images/02-detail-blocked.png)

Also built: accounts and sign-in, so changes are attributed in each task's
history; real-time updates between browsers over server-sent events; bulk
complete and archive that report per item rather than failing whole; and an event
log behind every task.

## Measured latency

At **200,000 tasks**, over HTTP, 50 requests per query, page size 50:

| Query | p50 | p95 | max |
|---|---|---|---|
| Default list | 1.0ms | 1.6ms | 2.2ms |
| Blocked, sorted by due date | 1.6ms | 3.0ms | 3.0ms |
| Name search | 4.4ms | 6.0ms | 7.5ms |
| Counts | 31.1ms | 42.7ms | 43.8ms |

Reproduce with:

```bash
cd api
go run ./cmd/seed -n 200000
go run ./cmd/bench
```

Paged queries cost the same at 20,000 rows and 200,000, because a keyset page
reads fifty index entries whatever sits behind them. The counts query is the
exception and the ceiling: it scans, so it grows with the table, 5ms at 20,000
and 31ms at 200,000.

These are single-client numbers on one machine with a warm cache. That is the
right shape for asking whether the query plans hold at size and says nothing
about behaviour under concurrent load. Conditions, the full table, and where each
measurement stops being true are in [`docs/05`](docs/05-performance.md).

## Architecture

Four layers with dependencies pointing one way: `api/` decodes and validates,
`service/` owns transactions, `domain/` holds pure rules and imports nothing,
`store/` holds all the SQL.

![Architecture](docs/images/architecture.png)

Full diagrams and the data model are in
[`ARCHITECTURE.md`](ARCHITECTURE.md).

## The documents

| | |
|---|---|
| [`DECISIONS.md`](DECISIONS.md) | The decision log the brief asks for. Two pages. |
| [`ARCHITECTURE.md`](ARCHITECTURE.md) | Layers, data model, indexes, the transaction worth walking through |
| [`docs/00`](docs/00-brief.md) | The brief, kept alongside so the audit has something to trace against |
| [`docs/01`](docs/01-requirements-interpretation.md) | Every ambiguity, and how it was resolved |
| [`docs/02`](docs/02-scope-and-tradeoffs.md) | What was cut, and why |
| [`docs/03`](docs/03-conventions.md) | Naming, error shapes, commit format |
| [`docs/04`](docs/04-implementation-plan.md) | The ticket breakdown `git log` follows |
| [`docs/05`](docs/05-performance.md) | Latency, conditions, and the limits of both |
| [`docs/06`](docs/06-audit.md) | The brief traced against the running application, and what that found |
| [`docs/07`](docs/07-manual-test-plan.md) | A browser-only checklist |
| [`docs/08`](docs/08-e2e-coverage.md) | What the tests cover, and what they do not |
| [`docs/09`](docs/09-interface-design.md) | Tokens, type, components, contrast |
| [`docs/10`](docs/10-working-with-ai.md) | Where AI was used, and what it caught |

API reference: <http://localhost:8080/docs> when running, or
[`openapi.yaml`](api/internal/api/openapi.yaml) in the repository. The reference
page loads Swagger UI from a CDN, so it needs network access; the specification
itself is served from the binary and does not.
