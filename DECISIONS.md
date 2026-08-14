# Decision log

The four questions the brief asks, answered directly. Longer versions live in
`docs/`: [ARCHITECTURE.md](ARCHITECTURE.md) for components,
[docs/01](docs/01-requirements-interpretation.md) for the ambiguities,
[docs/05](docs/05-performance.md) for measurements and query plans.

## 1. Ambiguities, and how they were resolved

- **One shared list, not one list per user.**

  The brief says "multiple users accessing the same TODO list", so I took it
  literally: no owner column, every account sees the same tasks. Concurrency is
  therefore a lost-update problem, solved by versioning rather than identity.

- **Archived and deleted are separate things.**

  Archived is a status the user sets and can unset. Deleting sets `deleted_at`
  and hides the row from normal queries, keeping it in Trash. If deleting
  simply archived the task, "data should not be permanently lost" could not be
  met.

- **Custom recurrence is a unit plus an interval.**

  Daily, weekly and monthly are `(day, 1)`, `(week, 1)` and `(month, 1)`.
  Custom is any other pair, such as every three weeks. Full iCal RRULE would
  have cost more than the dependency feature.

- **Recurrence counts from the original date, not from the last one.**

  Occurrence *n* is the anchor plus *n* intervals, clamped each time: 31
  January, 28 February, 31 March. Counting from the previous
  occurrence gives 28 March and never returns to the 31st. The step comes from
  the due date, not from when you tick it.

- **Blocked tasks cannot be completed either, not just started.**

  The brief only blocks the move to In Progress. Read literally, you could move
  a blocked task straight to Completed and skip the rule, so the check covers
  both transitions.

- **Deleting a blocker leaves its dependents blocked.**

  Deleting a task is not finishing it. The dependency rows survive, so
  restoring the blocker restores the chain. A deleted blocker is still listed
  and marked, so nothing is blocked by something invisible.

## 2. Architectural decisions and trade-offs

- **Go, PostgreSQL and React.**

  Go for the API: its standard library covers HTTP, routing and testing without
  a framework. PostgreSQL because the dependency graph needs recursive queries,
  and partial indexes and transactions carry the design. React because the
  interface is one page of list state.

- **Optimistic concurrency instead of locking.**

  Every update sends the version it read: `UPDATE ... WHERE id = ? AND
  version = ?`. No rows updated means someone else was first, and the API
  returns `409` with the current task. Locking would need expiry, heartbeats
  and a force-unlock path.

- **The count of unfinished dependencies is stored on the row.**

  A subquery per row cannot use the sort's index, so blocked tasks sorted by
  due date scan thousands of rows a page. It is updated in the same transaction
  as the change, and a property test recounts it from the dependency table.

- **Keyset pagination, and a sort column that is never null.**

  Pages use `WHERE (due_sort, id) > (?, ?)` instead of `OFFSET`, so page 400
  costs the same as page 1. The comparison is NULL when the due date is, so
  undated tasks vanish from every page. That looks correct ascending and breaks
  only descending, so `due_sort` is `NOT NULL`.

- **Accounts identify people; they do not separate data.**

  Sign-in checks the password with bcrypt, then stores a 256-bit random token
  in `sessions` with a seven-day expiry, returned in an `HttpOnly`,
  `SameSite=Lax` cookie. An opaque session, not a JWT: signing out revokes it
  immediately, which a self-contained token cannot. Accounts add attribution.

- **Server-sent events, published after the transaction commits.**

  Updates travel only from server to client, so WebSockets add a protocol
  upgrade for nothing. Messages say what changed and the client refetches
  normally, so the stream is never a second way to write. Subscribers live in
  memory: one instance only.

- **Bulk actions run one task at a time.**

  Each item carries its own version and transaction, so a blocked or stale item
  fails alone and the rest go through. One transaction around the batch would
  discard forty-nine good writes for one bad row.

## 3. What was not built, and why

- **Per-user lists.** They contradict "the same TODO list", and would leave the
  concurrency requirement with nothing to do.
- **Rate limiting, password reset and roles.** Accounts are scoped to identity,
  and an in-process rate limiter would only cover one instance.
- **Tags, subtasks, comments and attachments.** Not in the brief. Each needs its
  own table, endpoints and interface, and none of them exercises anything the
  brief is actually testing.
- **Optimistic interface updates.** Every write refetches instead of patching
  the cache, which would make the cache a second source of truth that can
  disagree with the server. The cost is a visible round-trip on save.
- **Front-end unit tests.** The cut I am least happy with. The risky logic is in
  the backend and 14 end-to-end tests cover the interface, but component tests
  would fail faster.

## 4. What would be done differently with more time

- **Give tasks an owner.**

  Accounts record who changed what, but no task knows whose it is, so a shared
  list cannot answer "what is on my plate". An assignee and a filter, not
  per-user lists: everyone still sees one list.

- **Tell people when something is due.**

  The application tracks due dates, counts what is overdue, and already pushes
  every change to every open tab, but it never tells anyone anything. A task due
  tomorrow is seen only by someone who goes looking for it. Reminders would ride
  the stream that already exists.

- **Give recurrence an end.**

  A repeating task repeats forever: there is no until-date and no occurrence
  count. The brief did not ask for one, and it is the first thing anybody would
  want.

- **Decide what "due today" means.**

  Dates are `timestamptz` and rendered in the browser's zone, which is
  internally consistent. But the list is shared, so two people in different
  zones disagree about which tasks are overdue. A shared list needs one agreed
  zone, or a per-user one applied wherever dates are compared.
