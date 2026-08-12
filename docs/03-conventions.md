# Conventions

Written before the code, so these are commitments rather than a description of whatever happened.

## Go layout

```
api/
  cmd/server/main.go        wiring, top to bottom, no magic
  cmd/seed/main.go          seeder, takes a row count
  internal/domain/          pure: no database, no HTTP, no context
    todo.go                 Todo, Status, Priority
    recurrence.go           next-due arithmetic
    transition.go           legal status moves
    errors.go               the closed error set
  internal/store/           all SQL, and nowhere else
    store.go                pool and a transaction helper
    todos.go                CRUD
    list.go                 the dynamic filter, sort and keyset query
    deps.go                 edges and counter maintenance
    events.go
  internal/service/         transactions and orchestration
  internal/api/             HTTP, thin
    server.go               routes and middleware
    errors.go               the only place a status code is chosen
    decode.go               decode and validate in one step
    openapi.yaml            hand-written, embedded, checked against the routes
  internal/events/hub.go
  migrations/
```

Packages are named for what they own. There is no `models/`, `utils/`, `helpers/` or `pkg/`. Something with no obvious home usually means I have not understood it yet, and giving it a vague folder hides that.

## Go rules

**No interfaces until there are two implementations.** `service` takes a concrete `*store.Store`. An interface with one implementation is a claim about flexibility the code does not have, and it makes every call one jump harder to follow.

**No `Clock` interface.** Pure functions take `now time.Time` as an argument. Testing the 31 January clamp is three lines with no mock and no setup.

**No mocking the database.** Integration tests run against real PostgreSQL, started by the test itself. The interesting bugs in this project live in the SQL: the keyset comparison, the partial indexes, the counter arithmetic. A mock passes happily while all three are broken.

**Errors are a small closed set, mapped once.** `ErrNotFound`, `ConflictError`, `BlockedError`, `ValidationError`. They are mapped to status codes in one file at the HTTP edge. No wrapping an error at every layer on the way up, which produces messages like "failed to handle request: failed to complete task: failed to update todo: no rows" and tells you nothing you could not see from the stack.

**SQL is a string constant next to the function that runs it.** No builder, no DSL. The one exception is `list.go`, where the filters are genuinely dynamic and a small local helper assembles conditions and arguments.

**Generics only where they remove duplication that already exists.** Three places: `decode[T]` for request bodies, `Page[T]` as the response envelope, and pgx's own row mapping. Two similar functions cost less than one abstraction that fits neither.

**Names are short in short scopes.** `t` for a todo inside a five-line function, not `todoItem`. No `Get` prefix on methods: `store.Todo(ctx, id)`, not `store.GetTodoByID(ctx, id)`, which stutters.

## Comments

Comment the parts that matter and skip the rest. Most functions need nothing, because the name and the signature already say it. A comment explains why, never what, and it stays short. No doc comment that restates a function name.

There are five places in this codebase where a reader would otherwise think the code is wrong, and those get a comment:

1. The month-end clamp in the recurrence arithmetic, because `time.AddDate` looks like it should work.
2. The four sites that maintain `unmet_deps_count`, and specifically why delete is not one of them.
3. Why `due_sort` exists at all, since it looks redundant next to `due_date`.
4. Why the event is published after the transaction commits rather than inside it.
5. The update hub only working for a single instance.

If a comment is not on that list, the better question is whether the code is unclear.

## Front end

```
web/src/
  main.tsx
  App.tsx
  api/
    client.ts               fetch wrapper and error parsing
    todos.ts                typed calls and their query hooks, same file
  components/
    TodoTable.tsx
    FilterBar.tsx
    TodoPanel.tsx
    DependencyPicker.tsx
    BlockedReason.tsx
    ConflictBanner.tsx
    ConfirmDialog.tsx
    TrashView.tsx
    BulkBar.tsx
  hooks/useEvents.ts
  lib/format.ts
  styles.css
```

Organised by type, not by feature. There is one feature. A `features/todos/` folder would be a folder pretending to be architecture.

API calls live in the same file as the hooks that wrap them, because they always change together.

TanStack Query is used because the update stream needs a cache to invalidate. Writing one by hand would waste an afternoon.

All list state lives in the URL: filters, sort, cursor, selection.

Mantine supplies the primitives and the default theme is used unmodified. Components compose Mantine rather than wrapping it. A house component layer on top of a component library is the abstraction this project least needs.

No `React.FC`. No `useCallback` or `useMemo` without a measured reason. Props are typed inline for single-use components rather than given a named interface above every one. No barrel files.

## Tests

`domain/` gets table tests with real cases and no infrastructure. This is where I would look first if I were reviewing this, so it is written to be read.

`store/`, `service/` and `api/` get integration tests against real PostgreSQL, with one helper that hands over a clean database.

One property test, for the `unmet_deps_count` invariant. It applies a random sequence of dependency and status changes and checks every counter against a query that recomputes it from scratch.

The query plans are asserted in tests, not checked by hand. An index that silently stops being used is exactly the regression that would otherwise ship.

No front-end unit tests. One end-to-end run covers the demo path. This is a deliberate cut and it is argued in `03`.

## Writing

These documents are read by a person, not a parser, and every sentence in them has to survive a follow-up question in the demo.

- No em-dashes. A full stop or a comma. If a sentence needs an em-dash it is two sentences.
- Short sentences, one idea each.
- Say "I" for decisions. This is solo work.
- Lead with the decision, then the reason.
- No throat-clearing. No "It is worth noting", "Additionally", "Furthermore", "In conclusion".
- No filler adjectives. No "robust", "seamless", "comprehensive", "leverage".
- No hedging. "This may potentially improve performance" says nothing. Say what it does, or say it was not measured.
- Admit gaps plainly. "I did not test past 200,000 rows" is stronger than silence, and a better answer in the demo than being caught out.
- Bullets for lists, prose for reasoning. A decision explained in bullets reads like notes rather than thinking.

## Commits

One ticket per feature, prefixed with its ID: `[SF-004]: Add next-due date arithmetic`. A ticket usually spans two to six commits, and documentation edits land in the same ticket as the code that earned them rather than in a cleanup pass at the end.

Commits are feature-shaped, not layer-shaped. "Add store layer" tells a reader nothing. "Spawn next occurrence on completion" tells them everything.
