import type { Todo } from '../types'
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

export function TaskList({
  todos,
  activeId,
  onOpen,
}: {
  todos: Todo[]
  activeId: number | null
  onOpen: (todo: Todo) => void
}) {
  return (
    <table className="w-full border-collapse text-left">
      <thead>
        <tr className="border-b border-rule-firm">
          <Th className="pl-4">Task</Th>
          <Th>Due</Th>
          <Th>Status</Th>
          <Th>Priority</Th>
          <Th className="pr-4">Created</Th>
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
