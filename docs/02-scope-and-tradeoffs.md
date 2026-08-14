# Scope and trade-offs

The brief says it has more requirements than fit in the time, on purpose. So this document is the other half of the interpretation work: not what each requirement means, but which ones I am building and which ones I am not.

## What I am building

Everything in the core, and all three of the optional features.

**Core.** Task CRUD with validation. Recurrence with the anchoring and clamping rules from `01`. Dependencies with cycle detection and blocking. Soft delete with a Trash view and restore. Filter, sort and keyset pagination that stay in the database. All three non-functional requirements treated as features rather than as prose.

**Optional.** Real-time updates over server-sent events. Authentication as identity. Bulk archive and bulk complete with per-item results.

The optional features come last and in a fixed order, in reverse of the order I would cut them: real-time first, then auth, then bulk operations. Real-time is the strongest thing to show in a demo, so it is the last thing I would give up. Deciding that order now means running short drops whole features cleanly rather than leaving three of them half finished.

## What I did not build

Each of these was considered and rejected. Some are things the brief hints at, some are things a reviewer might expect.

**Per-user task lists.** The first non-functional requirement says users share one list. Adding accounts that partition data would contradict it. Authentication supplies identity and attribution, never separation.

**Full recurrence grammar.** No cron, no iCal RRULE. A unit and an interval covers daily, weekly, monthly and custom, which is everything the brief names. RRULE would have cost more than the entire dependency feature.

**A scheduler that creates occurrences in advance.** The brief's wording is completion-triggered, and a background scheduler is a second process to run, deploy and reason about. It would also make the "no catch-up backlog" rule harder rather than easier.

**Password reset, email verification, refresh tokens.** Authentication is scoped to identity. Each of these is real work that proves nothing the brief asked about.

**Hardening the login endpoint.** What is there is sound as far as it goes: passwords are bcrypt hashed, the session cookie is `HttpOnly` and `SameSite=Lax`, and a wrong password and an unknown email return the same error so the response cannot be used to find out which addresses are registered. Even a missing account compares against a dummy hash, so it does not answer faster.

What is missing is everything that makes those defences hold under pressure. Nothing limits how fast an attacker can guess, and nothing locks an account after repeated failures. bcrypt makes each attempt cost something, which slows an attacker down but is not a control. The cookie sets `Secure` only when the request arrived over TLS, so locally it is absent and behind TLS it appears; setting it unconditionally would mean the cookie is silently dropped over plain HTTP. In front of real users that still needs TLS everywhere and HSTS, neither of which this repository configures. Database credentials are `todo:todo` in the compose file, which is right for a machine you can throw away and wrong everywhere else.

I would build rate limiting first. It is the cheapest of the three and the one whose absence is exploitable rather than merely untidy.

**Reminders and notifications.** A task list that knows a due date and never mentions it again puts the whole job of remembering back on the person. Email or push on something falling due, and on a blocker being completed so the person waiting finds out, is the feature I would add first if this were a product rather than an exercise. It needs a scheduler, a delivery channel and per-user preferences, none of which the brief asks for.

**Sub-tasks.** Dependencies model "this waits for that" between peers. They do not model "this is part of that", which is what people reach for when a task is too big. It is a different shape: a parent whose progress is derived from its children rather than set directly.

**WebSockets.** Updates only flow one way, from server to client. Server-sent events do that with less machinery and they reconnect on their own.

**Multiple server instances.** The live update hub keeps subscribers in memory. Two instances behind a load balancer would not see each other's changes. Postgres LISTEN/NOTIFY is the fix. The brief asks for something that runs locally, so I stated the limit instead of building for a deployment that does not exist.

**Sub-tasks, tags, attachments, comments, reminders, notifications.** Not in the brief. Adding features nobody asked for is the opposite of the prioritisation this exercise is testing.

**Kubernetes or a cloud deployment.** The guideline says it should run easily locally. Docker Compose does that.

**Front-end unit tests.** This is the cut I am least comfortable with, so it gets a paragraph rather than a line. The risk in this interface is concentrated in one path: create a task, block it, hit a conflict, delete and restore it, filter at scale. One end-to-end test walks exactly that path, and it doubles as the demo script. Component tests would have added coverage of code that mostly arranges someone else's components. If I had another day, this is where part of it would go.

**A generic data-access layer.** Four queries carry this project: the dynamic filter with keyset ordering, counter maintenance across other rows, the transactional recurrence spawn, and the cycle walk. None of them can be expressed generically. A `Repository[T]` would have abstracted only the easy cases while adding a second way to reach the database, so there would be two mechanisms to learn instead of one.

## What I would do with more time

Ordered by what I think matters most.

**Per-user timezones.** Everything is UTC, rendered in the browser's locale. A monthly task that crosses a daylight-saving boundary can land an hour off for a user in a different zone from whoever created it. The fix is a timezone on the user record and recurrence arithmetic that respects it. This is the simplification most likely to produce a real bug report.

**Horizontal scale for live updates.** Replace the in-memory hub with Postgres LISTEN/NOTIFY so more than one instance can serve traffic.

**Front-end component tests.** See above.

**A dependency graph view.** The chain in the panel shows one task's immediate neighbours, which is the common case. Seeing a whole cluster at once would need a real graph layout, and that is a different piece of work.

**Multi-column sort.** Currently one sort key plus the ID as a tiebreak. Sorting by priority and then due date is a reasonable thing to want, and the indexes would need to change to support it without a sort step.

**Making the blocking invariant standing rather than checked.** Right now a blocked task cannot enter In Progress or Completed, which means a Completed task can never have unfinished dependencies. I do not also guard transitions out of Completed, so the invariant holds by construction rather than by enforcement. Guarding both directions would make it impossible to violate rather than merely difficult.

## The interface

The brief says the interface does not need to be polished, and this one is not styled to be looked at: system fonts, a neutral palette, no visual identity. What it does spend effort on is making the domain legible, because the hard parts of this project are invisible in a plain table.

The colour system and what yields first when the window is narrow are decided in SF-31, once there is an interface to describe.

## Costs I am accepting

**The scale numbers will be honest, not flattering.** At 10,000 rows, keyset pagination and offset pagination perform about the same. The gap opens up further out, which is why the README will publish numbers at 200,000 as well. I am choosing keyset because its cost does not grow with depth, not because it rescues the experience at the size the brief names.

**Ten indexes on one table is a lot.** `todos` carries a primary key, one per sort key, two partial indexes for blocked and unblocked, one for recurring, a trigram index for substring search and a partial index for live rows. The brief asks for four sort keys and for the list not to degrade at 10,000 rows, and those two together are what pays for them. On a write-heavy table I would not do this.

**Strict dependency rules push work into the interface.** Because archiving or deleting a blocker does not unblock its dependents, a user can back themselves into a task they cannot start. The confirmation dialog and the blocked-reason panel exist to make that visible before it happens rather than confusing afterwards. A softer rule would have needed less interface. I think the strict rule is right, but the cost is real and it is paid in components.
