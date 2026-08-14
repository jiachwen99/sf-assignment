# Manual test plan

Everything here is done in a browser against a running stack. No code is read
and nothing is run from a terminal except the two commands that start the
application. Each check names the requirement it covers, so a reviewer can stop
at any point and know what has been established.

The automated suites cover the same ground and more; this exists so that a
reviewer who wants to see it themselves does not have to take a green tick on
trust. What the automated suites do *not* cover is in `08-e2e-coverage.md`.

## Starting

```
bun install --cwd web        # once, before the first start
docker compose up --build
```

The interface is on <http://localhost:5173> and the API on
<http://localhost:8080>. The browsable API reference is at
<http://localhost:8080/docs>.

For anything about scale, load the seed first. It takes about a minute:

```
cd api
go run ./cmd/seed -n 20000
```

The seed is deterministic, so the counts below will match. **It truncates first**,
so load it before working through the checks rather than after, or the tasks you
created will be gone. Without it the application is empty and still works; only
checks 14 and 15 need it.

---

## Tasks

**1 · A task can be created with everything the brief lists.**
Press **New task**. Give it a name, a description, a due date two days out,
status *Not started* and priority *High*. Save.
*Expect:* it appears in the list with the due date shown as "In 2 days" and
priority "High". Re-open it and every field you typed is still there.
→ *TODO management: id, name, description, due date, status, priority; create.*

**2 · A task can be edited and the change sticks.**
Open it, change the priority to *Low* and the name, save, then reload the page.
*Expect:* both changes survive the reload.
→ *Update.*

**3 · A task can be deleted and is not lost.**
Delete the task from its panel and confirm.
*Expect:* it leaves the list. Open **Trash** in the header and it is there, with
its description intact. Press **Restore** and it returns to the list whole.
→ *Delete; "data should not be permanently lost when a TODO is deleted".*

## Recurrence

**4 · A repeating task creates its next occurrence when completed.**
Create a task due tomorrow, set **Repeats** to *Weekly*, save. Press **Complete**.
*Expect:* the completed task stays completed, and the panel moves to a new task
of the same name due seven days after the original. The list shows the new one
carrying a **Weekly** badge.
→ *"When a recurring TODO is marked as completed, the next occurrence should be
created automatically."*

**5 · All four cadences are offered.**
Open the **Repeats** control on any task.
*Expect:* daily, weekly, monthly, and a custom option that takes a number and a
unit.
→ *"daily, weekly, monthly, or custom".*

**6 · A monthly task due on the 31st does not skip a month.**
Create a task due on 31 January of next year, set it to repeat monthly, complete
it.
*Expect:* the next occurrence is 28 February, not 3 March.
→ *Month-end clamping. This is the edge case most likely to be wrong; the
underlying rule is tested in `api/internal/domain`.*

## Dependencies

**7 · A task can wait for more than one other task.**
Create three tasks: *confirm the venue*, *finish the deck*, *send the
invitations*. Open *send the invitations*, and in **Links** use the picker to
make it wait for the other two.
*Expect:* both appear under "Waits for", the row gains a **Blocked** badge, and
the panel says it is waiting on 2 tasks.
→ *"A TODO can depend on one or more other TODOs."*

**8 · Blocked means blocked until *all* are done.**
With that task still blocked, try to set its status to *In progress*.
*Expect:* refused, and the message names a specific task to go and finish.
Now complete *confirm the venue* only, and try again.
*Expect:* still refused, and the message now names *finish the deck* instead.
Complete that too.
*Expect:* the badge clears and *In progress* is accepted.
→ *"cannot be moved to 'In Progress' until **all** of its dependencies are
'Completed'." The middle step is the one that distinguishes "all" from "any".*

**9 · A loop is refused and the loop is named.**
Make *confirm the venue* wait for *send the invitations*, which already waits
for it.
*Expect:* refused, with a message naming the cycle in the direction of waiting.
→ *Not in the brief. Left in because a dependency graph that accepts a cycle
produces tasks that can never start.*

