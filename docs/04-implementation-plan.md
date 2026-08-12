# Implementation plan

Written before any code. The commit log executes against it, so you can compare what I planned to what actually shipped. Where they differ, the retrospective in `DECISIONS.md` says why.

Every commit carries its ticket ID, so `git log --oneline` and this document can be read side by side. The log runs past the plan: the audit in SF-017 and the work it found are in there too.

## Tickets

| Ticket | Title | Depends on |
|---|---|---|
| SF-000 | Requirements interpretation, architecture and plan | none |
| SF-001 | Scaffold project with ReactJS, Go, PostgreSQL, Docker | SF-000 |
| SF-002 | Create, view and edit tasks | SF-001 |
| SF-003 | Handle concurrent edits with optimistic locking | SF-002 |
| SF-004 | Recurring tasks | SF-002 |
| SF-005 | Task dependencies and blocking | SF-002 |
| SF-006 | Soft delete and trash recovery | SF-002 |
| SF-007 | Filtering, sorting and pagination at scale | SF-005 |
| SF-008 | Substring search, the views rail and the archived view | SF-007 |
| SF-009 | Change history, event log and the task history panel | SF-005 |
| SF-010 | API documentation | SF-007 |
| SF-011 | Seed data and latency benchmarks | SF-007 |
| SF-012 | End-to-end suite covering the demo path | SF-011 |
| SF-013 | Real-time updates | SF-012 |
| SF-014 | User authentication | SF-013 |
| SF-015 | Bulk operations | SF-014 |
| SF-016 | Interface built around the domain | SF-008 |
| SF-017 | Requirements audit, and what it found | SF-016 |
| SF-018 | Decision log retrospective and README | SF-017 |

Two of these are worth calling out.

**SF-016 is the largest ticket here**, and it is all interface. A table of names is a functional todo list and a useless one: nothing in it shows which tasks are blocked, overdue or recurring, and those are the three hardest things in the codebase. Making the domain legible is the work.

**SF-017 exists because the audit has to be a ticket, not an intention.** Tracing every line of the brief against the running application found two filters with no control in the interface, and an event log that was written but never read. Neither is visible from inside the code, which is the point of checking against the brief instead of against your memory of it.

## How the order was chosen

**Documentation first, and all of it.** Interpretation, architecture, scope, conventions and this plan are knowable before any code exists, and the brief grades requirement interpretation first. Only the audit and the retrospective have to wait, and those are SF-017 and SF-018.

**A walking skeleton second.** SF-002 cuts a thin slice through every layer: migration, store, service, HTTP handler, and a React table. It does almost nothing, but it does it end to end. From that point on there is a running application, and every ticket after it leaves one.

**Features by feature, not by layer.** A ticket that adds recurrence touches the domain package, the store, the service, the API and the interface. It lands as one coherent piece of work. Building the whole store layer, then the whole service layer, would mean nothing runs until the last commit and the log would read like a directory listing.

**Optional features last, in reverse cut order.** Real-time, then auth, then bulk operations. That is the reverse of the order I would give them up, so if the week compresses, whole tickets drop rather than three features ending up half built.

**Pure logic before anything that stores it.** Inside a ticket, the domain functions come first and test-first. The recurrence arithmetic is the highest-risk code here and it needs no infrastructure to test.

## More than fits

The plan is bigger than the week. That is deliberate: the brief says it contains
more requirements than can be built in a reasonable timeframe, so a plan that
appears to fit is a plan that has quietly dropped something.

Writing the order to give things up in, before starting, is the point. The
alternative is discovering the overrun on day five and cutting in a panic.

### Cut ladder

If time runs short, I give in this order. No step leaves a half-built feature.

1. SF-015, then SF-014, then SF-013. The optional features, in their decided order.
2. SF-011 publishes numbers at 10,000 rows only, with the 200,000 method described but not run.
3. SF-010 documents the core routes and omits the optional ones.
4. SF-012 walks three of the demo steps instead of six: recurrence, blocking, conflict.
5. SF-006 ships Trash as an API endpoint and a link, without a dedicated view.

Below that I would cut a feature outright rather than ship several at half quality.

SF-000, SF-017 and SF-018 are never cut. A submission without its decision log fails the brief's own list of deliverables.

## Verification

| Gate | Command |
|---|---|
| Lint | `golangci-lint run` in `api/`, `npm run lint` in `web/` |
| Domain unit tests | `go test ./internal/domain/...` |
| Integration tests | `go test ./internal/store/... ./internal/service/... ./internal/api/...` |
| Counter property test | `go test -run TestCounterInvariant ./internal/store/...` |
| Query plans | `go test -run TestListQueryPlan ./internal/store/...` |
| End to end | `npx playwright test` in `web/` |
| Stack smoke | `docker compose up` from a clean clone |

Integration tests start their own PostgreSQL, so nothing needs to be running first. CI runs every gate on push and seeds 10,000 rows for the end-to-end run. Seeding 200,000 on every push would dominate the pipeline for no extra signal, so that measurement is taken locally.

### Performance targets

Measured by SF-011 and published in the README. At 200,000 rows: p95 under 100ms for the default list, and under 150ms for the worst case, which is the blocked filter combined with a sort.

If either target cannot be met without revisiting the pagination decision, that is worth saying out loud rather than quietly relaxing the number.

## Definition of done

- Every requirement is implemented or listed in `03` with a reason.
- Every gate above is green in CI.
- The README carries measured latency at both sizes with the command that reproduces it.
- `DECISIONS.md` fits two pages and every entry names the alternative it beat.
- The end-to-end test and the written demo script walk the same path, and a narrated pass takes under ten minutes.
- `openapi.yaml` and the running routes agree, enforced by a test.
- No dead code from approaches that did not work out. A week of solo work accumulates experiments and declaring done means removing them.
