# Decisions

Depth lives in `docs/`, linked where it matters. Opened here and appended to by
the ticket that makes each decision, rather than reconstructed at the end.

## Where the brief was ambiguous

The brief says it has more requirements than fit in the time, on purpose. So the
first work was deciding what each one means. Full walkthrough in
[`docs/01`](docs/01-requirements-interpretation.md). The ones that changed the
build most:

**"Daily, weekly, monthly, or custom" is one mechanism, not four.** A unit and an
interval. Daily is `(day, 1)`, custom is anything else. Cron and iCal RRULE were
disproportionate: RRULE alone would have cost more than the whole dependency
feature.

**Recurrence steps from the previous due date, not the completion date.** A bill
due on the 1st that I tick on the 18th is still due on the 1st next month.
Anchoring to completion moves it to the 18th and drifts again every month after.
That is what you get if you do not think about it, which is why it is worth
naming.

**Blocked tasks are refused Completed as well as In Progress.** The brief only
guards In Progress, so read literally you can tick a task done by skipping it. A
gate you can walk around is not a gate. It also pays for itself: if a blocked task
can never be Completed, checking direct dependencies makes the whole upstream
chain correct without walking the graph.

**A series has one live occurrence.** Completing one hands its schedule to the
occurrence it creates. Without that handover, reopening a completed occurrence
and finishing it again forks the series into two tasks with the same name and
date. Both ends of the link are recorded and clickable, so "did this recur, and
into what" is answerable from the task. Archiving pauses a series rather than
ending it, because a shelved task should not keep generating work.

**Archived and deleted are different things.** Archived is a status you set and
unset. Delete is soft: hidden everywhere, recoverable from Trash, row never
removed. Mapping delete onto Archived was cheaper, but then the status means two
unrelated things and there is no honest answer to what `DELETE` does.

Also argued in `docs/01`: Not Started can go straight to Completed but Archived
cannot, completing late produces one occurrence rather than thirty, and only
completing a task unblocks its dependents.

## Architecture

Go, React, PostgreSQL. Four layers, dependencies pointing one way. Diagrams in
[`ARCHITECTURE.md`](ARCHITECTURE.md).

**Raw SQL over an ORM.** Five tables, about fifty statements. `RowToStructByName`
removes the boilerplate that usually justifies an ORM, and the list query has to
be hand-written anyway, because that is where the index lives. An ORM would have
meant two mechanisms instead of one.

**A stored sort column, because nullable keys break keyset pagination.** The
cursor comparison `(due_date, id)` is NULL when the due date is, so undated tasks
vanish from every page. It hides, too: nulls sort last ascending, so it looks
correct until someone sorts descending. `due_sort` is `NOT NULL`, the due date or
`infinity`. It could not be generated: those must be immutable and the cast is
only stable.

**Blocked is a stored count, not a subquery.** Computed at query time it cannot
share an index with the sort key, so filtering blocked while sorting by due date
scans thousands of rows per page. Derived state drifts, which is what the
property test exists to catch.

**Row versions, not locks.** Two people edit the same task and the second save
silently erases the first. That is the failure the concurrency requirement points
at. Updates carry the version they read and are rejected with the current state if
it moved. Locking needs expiry, heartbeats and a force-unlock path, which is a lot
of machinery for a list nobody holds open. It also makes completion idempotent: a
double click reads one version, so the second write loses.

**Name search is substring, and the one query that is not a single seek.** A
leading wildcard cannot use a btree index, so this needs a trigram index and
resolves as a bitmap scan plus a sort. Prefix matching avoids that and is the
wrong trade: nobody searches by typing the first word of a task name. The client
holds the query until three characters, keeping the least selective searches off
the database.

**Keyset pagination, with an honest caveat.** At 10,000 rows offset is also fast
and you would not feel the difference. Keyset was chosen because its cost does
not grow with depth, not because it rescued anything at the size the brief names.
Measured numbers at both sizes are in the README.

**Postgres enums for status and priority**, because they compare by declaration
order, so sorting returns High, Medium, Low with no `CASE`. A `CASE` in the
`ORDER BY` would have stopped the index serving the sort.

**Authentication is identity, never partitioning.** The concurrency requirement
says users share one list; the optional features add accounts. Read carelessly
those contradict. Accounts supply attribution, so the conflict message and the
history name who, instead of saying "someone else".

**The rail separates counts that add up from counts that cannot.** Every task has
one status, so Not started, In progress, Completed and Archived sum to the total
and a reader can check it. Overdue, blocked and recurring cut across those and
overlap each other, so they sum to nothing. In one flat list the two look alike,
someone adds them up, and the interface looks wrong. Two captioned groups, with
trash outside both. A test holds the first group to its claim.

**The interface encodes domain state rather than decorating it.** System fonts, a
neutral palette, nothing styled to be looked at. Three things have to be legible
from a row: a task you cannot start, a date you have missed, and a task that
comes back, so colour is spent on those and the primary action. The dependency
chain is the one piece of real interface work, because a list of names does not
show a relationship.

## What I did not build

Reasoning in [`docs/02`](docs/02-scope-and-tradeoffs.md).

Per-user lists, because they contradict the shared list. A recurrence grammar,
because a unit and interval covers everything the brief names. A background
scheduler, because the brief's wording is completion-triggered. Password reset,
because auth is scoped to identity. WebSockets, because updates flow one way.
Multiple instances, because the update hub is in-process.

**A generic data-access layer.** Four queries carry this project: the dynamic
filter with keyset ordering, counter maintenance across other rows, the
transactional recurrence spawn, and the cycle walk. None is expressible
generically, so a `Repository[T]` would have abstracted the easy cases and added
a second way to reach the database.

**Front-end unit tests.** The cut I am least comfortable with. Risk concentrates
in one path and 28 end-to-end tests walk it, but component tests would fail
faster and point straight at the cause.
