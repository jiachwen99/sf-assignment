# Working with AI

The brief welcomes AI session transcripts as optional supplementary material. A
transcript shows what was typed. This is the more useful version: where I used
it, where I did not, and what it actually changed about the result.

## The division

**I wrote the code and did the scaffolding.** The schema, the queries, the
handlers, the components, the Docker setup — that is the part I am confident in
and the part I wanted to own, because the architectural decisions are made in
the writing, not afterwards.

**I used AI for three things.**

*Restructuring what I had already written.* The front end grew screen by screen
and drifted, the way front ends do: three near-identical button styles, two
control classes differing by a couple of pixels of padding, one icon pasted into
two files. I had it work through the whole tree with that as the brief, and the
result is the `ui/` and `lib/` split described in `09-interface-design.md`.

*Writing documentation from my context rather than from the code.* The
architecture and decision documents needed to say *why*, and why does not live
in the source. I gave it the reasoning and had it write the prose, then checked
what came back against the database and the migrations.

*Running the seed and exercising test cases, especially edge cases.* This is
where it paid for itself, and the rest of this document is about that.

## What it found that would otherwise have shipped

Each of these was live in the repository and would have reached a reviewer.

**The seeder disagreed with the application about "no due date".** The
application writes `'infinity'` into the sort column for a task with no due
date. The seeder wrote `9999-12-31`. Identical on every screen and identical in
every count, and wrong the moment a keyset page compares the two: a task edited
through the interface would sort ahead of every seeded task meant to be its
equal. 5,867 rows. Found by querying the seeded data against the application's
own invariants rather than by looking at it.

**"All dependencies must be complete" was never actually tested.** Every
blocking test in the suite used a single dependency — and with one dependency,
"all are complete" and "any is complete" are the same sentence. An
implementation that cleared the counter on the first completion instead of
decrementing it would have passed the entire suite. The behaviour was correct;
nothing was holding it there.

**The trash returned every row it had.** Every other list in the application is
keyset-paged. That one had no limit, and the front end rendered all of it. It is
also the one table nothing ever prunes.

**Two documents claimed a colour-contrast check that did not exist.** The
sentence described a real technique in convincing detail. There was no such
test. The palette turned out to pass, which was luck. The check exists now.

**The race detector had never once run.** It needs a C toolchain, and without
one it fails open rather than loudly, so every local invocation had been
silently doing nothing.

**Two components silently discarded any class passed to them.** `Badge` and
`Notice` spread their props after their own `className`, so a caller styling
them replaced the component instead of extending it. Invisible until somebody
passes a class.

**Sorting by due date descending puts undated tasks first.** Which follows from
treating "no due date" as infinitely far away, and is defensible — but I had
written the opposite into a test plan from memory an hour earlier.

## What that changed about how I work

The pattern in that list is not that AI is good at finding bugs. It is that
**almost none of them were findable by reading.** They were found by running the
thing and comparing the answer to what the answer should have been. Several were
in my own tests and documentation rather than in the product.

So the habits that came out of it:

**Prove a test can fail before trusting it.** Every check added late in this
project was run once against deliberately broken code to confirm it goes red.
The contrast test carries that permanently, as a sentinel case that must be
caught, so a green tick cannot mean the parser stopped working.

**Check claims against the running application, not the source.** The audit in
`06-audit.md` was done that way on purpose, and it is why it found things. Four
numbers in the architecture document had been true when written and had gone
stale silently — table counts, index counts, statement counts. Reading would not
have caught one of them.

**Make a checkable claim checkable by something other than its author.** The
OpenAPI parity test compares the specification against the registered routes in
both directions and has caught six undocumented endpoints. The contrast test
reads the palette out of the stylesheet rather than from a table somebody has to
remember to update.

## Where it was not useful

It is confidently wrong in a specific way that is worth naming: it will write a
detailed, plausible account of something that does not exist. The contrast check
is the clean example — a whole sentence about a sentinel and a technique, for a
file that had never been created. Twice, it produced a fix for the event-stream
proxy that was reasoned convincingly and did not work, and the second attempt
killed every stream on establishment; the fix was reverted both times and the
limitation documented instead.

It also cannot tell you what matters. It will happily produce fifteen
undifferentiated observations when three of them are worth a reviewer's time.
Deciding which three is the part of this that is mine.

The working rule I settled on: **it is good at doing the work and untrustworthy
about whether the work was done.** Anything it reported, I checked against the
database, the browser, or a test I had watched fail.
