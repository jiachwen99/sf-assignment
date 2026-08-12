# How I read the brief

The brief says it contains more requirements than can be built in a reasonable timeframe, and that this is intentional. So the first job is not building. It is deciding what each requirement actually means, because several of them have more than one honest reading, and picking the wrong one quietly costs a day.

This document lists every place I had to make that call. Each entry gives the ambiguity, the options I saw, what I picked, and why. Where I departed from the literal text of the brief, I say so.

`docs/00-brief.md` has the requirements transcribed with no interpretation, so you can compare the two.

## The answers, in one table

The reasoning for each is below. This is here so you do not have to read all of
it to find the one you care about.

| The question | What I decided |
|---|---|
| What does "custom" mean? | A unit and an interval. Daily is `(day, 1)`, custom is anything else |
| Does the next occurrence step from the due date or the completion date? | The due date, through a stored anchor |
| A recurring task completed very late? | One occurrence, not a backlog of thirty |
| The next monthly occurrence after 31 January? | 28 February, then back to 31 March. The anchor survives the clamp |
| Reopen an old occurrence and complete it again? | Nothing spawns. The schedule moved to the occurrence it created |
| Does an archived recurring task still produce occurrences? | No. Archiving pauses the series |
| A recurring task with no due date? | The next occurrence has none either. There is nothing to step from |
| Should Completed be blocked as well as In Progress? | Yes. Guarding only In Progress leaves a way round it |
| Does an archived or deleted dependency still block? | Yes. Only completing releases a dependency |
| What happens to dependency links when a task is deleted? | They are kept, which is what makes restore exact |
| Which status transitions are legal? | Anything to Archived. Archived leaves only to Not started. Blocked refuses both In Progress and Completed |
| Can a task depend on itself, or form a cycle? | Refused, and the error names the loop it found |
| Is Archived a status or is it deletion? | A status you set and unset. Delete is separate and soft |
| What does "data should not be permanently lost" require? | Soft delete with links intact, so restore returns the task as it was |
| What does "10,000+ items without degrading" require? | The cost of a page must not grow with depth. Keyset paging, all filtering in the database |
| Filter by name: prefix or substring? | Substring, held back until three characters |
| Filter by due date: exact day or range? | A range, two inputs |
| Sort by priority alphabetically? | No. By meaning, so High, Medium, Low. That is why they are Postgres enums |
| Can two tasks have the same name? | Yes. The keyset cursor breaks ties on id |
| What does "multiple users concurrently" require? | It points at a silent overwrite. Row versions, and a 409 carrying the current state |
| Does authentication contradict the shared list? | Only if accounts partition it. They supply identity, never separation |

---

## Recurrence

### What does "custom" mean?

The brief lists daily, weekly, monthly, or custom. It does not say what custom is.

I read the four as one mechanism: a unit and an interval. Daily is `(day, 1)`. Weekly is `(week, 1)`. Monthly is `(month, 1)`. Custom is any other combination, like every three weeks.

The alternative was to support a real recurrence grammar, either cron or iCal RRULE. Both are the right answer for a calendar product. Neither is the right answer here. RRULE alone would eat more of the week than the entire dependency feature, and nothing in the brief suggests a todo list needs "the third Tuesday of every month".

Collapsing four listed features into one mechanism is the single biggest scope decision in this project.

### Does the next occurrence step from the due date or the completion date?

The brief says the next occurrence is created "based on its schedule". It does not say what the schedule is anchored to.

I step from the previous due date.

Take a monthly bill due on the 1st. I tick it on the 18th because I was busy. Anchored to the due date, the next one is the 1st of next month. Anchored to the completion date, it becomes the 18th, and next month it drifts again. Within a year a monthly task has wandered halfway around the calendar.

Completion-date anchoring is easier to implement and it is what you get if you do not think about it. That is why it is worth calling out.

### What happens when a recurring task is completed very late?

Not addressed by the brief.

A daily task that has not been touched for a month has thirty missed occurrences. If completing it generates all of them, one click produces thirty rows. The user has just created their own performance problem, and the third non-functional requirement is about exactly that.

So I skip missed periods. Completing late produces one occurrence, dated to the next future one. The history of what was missed is not reconstructed, because it was never real.

### What is the next monthly occurrence after 31 January?

Not addressed by the brief, and the obvious implementation is wrong.

Go's `time.AddDate(0, 1, 0)` on 31 January returns 3 March. It adds one to the month, gets 31 February, and normalises the overflow. Nobody wants that.

I clamp to the last valid day of the target month, and keep the original anchor. So 31 January goes to 28 February, and the one after that is 31 March, not 28 March. The anchor survives the clamp.

