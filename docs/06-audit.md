# Requirements audit

Written after the features were finished and before the log was written, against
the running application rather than against the code. The rule was that reading
a function does not count as checking it: every row below was exercised through
the HTTP API or the browser, and where a row cites a test, the test was made to
fail on purpose to confirm it was capable of failing.

Run against a database of 20,359 live tasks and 121 in the trash.

## The brief, traced

| Requirement | How it was checked | Result |
| --- | --- | --- |
| Id, name, description, due date, status, priority | Created and read back over HTTP; `description` edited through the API and through the panel | All six present and writable |
| Create, read, update, delete | All four exercised over HTTP | Pass |
| Four statuses including Archived | Filtered the live list by each | Pass |
| Priority low / medium / high | Filtered the live list by each | Pass |
| Recur daily, weekly, monthly, custom | Completed one of each and read the next occurrence's due date | All four; the 3-week custom landed on day 168, exactly 8 periods from the anchor |
| Completing a recurring task creates the next occurrence | Response names the spawned task; history shows it | Pass |
| A task can depend on one or more others | Built a diamond and a three-deep chain | Pass |
| Cannot start until **all** dependencies are completed | 1-of-2 complete → 409 naming the one still open; 2-of-2 → 200 | Pass, and see finding 1 |
| Filter by status, priority, due date, blocked/unblocked | Each filter run against the live list | All four |
| Sort by due date, priority, status, name | Each sort run in both directions | All four |
| Web UI: create, edit, delete, filter, sort | Driven in a browser | Pass |
| Multiple users on one list concurrently | Two accounts against one list; optimistic locking returns 409 on a stale write | Pass, and see finding 3 |
| Data not permanently lost on delete | Deleted a task, confirmed the row and its body survive in the database, absent from the list, present in the trash, restored with every field intact | Pass |
| 10,000+ items without degrading the experience | Measured at 200,000 rows in `05-performance.md`; keyset paging means a page costs the same at any size | Pass, with the counts query named as the ceiling |
| Error handling and input validation | Nine malformed requests: empty name, unknown status, unknown priority, zero interval, interval without a unit, unknown field, malformed JSON, non-numeric id, unknown id | Correct status and a field-level message on each |
| Tests for core functionality | Unit, integration against a real Postgres, a property test, and an end-to-end suite | Pass |
| Runs locally | `docker compose up` from a clean checkout | Pass |
| Nice-to-have: auth, real-time, bulk, Docker and CI, architecture diagram | All five built | Pass |

## What the audit found

**1. "All dependencies" was never actually tested.** Every blocking test used a
single dependency, where "all are complete" and "any is complete" are the same
sentence. `TestADiamondIsAllowed` builds two dependencies but completes neither.
An implementation that cleared the counter on the first completion instead of
decrementing it would have passed the entire suite. The behaviour turned out to
be correct — verified over HTTP before touching anything — but nothing was
holding it there. Added `TestATaskWithTwoDependenciesNeedsBothCompleted`, which
walks a task from two blockers to one to none.

**2. The trash was the one unbounded list.** Every other list is keyset-paged;
`listTrash` had no `LIMIT`, ignored the one it was given, and the front end
fetched the whole array into a single cache entry and rendered all of it. A
trashed task costs about 330 bytes on the wire, so a trash holding as many rows
as the performance document loads into the table would be a single response of
roughly 65MB and as many DOM nodes. It is also the one table nothing ever prunes,
so it only ever grows. Capped at
the hundred most recently deleted, which is the end you go there for. The header
count now comes from the counts query rather than from the length of what
arrived, so it still reports the truth, and the view says plainly when there is
more behind it. `TestTheTrashIsBounded` covers it, and fails without the cap.

**3. Concurrent writes were believed correct, not known to be.** The counter is
maintained by application code inside a transaction, and every test for it was
sequential, so two people completing two different blockers of the same task at
the same moment was untested. `TestConcurrentCompletionsLeaveTheCounterCorrect`
releases ten goroutines at once against one dependent and asserts the count ends
at zero. It passes: the row locks serialise it as expected. The claim is now
tested rather than assumed.

**4. The race detector had never run.** `-race` needs cgo and a C toolchain, and
on the machine this was built on it fails open rather than loudly — every local
invocation was silently a no-op. The event hub is real concurrent code and
nothing was checking it. CI now runs `go test -race ./...`, which is the only
place it actually executes.

**5. Three gaps in the deliverables**, all in the documents rather than the
application, all carried into the log:

- The README the brief asks for as deliverable 2 did not exist.
- The decision log covered three of its four required topics; "what you would do
  differently with more time" was missing.
- The decision log was already 1,146 words, about 2.3 pages against a stated
  limit of one to two, so the missing section had to be paid for by cutting
  rather than added on top.

**6. A closed browser tab leaks its event subscription, and it affects the stack
that ships.** This was known and documented in the events package as a
development-proxy caveat. Measuring it turned one sentence of that into
something more specific, and one of the sentences it turned out to contradict was
mine.

A connection straight to the API releases: subscribers 28 → 29 on open, back to
28 within three seconds of the client going away, which is `r.Context().Done()`
firing as intended and is what end-to-end test 9 asserts. Through the Vite
development proxy it never releases — three tabs opened and closed left three
permanent subscribers, and waiting past the twenty-second heartbeat did not
recover them. The heartbeat cannot help, because the API is writing to the proxy,
and the proxy's socket is perfectly healthy; the write succeeds, so nothing ever
fails. One tab costs one subscriber until the API restarts.

The part I had wrong: I had recorded this as not mattering outside development.
But `docker compose up` runs `bun run dev`, so the development server *is* the
shipped stack, and there is no production serving path in this repository to be
unaffected. The leak is in the environment a reviewer will actually run.

Two fixes were tried and both failed, so the proxy was left as it was. With the
proxy instrumented, `res.close` and `res.socket.close` never fire at all, and
`req.close` fires as soon as the request headers are through — destroying the
upstream socket there kills the stream on establishment, and the subscriber count
never even reaches one. That is the same dead end SF-013 found, now with the
event trace to say why rather than a recollection. The honest fix is
architectural: serve the built assets so no proxy sits in the path.

**7. One smaller thing.** A comment explaining that delete carries a version had
drifted twenty lines from `DeleteTodo` and sat above the search query instead.
Moved back.

## What this audit did not cover

Load and contention: every measurement in this project is single-client, which is
the right shape for asking whether the query plans hold at size and says nothing
about behaviour under concurrent load. Accessibility beyond colour contrast.
Security beyond the authentication path. Response bodies are checked against the
specification by hand rather than by a schema test, so a renamed field would pass
the parity check.
