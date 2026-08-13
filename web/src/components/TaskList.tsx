import type { SortDir, SortField, Todo } from '../types'
import { describeCreated, describeDue, exactDue } from '../lib/dates'
import { priorityLabel, statusLabel } from '../lib/format'
import { recurrenceLabel } from '../lib/recurrence'
import { Badge } from './ui/Badge'
import { RepeatIcon } from './ui/icons'

/*
 * A table, because the job is scanning a few hundred rows rather than reading.
 *
 * Due dates are relative while that is still useful and absolute past a
 * fortnight, so a list sorted by due date does not read as though every row is
 * identical. Created goes last: it is metadata, not something you act on.
 */

const dueTone: Record<string, string> = {
  none: 'text-ink-faint',
  late: 'text-late',
  today: 'text-ink',
  soon: 'text-ink-soft',
  later: 'text-ink-soft',
}

const rankMark: Record<string, string> = {
  low: 'text-ink-faint',
  medium: 'text-ink-soft',
  high: 'text-ink',
}

// Dates read newest first, names and ranks from the top, so the first click on
// a column gives the order somebody actually wanted.
const firstDirection: Record<SortField, SortDir> = {
  name: 'asc',
  status: 'asc',
  priority: 'asc',
  due: 'asc',
  created: 'desc',
}

export function TaskList({
  todos,
  activeId,
  sort,
  dir,
  onSort,
  onOpen,
}: {
  todos: Todo[]
  activeId: number | null
  sort: SortField
  dir: SortDir
  onSort: (sort: SortField, dir: SortDir) => void
  onOpen: (todo: Todo) => void
}) {
  // Clicking the active column reverses it; clicking another switches to it.
  const sortBy = (field: SortField) =>
    onSort(field, field === sort ? (dir === 'asc' ? 'desc' : 'asc') : firstDirection[field])

  const header = (field: SortField, label: string, className = '') => (
    <Th className={className}>
      <button
        type="button"
        onClick={() => sortBy(field)}
        data-testid={`sort-${field}`}
        aria-sort={field === sort ? (dir === 'asc' ? 'ascending' : 'descending') : 'none'}
        className="inline-flex items-center gap-1 rounded transition-colors hover:text-ink-soft"
      >
        {label}
        {/* The arrow is drawn from the state rather than toggled alongside it,
            so it cannot disagree with the order on screen. */}
        {field === sort && (
          <span aria-hidden className="text-[10px] text-ink-faint">
            {dir === 'asc' ? '▲' : '▼'}
          </span>
        )}
      </button>
    </Th>
  )

  return (
    <table className="w-full border-collapse text-left">
      <thead>
        <tr className="border-b border-rule-firm">
          {header('name', 'Task', 'pl-4')}
          {header('due', 'Due')}
          {header('status', 'Status')}
          {header('priority', 'Priority')}
          {header('created', 'Created', 'pr-4')}
        </tr>
      </thead>
      <tbody>
        {todos.map((todo) => {
          const due = describeDue(todo.dueDate)
          const finished = todo.status === 'completed'
          const repeats = recurrenceLabel(todo)
          return (
            <tr
              key={todo.id}
              onClick={() => onOpen(todo)}
              data-testid="todo-row"
              data-id={todo.id}
              aria-selected={todo.id === activeId}
              className={`cursor-pointer border-b border-rule transition-colors duration-150 ${
                todo.id === activeId ? 'bg-action-wash' : 'hover:bg-raised'
              }`}
            >
              <td className="py-1.5 pl-4">
                <div className="flex min-w-0 items-center gap-2">
                  <span
                    className={`truncate text-[13px] ${finished ? 'text-ink-faint line-through' : 'text-ink'}`}
                  >
                    {todo.name}
                  </span>
                  {/* Blocked comes first: it is the one thing that stops you
                      acting on a row, so it should be read before the name has
                      finished registering. */}
                  {todo.unmetDeps > 0 && (
                    <Badge tone="halt" data-testid="blocked-badge" title="Waiting on unfinished work">
                      Blocked
                    </Badge>
                  )}
                  {/* The cadence is the point, not the fact that an icon is
                      present. An unlabelled glyph makes a reader hover every
                      row to find out what it means. */}
                  {repeats && (
                    <Badge data-testid="repeats-badge">
                      <RepeatIcon />
                      {repeats}
                    </Badge>
                  )}
                </div>
              </td>
              <td className="py-1.5">
                <span className={`text-[13px] ${dueTone[due.state]}`} title={exactDue(todo.dueDate)}>
                  {due.text}
                </span>
              </td>
              <td className="py-1.5 text-[13px] text-ink-soft">{statusLabel(todo.status)}</td>
              <td className="py-1.5">
                <span className={`text-[13px] ${rankMark[todo.priority]}`}>
                  {priorityLabel(todo.priority)}
                </span>
              </td>
              <td className="tabular py-1.5 pr-4 text-[13px] text-ink-faint">
                {describeCreated(todo.createdAt)}
              </td>
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

function Th({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return (
    <th className={`py-2 text-[13px] font-medium text-ink ${className}`}>{children}</th>
  )
}

export function TaskListSkeleton() {
  return (
    <div className="space-y-2 p-4" aria-hidden>
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="h-4 rounded bg-sunk" style={{ width: `${88 - i * 6}%` }} />
      ))}
    </div>
  )
}