This has its own test. It is the kind of bug that ships and then surfaces once a year.

### If an old occurrence is reopened, does completing it spawn again?

No. Completing an occurrence hands its schedule to the one it creates, so only
the open occurrence can spawn. Reopening a completed one and finishing it again
records the completion and creates nothing.

The alternative, leaving the schedule on every occurrence, forks the series: two
open tasks with the same name and the same due date, and no way to tell which
one is the real next one.

### Does an archived recurring task still produce occurrences?

No. Archiving is shelving, and a shelved task should not keep generating work.
Completion is refused while archived, so the series pauses until it is
unarchived. Deleting it stops the series the same way.

### What if a recurring task has no due date?

Then the next occurrence has no due date either. There is nothing to step from. It is a slightly odd thing to create, but rejecting it would be inventing a rule the brief does not ask for.

---

## Dependencies

### The brief only blocks "In Progress". Should it also block "Completed"?

The brief says a dependent task cannot be moved to In Progress until all its dependencies are Completed. It says nothing about moving straight to Completed.

Read literally, you can tick a task done while its prerequisites are unfinished. You just have to skip In Progress on the way.

I block both. Guarding one and not the other leaves the rule optional.

This is a deliberate departure from the literal text, and it buys something concrete beyond tidiness. If a blocked task can never be Completed, then a Completed task can never have unfinished dependencies. That makes the whole upstream chain correct by checking only direct dependencies. I never have to walk the graph at read time.

### Does an archived or deleted dependency still block?

Not addressed. It matters, because it decides whether a user can unblock themselves by deleting the thing in their way.

Only Completed satisfies a dependency. Archiving a blocker does not unblock the dependent. Neither does deleting it.

The softer reading is defensible: if I archived it, I am done with it, so let it count. I rejected that because archiving means shelved, not finished, and "delete the blocker to proceed" is a data-loss shaped escape hatch.

Strictness has a cost, and I pay it in the interface rather than softening the rule. Archiving or deleting a task that others depend on shows a confirmation naming exactly which tasks will stay blocked. And any blocked task can tell you why it is blocked, listing each blocker, its current state, and the two ways forward: finish it, or drop the link.

### What happens to dependency links when a task is deleted?

Not addressed, and it decides whether restore actually works.

Links survive deletion. The row is soft deleted, the edges stay untouched. Restore is then just clearing a timestamp, and everything downstream is exactly as it was.

The alternative, cascading the delete to the edges, makes restore a lie. You get the task back without its relationships, and the second non-functional requirement says data should not be permanently lost. Relationships are data.

### Which status transitions are legal?

The brief lists four statuses and constrains exactly one move: a dependent task cannot go to In Progress until its dependencies are Completed. It says nothing about the rest.

**Not Started straight to Completed is allowed.** Forcing a task through In Progress to tick it off is ceremony, and nobody works that way. The dependency rule still applies to both.

**Completed can go back to Not Started or In Progress.** A mis-click has to be recoverable. Reopening a recurring task does not retract the occurrence its completion already created, because that occurrence is a real task somebody may already have edited.

**Archived is outside the active lifecycle.** Anything can be archived, since shelving is not progress and is never blocked. But an archived task cannot go straight to Completed: it returns to Not Started first. Archiving means shelved, not finished, so completing something you had put away is two decisions and the interface asks for both. The panel offers Unarchive rather than Complete on an archived task, so it never presents an action the server will refuse.

### Can a task depend on itself, or form a cycle?

Not addressed. Both are rejected. A cycle means nothing can ever start, which is worse than useless because the interface would show a set of tasks all permanently blocked with no explanation. The error names the path it found, so you can see which link to remove.

---

## Delete and archive

### Is "Archived" a status or is it deletion?

The brief lists Archived as one of four statuses. It also asks for standard CRUD, which includes delete. It never says how the two relate.

I kept them separate. Archived is a status you set and unset freely, like the other three. Delete is a soft delete: the task disappears from every normal view, lives in Trash, and can be restored. The row is never removed.

Mapping delete onto Archived would have been cheaper. I did not, because then the status means two unrelated things at once, and there is no honest answer to what the DELETE endpoint does.

### What does "data should not be permanently lost" require?

This is the second non-functional requirement, and it is one sentence.

I read it as: deleting is recoverable by the user, not just by a DBA with a backup. So delete sets a timestamp, Trash lists what is in there, and restore brings it back with its relationships intact.

The weaker reading is "we take backups". That is true of any database and would make the requirement meaningless.

---

## Finding tasks

