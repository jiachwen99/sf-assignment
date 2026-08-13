import type { Blocker, Todo } from '../types'

/*
 * The linking, made visible.
 *
 * A list of names does not show a relationship. This draws the task in the
 * middle of its chain: what it waits on above, what waits on it below, with
 * connectors between. Every node is clickable, so the chain is also how you
 * navigate it.
 *
 * The rule the interface has to teach: only completing a task releases what
 * depends on it. Archiving or deleting a blocker does not. That is why each
 * node shows its state rather than just its name.
 */

const dotFor = (status: Blocker['status']) =>
  status === 'completed' ? 'bg-done' : status === 'archived' ? 'bg-ink-faint' : 'bg-halt'

function Node({
  item,
  satisfied,
  onOpen,
  onUnlink,
}: {
  item: Blocker
  satisfied: boolean
  onOpen: (id: number) => void
  onUnlink?: (id: number) => void
}) {
  return (
    <div className="group flex items-center gap-2.5">
      <span className={`size-1.5 shrink-0 rounded-full ${dotFor(item.status)}`} aria-hidden />
      <button
        type="button"
        onClick={() => onOpen(item.id)}
        className="min-w-0 flex-1 truncate text-left text-[13px] text-ink hover:underline"
      >
        {item.name}
      </button>
      <span className="shrink-0 text-[11px] text-ink-faint">
        {satisfied ? 'done' : item.status === 'archived' ? 'archived' : 'not done'}
      </span>
      {onUnlink && (
        <button
          type="button"
          onClick={() => onUnlink(item.id)}
          aria-label={`Remove dependency on ${item.name}`}
          className="shrink-0 rounded px-1 text-[11px] text-ink-faint opacity-0 transition group-hover:opacity-100 hover:text-late focus-visible:opacity-100"
        >
          unlink
        </button>
      )}
    </div>
  )
}

// The line is what turns a list of names into a chain, so it has to be
// visible enough to read as a connection.
function Connector() {
  return <div className="my-0.5 ml-[3px] h-3 w-0.5 rounded-full bg-rule-firm" aria-hidden />
}

export function DependencyChain({
  task,
  dependencies,
  dependents,
  onOpen,
  onUnlink,
}: {
  task: Todo
  dependencies: Blocker[]
  dependents: Blocker[]
  onOpen: (id: number) => void
  onUnlink: (id: number) => void
}) {
  const blocking = dependencies.filter((d) => d.status !== 'completed')
  const isBlocked = blocking.length > 0

  if (dependencies.length === 0 && dependents.length === 0) {
    return (
      <p className="text-[13px] text-ink-faint" data-testid="dependency-chain">
        Nothing links to this task yet. Add a dependency to make it wait for something else.
      </p>
    )
  }

  return (
    <div className="space-y-2" data-testid="dependency-chain">
      {dependencies.length > 0 && (
        <>
          <p className="text-[11px] font-medium tracking-wide text-ink-soft uppercase">
            Waits for
          </p>
          <div className="space-y-0">
            {dependencies.map((d, i) => (
              <div key={d.id}>
                <Node
                  item={d}
                  satisfied={d.status === 'completed'}
                  onOpen={onOpen}
                  onUnlink={onUnlink}
                />
                {i < dependencies.length - 1 && <Connector />}
              </div>
            ))}
          </div>
          <Connector />
        </>
      )}

      {/* The task itself, so the chain reads through it rather than around it. */}
      <div className="flex items-center gap-2.5 rounded-md bg-sunk px-2 py-1.5">
        <span
          className={`size-1.5 shrink-0 rounded-full ${isBlocked ? 'bg-halt' : 'bg-ink-faint'}`}
          aria-hidden
        />
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-ink">
          {task.name}
        </span>
        <span className="shrink-0 text-[11px] text-ink-faint">this task</span>
      </div>

      {dependents.length > 0 && (
        <>
          <Connector />
          <p className="text-[11px] font-medium tracking-wide text-ink-soft uppercase">
            Waited on by
          </p>
          <div className="space-y-0">
            {dependents.map((d, i) => (
              <div key={d.id}>
                <Node item={d} satisfied={d.status === 'completed'} onOpen={onOpen} />
                {i < dependents.length - 1 && <Connector />}
              </div>
            ))}
          </div>
        </>
      )}

      {isBlocked && (
        <p className="pt-1 text-[12px] text-halt">
          {blocking.length === 1 ? 'This is waiting on one task.' : `This is waiting on ${blocking.length} tasks.`}{' '}
          Archiving or deleting a blocker does not release it, only completing does.
        </p>
      )}
    </div>
  )
}
