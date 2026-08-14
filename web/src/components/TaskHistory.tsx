import { useTaskEvents } from '../api/todos'
import type { TaskEvent } from '../types'
import { describeDue } from '../lib/dates'

/*
 * The event log, made readable.
 *
 * Every change is already written inside the transaction that made it, which
 * makes this the only place in the interface that can answer "what happened to
 * this task, and who did it". Signing in exists to fill that second line; if
 * nothing ever read the log, attributing changes would be pointless.
 *
 * Each row says what changed rather than naming an event kind, because
 * "status_changed" is the system's word and "Not started to In progress" is the
 * user's.
 */

const labels: Record<string, string> = {
  created: 'Created',
  updated: 'Edited',
  status_changed: 'Status changed',
  completed: 'Completed',
  spawned: 'Created by a recurrence',
  deleted: 'Moved to trash',
  restored: 'Restored from trash',
  dependency_added: 'Dependency added',
  dependency_removed: 'Dependency removed',
}

const statusWords: Record<string, string> = {
  not_started: 'Not started',
  in_progress: 'In progress',
  completed: 'Completed',
  archived: 'Archived',
}

// A recurrence links two tasks, so the log carries the link rather than
// describing it. Either end opens the other.
function linked(event: TaskEvent): { id: number; label: string } | null {
  const p = event.payload ?? {}
  if (event.kind === 'completed' && typeof p.spawned === 'number') {
    // Not lowercased. describeDue returns both "In 3 days" and "Aug 31", and
    // folding the case to fit one sentence turns the month into "aug".
    const due =
      typeof p.spawnedDueDate === 'string' || p.spawnedDueDate === null
        ? describeDue(p.spawnedDueDate as string | null).text
        : null
    return {
      id: p.spawned,
      label: due ? `Created the next occurrence — ${due}` : 'Created the next occurrence',
    }
  }
  if (event.kind === 'spawned' && typeof p.from === 'number') {
    const name = typeof p.fromName === 'string' ? p.fromName : `task ${p.from}`
    return { id: p.from, label: `From ${name}` }
  }
  return null
}

function detail(event: TaskEvent): string | null {
  const p = event.payload ?? {}

  if (event.kind === 'status_changed' || event.kind === 'completed') {
    const from = typeof p.from === 'string' ? statusWords[p.from] : null
    const to = typeof p.to === 'string' ? statusWords[p.to] : null
    return from && to && from !== to ? `${from} to ${to}` : null
  }
  if (event.kind === 'spawned') {
    return typeof p.dueDate === 'string' || p.dueDate === null
      ? `Due ${describeDue(p.dueDate as string | null).text}`
      : null
  }
  if (event.kind === 'dependency_added' || event.kind === 'dependency_removed') {
    // The name was captured when the event was written, so it still reads
    // correctly after the other task is renamed or deleted.
    if (typeof p.dependsOnName === 'string') return p.dependsOnName
    return typeof p.dependsOn === 'number' ? `Task ${p.dependsOn}` : null
  }
  if (event.kind === 'created' && typeof p.name === 'string') return p.name
  return null
}

function when(iso: string, now = new Date()) {
  const then = new Date(iso)
  const mins = Math.round((now.getTime() - then.getTime()) / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  if (mins < 1440) return `${Math.round(mins / 60)}h ago`
  const days = Math.round(mins / 1440)
  if (days === 1) return 'yesterday'
  if (days < 14) return `${days} days ago`
  return then.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function TaskHistory({
  todoId,
  onOpenTask,
}: {
  todoId: number
  onOpenTask: (id: number) => void
}) {
  const { data: events, isPending } = useTaskEvents(todoId)

  if (isPending) {
    return (
      <div className="space-y-2" aria-hidden>
        {[0, 1, 2].map((i) => (
          <div key={i} className="h-3 rounded bg-sunk" style={{ width: `${70 - i * 12}%` }} />
        ))}
      </div>
    )
  }

  if (!events || events.length === 0) {
    return <p className="text-[13px] text-ink-faint">No changes recorded yet.</p>
  }

  return (
    <ol className="space-y-0" data-testid="task-history">
      {events.map((event, i) => {
        const line = detail(event)
        const link = linked(event)
        return (
          <li key={event.id} className="flex gap-2.5">
            {/* A rule joining the dots, so the log reads as a sequence rather
                than a list of unrelated rows. */}
            <div className="flex flex-col items-center pt-1.5">
              <span className="size-1.5 shrink-0 rounded-full bg-rule-firm" aria-hidden />
              {i < events.length - 1 && <span className="w-px flex-1 bg-rule" aria-hidden />}
            </div>

            <div className="min-w-0 flex-1 pb-2.5">
              <div className="flex items-baseline gap-2">
                <span className="text-[13px] text-ink">{labels[event.kind] ?? event.kind}</span>
                <span className="tabular ml-auto shrink-0 text-[11px] text-ink-faint">
                  {when(event.createdAt)}
                </span>
              </div>
              {line && <p className="truncate text-[12px] text-ink-soft">{line}</p>}
              {/* Says nobody rather than saying nothing, because a blank line
                  reads as a rendering fault rather than as an unattributed
                  change. Everything recorded before accounts existed is one. */}
              <p className="text-[11px] text-ink-faint">{event.actor ?? 'Not signed in'}</p>
              {link && (
                <button
                  type="button"
                  onClick={() => onOpenTask(link.id)}
                  data-testid="history-link"
                  className="block max-w-full truncate text-left text-[12px] text-action underline-offset-2 hover:underline"
                >
                  {link.label}
                </button>
              )}
            </div>
          </li>
        )
      })}
    </ol>
  )
}
