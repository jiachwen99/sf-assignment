# What the tests cover, and what they do not

108 automated tests: 92 in Go and 16 under Playwright. The counts below are the
number of test functions, and they add up to that total. This document is
mostly about the second half of its title, because a list of everything that
passes tells a reviewer very little.

## The shape of it

| Suite | Tests | What it runs against |
| --- | ---: | --- |
| `internal/domain` | 7 | Pure functions. No database, no HTTP. |
| `internal/store` | 70 | A real PostgreSQL in a container, one per package, truncated between tests. |
| `internal/service` | 7 | The same, one layer up: the bulk paths, per item. |
| `internal/api` | 3 | The specification against the routes, and the reference page. |
| `internal/events` | 5 | The in-process hub, with no database at all. |
| `e2e/demo.spec.ts` | 14 | A browser against the whole stack, on seeded data. |
| `e2e/contrast.spec.ts` | 2 | Arithmetic over `styles.css`. No browser. |

The weight sits in `store` because that is where the decisions are. The blocking
rule, the counter, the keyset cursor and the soft delete are all enforced in SQL
inside a transaction, and a test that mocked the database would be testing the
mock.

## What each group defends

**`domain`** — the recurrence arithmetic, and the status transition rules. The 31
January clamp lives here: `AddDate` normalises overflow, so a naive month step
turns 31 January into 3 March. Testing it costs three lines because the package
imports nothing.

**`store`** — by file: dependencies and cycles (14), users and sessions (11),
tasks (10), listing, filtering and paging (9), the event log (8), trash (7),
completion and recurrence (6), counts (3), and one property test and one
concurrency test standing on their own.

Two of those are worth naming. The property test drives a random sequence of
completions and reopenings across a random dependency graph and asserts the
denormalised `unmet_deps_count` still matches a recount from the edge table; it
exists because that counter is maintained by hand at four separate sites. The
concurrency test releases ten goroutines at once against ten blockers of one
task and asserts the count lands at zero, because everything else in the file is
sequential and would not notice a lost update.

**`service`** — the bulk paths, which are the only place a single request runs a
sequence of independent transactions: one stale item not discarding the rest, a
blocker and its dependent both completing when the order allows it, the same
pair refused the other way round, and results coming back in the order given.

**`api`** — a parity test comparing `openapi.yaml` against the registered routes
in both directions, which has caught six undocumented endpoints, each the moment
it was added; plus two holding the reference page together. The reference page
loads Swagger UI from a CDN, so one of those tests fixes the exact version and
requires an integrity hash and `crossorigin` on both assets: a floating major
version cannot be integrity checked, and an integrity hash without `crossorigin`
is silently ignored.

**`events`** — every subscriber receiving a change, release removing a subscriber
and closing its channel, a slow subscriber not stalling the publisher, and
concurrent subscribe against publish.

**`demo.spec.ts`** — the path a demo actually walks, in a browser, against
seeded data rather than a clean table: create and edit and delete, trash and
restore, recurrence, blocking, a refused cycle, a version conflict between two
windows, filtering and sorting and paging at scale, real-time between two
browsers, subscription release, two accounts and attribution, and bulk.

One of those is not part of the demo. A save awaits the write before closing the
panel, and anything done in that gap races it; opening another task meant the
close landed on the wrong one. The suite found it by failing about one run in
three, and the test now holds the response open until the second task is on
screen, so it fails every time rather than sometimes.

**`contrast.spec.ts`** — reads the OKLCH tokens out of `styles.css`, converts
them the way a browser does, and holds every text pair to WCAG AA. See
`09-interface-design.md` for why this was added late.

---

## What none of it covers

This is the part worth reading.

**Load.** Every measurement in this project is one client at a time. The
concurrency test proves the counter is *correct* under simultaneous writes; it
says nothing about throughput, connection pool contention, or lock waits under
real traffic. `05-performance.md` states the same limit about the latency
figures.

**Response schemas.** The parity test compares paths and methods. A field
renamed in Go without being renamed in `openapi.yaml` passes it. The schemas were
checked by hand against live responses, which is not the same as a guarantee.

**More than one process.** Subscribers live in memory, so two API instances
behind a load balancer would not see each other's changes. Nothing tests that,
because nothing in this repository runs two instances.

**The browser beyond one engine and one size.** Chromium at 1440×900. No
Firefox, no WebKit, no mobile viewport. The layout has a breakpoint at `xl` that
only one side of is ever exercised.

**Accessibility beyond colour.** Contrast is now checked. Keyboard traps, focus
management in the panel and the confirmation dialog, and screen-reader labelling
of the dependency chain are not.

**The reference page offline.** `/docs` pulls Swagger UI from unpkg, so it needs
network access to render even though everything it documents is local. The
specification itself is served from the binary at `/openapi.yaml` and does not.

**Anything after a failure.** There is no test for the API being down, the
database refusing a connection, or a request timing out mid-write. The interface
has an error state for a failed list load and nothing exercises it.

**The seeder.** It has no tests. It is checked by running it and querying the
result against the application's own invariants, which is how it was found
writing `9999-12-31` into a column the application fills with `infinity`.

**The race detector, locally.** It needs cgo and a C toolchain, and on a machine
without one the flag fails open rather than loudly. It runs in CI and only there.

## How to run them

```
cd api && go test ./...          # needs Docker for the database containers
cd web && bun run test:e2e       # needs the stack up; see 07-manual-test-plan.md
```

CI runs both on every push, plus `go vet`, `gofmt` and a TypeScript build. The
Go suite runs there under `-race`, which is the only place the race detector
actually executes.
