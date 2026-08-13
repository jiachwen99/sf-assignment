import { countFor } from '../lib/views'
import type { Counts, ViewId } from '../types'

/*
 * Views, not switches. "What is overdue" and "what is stuck" are the questions
 * anyone asks of a task list, so they are places you can go with a live total
 * rather than filters you have to know to set.
 *
 * Two kinds, kept apart because they answer different questions. The four
 * statuses partition the list and add up to the total above them, which a
 * reader can check by eye. Overdue, blocked and recurring cut across those
 * statuses and overlap each other, so they add up to nothing. Mixed into one
 * flat list they invite the arithmetic and then look broken.
 */

type Item = { id: ViewId; label: string; hint: string }

const all: Item = { id: 'all', label: 'All tasks', hint: 'Everything except deleted tasks' }

const statuses: Item[] = [
  { id: 'not_started', label: 'Not started', hint: 'Written down, not picked up' },
  { id: 'in_progress', label: 'In progress', hint: 'Being worked on' },
  { id: 'completed', label: 'Completed', hint: 'Done' },
  { id: 'archived', label: 'Archived', hint: 'Shelved without finishing' },
]

const across: Item[] = [
  {
    id: 'overdue',
    label: 'Overdue',
    hint: 'Past due and not finished. Also counted in a status above',
  },
  { id: 'blocked', label: 'Blocked', hint: 'Waiting on unfinished work. Can also be overdue' },
  {
    id: 'recurring',
    label: 'Recurring',
    hint: 'On a schedule. Opens the next occurrence when completed',
  },
]

const tone: Partial<Record<ViewId, string>> = {
  overdue: 'text-late',
  blocked: 'text-halt',
}

export function ViewRail({
  active,
  counts,
  onSelect,
  orientation = 'vertical',
}: {
  active: ViewId
  counts?: Counts
  onSelect: (view: ViewId) => void
  // Below the breakpoint the same views become a scrollable strip. Hiding the
  // rail without a replacement would leave overdue and recurring reachable only
  // by editing the URL, which is not a missing view but a missing control.
  orientation?: 'vertical' | 'horizontal'
}) {
  const horizontal = orientation === 'horizontal'

  const button = (view: Item) => {
    const isActive = view.id === active
    const count = countFor(counts, view.id)
    return (
      <button
        key={view.id}
        type="button"
        onClick={() => onSelect(view.id)}
        title={view.hint}
        aria-current={isActive ? 'page' : undefined}
        data-testid={horizontal ? `view-${view.id}-compact` : `view-${view.id}`}
        className={`flex items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-[13px] transition-colors ${
          horizontal ? 'shrink-0 border border-rule' : 'w-full justify-between'
        } ${
          isActive
            ? 'bg-sunk font-medium text-ink shadow-[inset_2px_0_0_var(--color-action)]'
            : 'text-ink-soft hover:bg-sunk hover:text-ink'
        }`}
      >
        <span className="truncate">{view.label}</span>
        {/* In full, never abbreviated. "20k" beside "1,944" reads as a
            different kind of number. */}
        {count !== undefined && (
          <span
            className={`tabular shrink-0 text-[11px] ${
              isActive ? 'text-ink-soft' : (tone[view.id] ?? 'text-ink-faint')
            }`}
          >
            {count.toLocaleString()}
          </span>
        )}
      </button>
    )
  }

  if (horizontal) {
    return (
      <nav
        aria-label="Views"
        className="flex gap-1 overflow-x-auto border-b border-rule px-3 py-2 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        {[all, ...statuses, ...across].map(button)}
      </nav>
    )
  }

  return (
    <nav aria-label="Views" className="flex flex-col p-2">
      {button(all)}
      <Group label="Status" note="Every task has exactly one, so these add up to all tasks">
        {statuses.map(button)}
      </Group>
      <Group
        label="Across statuses"
        note="A task can be in several of these at once, so they do not add up"
      >
        {across.map(button)}
      </Group>
    </nav>
  )
}

function Group({
  label,
  note,
  children,
}: {
  label: string
  note: string
  children: React.ReactNode
}) {
  return (
    <div className="mt-3">
      <p
        title={note}
        className="mb-1 px-2.5 text-[10px] font-medium tracking-wide text-ink-faint uppercase"
      >
        {label}
      </p>
      <div className="flex flex-col gap-0.5">{children}</div>
    </div>
  )
}