**10 · Archiving a blocker does not release what waits on it.**
Block a task, then archive its blocker rather than completing it.
*Expect:* the dependent stays blocked.
→ *"until all of its dependencies are Completed" — archiving is not completing.*

**10a · A completed task can be shown as blocked, and that is deliberate.**
Complete a blocker, then complete the task that waited on it, then reopen the
blocker by setting it back to *In progress*.
*Expect:* the task you completed is still **Completed** and now also carries a
**Blocked** badge, and appears under the **Blocked** view.
→ *Not a contradiction, though it reads like one at first. The count means "how
many of my dependencies are unfinished", which is a fact about the graph and not
about my own status. It has to be maintained this way: if you now reopen that
completed task, it must be refused, and it is. The alternative — clearing the
count for anything already completed — would let a reopened task start with an
unfinished dependency. The cost is that the **Blocked** view answers "has
unfinished dependencies" rather than "is stuck", and includes completed work.*

## Filtering and sorting

**11 · Every filter the brief names.**
Use the filter row: status, priority, the due-date range, and the
blocked/unblocked control, one at a time.
*Expect:* each narrows the list, the count in the rail agrees, and the URL
changes. Copy the URL into a new tab and the same filtered list loads.
→ *"Filter by: status, priority, due date, dependency status."*

**12 · Every sort the brief names.**
Click the **Due**, **Priority**, **Status** and **Task** column headings.
*Expect:* each sorts, an arrow marks the active column, and a second click
reverses it. Under *Due*, tasks with no due date sort last ascending and first
descending: an absent due date is treated as infinitely far away rather than as
an empty value, so it sits at the far end whichever end that is.
→ *"Sort by: due date, priority, status, name."*

**13 · Filters and sorts combine.**
Filter to *High* priority and sort by due date.
*Expect:* both hold at once, and paging further down the list keeps both.

## Scale and concurrency

**14 · The list stays quick with twenty thousand tasks.** *(needs the seed)*
Clear the filters so the full list loads, then scroll to the bottom repeatedly.
*Expect:* each page arrives without a visible pause. The rail shows counts in
the thousands.
→ *"handle a TODO list with 10,000+ items without degrading user experience."
Measured numbers are in `05-performance.md`.*

**15 · Search finds a task inside its name.** *(needs the seed)*
Type a fragment of a word into the name box, not the first word.
*Expect:* matches on the fragment.

**16 · Two people editing the same task cannot overwrite each other.**
Open the same task in two browser windows. Change the priority in the first and
save. Now change the name in the second and save.
*Expect:* the second is refused, told the task changed, and offered the current
version rather than silently losing the edit.
→ *"The API should support multiple users accessing the same TODO list
concurrently."*

**17 · A change in one window appears in the other.**
Keep both windows open on the list. Create a task in one.
*Expect:* it appears in the other within a second or so, without a refresh.
→ *Nice-to-have: real-time updates.*

## Accounts and history

**18 · Signing in attributes what you do.**
Register an account from the header menu, then edit a task.
*Expect:* the task's **History** shows the change against your name. Sign out and
edit again: the new entry is recorded as unattributed rather than as somebody
else.
→ *Nice-to-have: user authentication and registration.*

**19 · History records what happened, not just that something did.**
Open a task that has been edited a few times.
*Expect:* creation, status changes with their before and after, and dependency
links, in order.

## Bulk

**20 · A batch applies what it can and says what it could not.**
Select several tasks with the row checkboxes, including at least one blocked
task, and press **Complete**.
*Expect:* the ones that could be completed are, and the blocked one is listed by
name with the reason, rather than the whole batch failing.
→ *Nice-to-have: bulk operations.*

## Errors

**21 · The API refuses bad input with a reason.**
Try to save a task with an empty name.
*Expect:* refused, with the message attached to the name field rather than a
generic failure.
→ *"Implement error handling and input validation for API requests."*

---

## What this plan deliberately does not check

Anything needing more than a browser: query plans, the counter's behaviour under
concurrent writes, and the latency figures. Those are covered by the Go suite and
by `05-performance.md`, and a reviewer following this plan should treat them as
claims to be checked there rather than here.