### What does "10,000+ items without degrading user experience" require?

This is the third non-functional requirement, and it is the one that shapes the data model.

I read it as: the cost of a page does not grow with how deep you are in the list. That rules out `OFFSET`, which makes the database walk and discard every row before the page you asked for.

So I use keyset pagination. The cursor carries the sort key of the last row, and the next page is an index seek.

I want to be honest about this one. At 10,000 rows, offset is also fast. You would not feel the difference. I chose keyset because its cost does not change as the list grows, not because it rescues a broken experience at the size the brief names. The README publishes measured numbers at 10,000 and at 200,000 so you can see where the gap actually opens up.

Filtering by blocked state is the harder half. Asking "is anything blocking this task" with a subquery cannot share an index with the sort. Filtering to the 50 blocked tasks among 10,000 while sorting by due date would check thousands of rows per page. So each task carries a count of its unfinished dependencies, maintained when it changes. Blocked becomes a plain indexed column that composes with sorting.

That is denormalisation, and the risk is drift. There is a property test that runs a random sequence of dependency changes, completions and reopenings, then checks every counter against a query that computes it from scratch.

### Filter by name: prefix or substring?

Not addressed. The brief lists filters and does not say how name matching works.

I started with prefix matching, because a prefix uses the ordinary index and keeps the promise that a filter, sort and page resolve as one index seek. That was the wrong call for the user: nobody searches by typing the first word of a task name.

It is substring matching now, backed by a trigram index. The honest cost is that name search is the one filter that does not resolve as a single seek. It is a bitmap scan followed by a sort, and I would rather state that than claim a property the query does not have.

Two things keep it cheap. The client waits for three characters before it asks, because shorter terms match most of the table and cost a scan to say nothing useful. And the request is debounced, so typing a word is one query rather than one per keystroke.

### Filter by due date: exact day or range?

The brief says "due date". Filtering to one exact timestamp would match nothing, so I read it as a range, with an overdue preset since that is what people actually want.

Overdue means past due and not finished. Completed and archived tasks are excluded, and tasks with no due date are excluded, because a task with no deadline cannot miss one.

### Sort by priority and status: alphabetically?

Alphabetically, High comes before Low comes before Medium. That is obviously not what anyone means.

Both sort by their real order. Priority runs Low, Medium, High. Status follows the lifecycle: Not Started, In Progress, Completed, Archived. I get this from the database rather than the application, using enum types, which compare by declaration order. It sorts correctly and stays indexable.

### Can two tasks have the same name?

Yes. Identity is the ID. Recurrence produces occurrences with the same name by design, so requiring uniqueness would break the feature.

---

## Concurrency

### What does "multiple users accessing the same TODO list concurrently" require?

This is the first non-functional requirement. Read loosely it says nothing, because any web server handles concurrent requests.

I read it as the thing that actually goes wrong: two people open the same task, both edit, and the second save silently erases the first. Nobody sees an error. The work is just gone.

Every task carries a version. Updates send the version they read. If it no longer matches, the write is rejected and you get the current state back, with an option to reload. Nothing is overwritten silently.

I considered locking the row while someone edits it. That needs lock expiry, heartbeats for browsers that were closed, and a force-unlock path for when it goes wrong. It is a lot of machinery for a list nobody holds open for long.

One thing that falls out of this for free: it also makes completing a recurring task idempotent. Two clicks, or a double-submit, both read the same version, so the first wins and the second is rejected. No duplicate occurrence, and no separate mechanism to make that true.

---

## The optional features

### Does adding authentication contradict the shared list?

The first non-functional requirement says multiple users share one list. The optional features include user accounts. Read carelessly, adding accounts means each user gets their own tasks, which contradicts the requirement.

So authentication is identity only. It never partitions data. Everyone who logs in sees the same list. What accounts buy is attribution: the conflict message can say who changed the task instead of "someone else".

That is the reading that makes both requirements true at once.

---

## What I have not resolved

Every ambiguity above has an answer. These are open in a different sense: I know what I did, and I know it is a simplification.

**Timezones.** Everything is stored in UTC and rendered in your browser's locale. There are no per-user timezones. This is fine until a monthly task crosses a daylight-saving boundary for a user in a different zone from the one who created it. Fixing it properly means a timezone on the user record and recurrence arithmetic that respects it.

**One server.** Live updates are broadcast from memory in a single process. Run two instances behind a load balancer and a client connected to one will not see changes made through the other. The fix is Postgres LISTEN/NOTIFY. I did not build it because the brief asks for something that runs locally, but the limitation is real and I would rather state it than have it found.
